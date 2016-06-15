+++
title = "Managing Syscall Overhead with crypto/rand"
slug = "managing-syscall-overhead-with-crypto-rand"
date = "2016-06-10T00:12:35-07:00"
categories = ["Go"]
keywords = ["Go", "golang", "crypto", "rand", "crypto/rand"]
+++

The overhead of using secure random numbers can be a headache if the generation of those numbers is
in your server's critical path. In this post, I'll look at a couple of techniques to bypass the
overhead of generating random numbers in a Go program and make a recommendation on what method to
use.

Consider an application that needs to generate a nonce for each request which is also I/O bound,
meaning it does more waiting on I/O than anything else. As well, the general requirements include
being sensitive to latency through the system. This system generates the nonce by making a call to 
[`crypto/rand.Read`](https://golang.org/pkg/crypto/rand/#Read) and filling a 16 byte array (not a
slice). All "external calls" are modeled by a single `runtime.Gosched()` to simulate a call without
taking too much time.

In reality a networked program will have many opportunities to schedule a background goroutine in
the course of serving a request. There's many function calls, channel sends, and other preempt
points that allow it to run in the background. This means that pure benchmarks are not going to do a
real system justice, but they will give a good comparison of overhead between different methods.

## Baseline

In order to know that one way is better, we have to have a baseline to compare it to. Call it the
naive solution. The baseline solution generates the nonce as it's needed and immediately "makes a
call" to another service.

```
package main_test

import (
	"crypto/rand"
	"runtime"
	"testing"
)

func BenchmarkBaseline(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		var b [16]byte
		for pb.Next() {
			rand.Read(b[:])
			runtime.Gosched()
		}
	})
}
```

## Solution 1: Buffered Channel and Generator Goroutine

This is a solution I employ in [rend](https://github.com/netflix/rend), a memcached proxy and server
that I created at Netflix. On program start, there is an init function that spins off a goroutine
whose sole purpose is to read 16 byte arrays of random data from `crypto/rand` and send them into a
buffered channel. 

The main purpose of this method is to avoid a blocking [`getrandom(2)`](http://man7.org/linux/man-pa
ges/man2/getrandom.2.html) syscall in the hot path of a request, replacing it instead with a call to
a channel receive. In the majority of cases, this should follow an optimistic path and receive a
value much quicker than a syscall. For the truly brave and curious, the channel receive code is the
`chanrecv` function in [this file](https://golang.org/src/runtime/chan.go).

```
var c chan [16]byte

func init() {
	c = make(chan [16]byte, 1000)
	go func(c chan [16]byte) {
		for {
			var b [16]byte
			rand.Read(b[:])
			c <- b
		}
	}(c)
}

func BenchmarkChannel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			<-c
			runtime.Gosched()
		}
	})
}
```

The advantages to this scheme are threefold:

1. When the server is busy serving a request, the data is almost always immediately available
1. When the server is not busy serving requests, it can take time to do syscalls and get random data
1. Simple interface

## Solution 2: Amortize

The second attempted solution is to amortize the cost of the syscalls by asking for a larger chunk
of data each time and then using that data over several calls. In this case, I am assuming a scheme
where separate goroutines will each pull data on their own and manage it internally.

```
func BenchmarkAmortized(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		var b [256]byte
		var count int64
		for pb.Next() {
			cur := count & 0xF
			count++
			if cur == 0 {
				rand.Read(b[:])
			}
			start, end := cur*16, (cur+1)*16
			_ = b[start:end]
			runtime.Gosched()
		}
	})
}
```

According to the [`getrandom(2)` man page](http://man7.org/linux/man-pages/man2/getrandom.2.html)
the maximum data that can be retrieved at once without blocking in the kernel is 256 bytes. This
maximum is used to attempt to amortize the cost over the most requests.

## Solution 3: Amortize and use buffered channel

Of course, if each of these independently *could* actually increase the performance of our program,
we should try them at the same time. This method combines the previous two methods by using a
separate goroutine to manage reading data and writing to a channel, but instead of reading only 16
bytes at a time it reads 256 and splits it into 16 byte chunks.

```
var ac chan [16]byte

func init() {
	ac = make(chan [16]byte, 1000)
	var b [256]byte
	var count int64
	go func(ac chan [16]byte) {
		for {
			cur := count & 0xF
			count++
			if cur == 0 {
				rand.Read(b[:])
			}
			start, end := cur*16, (cur+1)*16
			var out [16]byte
			copy(out[:], b[start:end])
			ac <- out
		}
	}(ac)
}

func BenchmarkChannelAmortized(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			<-ac
			runtime.Gosched()
		}
	})
}
```

## Solution 4: Round-robin Channels

This is a new idea on top of the channel idea above. Instead of having only one channel, have many.
Each goroutine using the random data can run through each channel in a round-robin fashion,
hopefully avoiding conflict with other consumers of random data. This also implies that each channel
has an independent producer that is the only writer to that channel. The premise fo this approach is
that it wil reduce lock contention on the channels.

```
var crr []chan [16]byte

func init() {
	crr = make([]chan [16]byte, 32)
	for i := range crr {
		c = make(chan [16]byte, 1000)

		go func(c chan [16]byte) {
			for {
				var b [16]byte
				rand.Read(b[:])
				c <- b
			}
		}(c)

		crr[i] = c
	}
}

func BenchmarkChannelRoundRobin(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		var count int
		for pb.Next() {
			count++
			cur := count % len(crr)
			<-crr[cur]
			runtime.Gosched()
		}
	})
}
```

## Solution 5: Many Writers

Reader suggestion time. Luna Duclos pointed out that it may have been the generator goroutine that
was falling behind and not the channel lock contention getting in the way. To test this, I
created one more test case where there is a single channel with 16 writers instead of a single
writer. This is very similar to the initial channel test except for the loop in the `init()` func.

```
var cmw chan [16]byte

func init() {
	cmw = make(chan [16]byte, 1000)
	for i := 0; i < 32; i++ {
		go func(cmw chan [16]byte) {
			for {
				var b [16]byte
				rand.Read(b[:])
				cmw <- b
			}
		}(cmw)
	}
}

func BenchmarkChannelManyWriters(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			<-cmw
			runtime.Gosched()
		}
	})
}
```

## Solution 6: Amortized Over a Channel with Many Writers

This combines all of the above strategies into one. There's many writers all writing into one
channel, each of which is amortizing the overhead of the syscalls.

```
var acmw chan [16]byte

func init() {
	acmw = make(chan [16]byte, 1000)

	for i := 0; i < 32; i++ {
		go func(acmw chan [16]byte) {
			var b [256]byte
			var count int64
			for {
				cur := count & 0xF
				count++
				if cur == 0 {
					rand.Read(b[:])
				}
				start, end := cur*16, (cur+1)*16
				var out [16]byte
				copy(out[:], b[start:end])
				acmw <- out
			}
		}(acmw)
	}
}

func BenchmarkChannelAmortizedManyWriters(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			<-acmw
			runtime.Gosched()
		}
	})
}
```

## Results

The benchmarks above were all run 1000 times and summarized using [benchstat](https://godoc.org/rsc.
io/benchstat). The Go source code can be found [here](/files/2016/06/cryptorand_test.go) and the
raw output data from my test runs is [here](/files/2016/06/results).

```
$ go test -bench . -count 1000 -timeout 24h 2>&1 | tee -a results
$ benchstat results
name                           time/op
Baseline-2                     1.12µs ± 9%
Channel-2                      1.82µs ±10%
Amortized-2                     777ns ± 7%
ChannelAmortized-2             1.29µs ± 7%
ChannelRoundRobin-2            1.31µs ± 8%
ChannelManyWriters-2           1.29µs ± 8%
ChannelAmortizedManyWriters-2   932ns ± 8%


```

The amortized approach is the clear winner here. The solution in rend, at the time of this writing,
can serve to show you want a case of premature optimization looks like. I thought it would be faster
to use a channel, but it turns out that it is a little bit slower overall. In reality, the program
is not used at a high enough volume for the differences shown here to matter, but it's nice to know
that I, like many others, have discovered a place where I can learn from past mistakes.

Hopefully, if you're using `crypto/rand` to generate nonces in your application, you can use one of
the above methods to keep the syscall overhead out of your critical path. If you do use one of the
above methods or know of another one that has been useful to you, please leave a comment below or
message me on twitter [@sgmansfield](https://twitter.com/sgmansfield).  
