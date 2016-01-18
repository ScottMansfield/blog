+++
title = "Locking in crypto/rand"
slug = "locking-in-crypto-rand"
date = "2016-01-10T19:55:30-08:00"
categories = ["Go", "Performance"]
keywords = ["go", "golang", "random", "rng", "math/rand", "crypto/rand"]

+++

As a followup to my [previous post]({{< relref "the-hidden-dangers-of-default-rand.md" >}})
detailing my journey through some profiling of the `math/rand` package, I wanted to write about the
`crypto/rand` package. A couple people have suggested that I take a look at that instead of worrying
about locking in `math/rand`. On the surface, it's an easy interface that fills a byte slice full of
cryptographically secure random data. I modified the `rand_default.go` program from the previous
post to create a [new program](/files/2016/01/crypto_rand_default.go) to pull data from
`crypto/rand` instead of `math/rand`. The full text of the program is below:

```go
package main

import "fmt"
import "crypto/rand"
import "os"
import "runtime/pprof"
import "sync"
import "time"

func randData(n int) []byte {
    b := make([]byte, n)
    rand.Read(b)

    for i := range b {
        b[i] = byte('A') + (b[i] % 26)
    }

    return b
}

func toInt(b [4]byte) int {
    i := int(b[0])
    i |= (int(b[1]) << 8 )
    i |= (int(b[2]) << 16)
    i |= (int(b[3]) << 24)

    return i
}

const numRuns = 10

func main() {
    f, err := os.Create("crypto_rand_default.prof")
    if err != nil {
        panic(err.Error())
    }

    pprof.StartCPUProfile(f)
    defer pprof.StopCPUProfile()

    start := time.Now()
    for i := 0; i < numRuns; i++ {
        do()
    }
    total := time.Since(start)
    perRun := float64(total) / numRuns

    fmt.Printf("Time per run: %fns\n", perRun)
}

const numRoutines = 10

func do() {
    start := make(chan struct{})
    comm := make(chan []byte)

    var read, write sync.WaitGroup
    read.Add(numRoutines)
    write.Add(numRoutines)

    for i := 0; i < numRoutines; i++ {
        go func() {
            var r [4]byte

            <-start
            for j := 1; j < 10000; j++ {
                _, err := rand.Read(r[:])
                if err != nil {
                    panic(err.Error())
                }
                comm <- randData(toInt(r) % 10000)
            }
            write.Done()
        }()

        go func() {
            var sum int
            <-start
            for c := range comm {
                sum += len(c)
            }
            fmt.Println(sum)
            read.Done()
        }()
    }

    close(start)
    write.Wait()
    close(comm)
    read.Wait()
}
```

The difference between this program and the original is the use of `crypto/rand` in the `randData`
function, and the addition of a `toInt()` function that made my life a bit easier when using byte
slices to get randomness. Running the program gave an average time per run of 25.72 seconds, which
is almost 2x the original program at 13.17 seconds.

```
$ go build crypto_rand_default.go
$ ./crypto_rand_default
(output elided)
Time per run: 25717479513.700001ns
```

Below is the graph generated with the go `pprof` tool. Profiling stops the program 100 times a
second to analyze call stacks to determine timing and the call graph of the program empirically,
rather than through source analysis. It also will tell how much time was spent in a particular
function and cumulatively in its reachable nodes in the call graph.


![crypto_rand_default](/img/2016/01/crypto_rand_default.svg)
[link to image](/img/2016/01/crypto_rand_default_orig.svg)

This is very interesting, now. The vast, vast majority of the time (97.88% in fact) is spent making
syscalls. If we take a look at the [Go source code](https://golang.org/src/crypto/rand/rand_linux.go)
we can see that a call to `crypto/rand.Read()` will (on linux) call the `getrandom(2)` syscall to
read random data. Nowhere in the Go code does it seem like there's anything that should stop us from
executing as fast as we can, other than the overhead of performing a syscall, which involves some
context switching and some buffer copying.

## A Bit of Benchmarking

A reader on the previous post (h/t to kune18) wanted to compare the speed of output of the different
methods of generating random numbers in raw data per second. He wrote up a [playground](https://play
.golang.org/p/BlUK6SijCA) to demonstrate a reader around the `math/rand` package. His benchmarks
showed 431 MB/s from `math/rand` and 21 MB/s from `crypto/rand`. Unfortunately, the playground post
wasn't complete enough to reproduce, so I created [my own program](/files/2016/01/rand_speed.go) to
do just that. I got similar results, with 416 MB/s and 13 MB/s respectively. This means a single
unfettered `math/rand.Rand` instance is 31.28x faster than using `crypto/rand.Read()`.

```
$ go build rand_speed.go && ./rand_speed
4361945120 bytes read in 10 seconds from math/rand
415.9875030517578 MB/s
139436064 bytes read in 10 seconds from crypto/rand
13.297659301757813 MB/s
```

## One Level Deeper

I wanted to understand what was happening at a deeper level than just "the syscalls are taking a
long time." I tried to use `strace` to find the calls to `getrandom()` but unfortunately it appears
that the strace version that I had my hands on was not aware of it at all, and would give me a
`syscall_318` instead. Even this was superficial, however, since the tool only showed me what I
already knew.

`perf_events` is a tool much like the `runtime/pprof` package provides. It allows a user to sample
a program's stacks at a specified frequency to determine where it is spending most of its time. It
can do a whole lot more than that, however this was the use I needed. For anyone interested in
learning more about `perf_events`, Brendan Gregg has a [very good introduction](http://www.brendangr
egg.com/perf.html) to the tool.

In order to use `perf_events`, you must have it installed on your system.

```
$ sudo apt-get install linux-tools-common linux-tools-generic linux-tools-`uname -r`
```

To prevent any possible conflict, I removed the code in the test program that performed CPU
profiling from within Go. Below is the run used to collect data for analysis of the final program:

```bash
$ go build crypto_rand_default.go
$ sudo perf record -F 99 ./crypto_rand_default # sudo needed here to access kernel symbols
(output elided)
Time per run: 25257292731.000000ns
[ perf record: Woken up 8 times to write data ]
[ perf record: Captured and wrote 1.707 MB perf.data (44503 samples) ]
```

The `perf report` command brings up a screen in the terminal that reports the time spent in each
function in the program and in the kernel that it can report. It is ordered in time spent in the
function.

```
$ sudo perf report
```

```
Samples: 44K of event 'cpu-clock', Event count (approx.): 452545450020
Overhead  Command          Shared Object        Symbol
  50.41%  crypto_rand_def  [kernel.kallsyms]    [k] _raw_spin_unlock_irqrestore
  39.55%  crypto_rand_def  [kernel.kallsyms]    [k] extract_buf
   3.70%  crypto_rand_def  [kernel.kallsyms]    [k] __memset
   2.19%  crypto_rand_def  [kernel.kallsyms]    [k] copy_user_generic_string
   1.48%  crypto_rand_def  crypto_rand_default  [.] main.randData
   1.01%  crypto_rand_def  [kernel.kallsyms]    [k] memzero_explicit
   0.39%  crypto_rand_def  [kernel.kallsyms]    [k] finish_task_switch
   0.28%  crypto_rand_def  [kernel.kallsyms]    [k] extract_entropy_user
   0.14%  crypto_rand_def  [kernel.kallsyms]    [k] sha_init
   0.12%  crypto_rand_def  [kernel.kallsyms]    [k] _copy_to_user
   0.12%  crypto_rand_def  [kernel.kallsyms]    [k] _raw_spin_lock_irqsave
   0.06%  crypto_rand_def  [kernel.kallsyms]    [k] __do_softirq
   0.06%  crypto_rand_def  [kernel.kallsyms]    [k] retint_careful
   0.04%  crypto_rand_def  crypto_rand_default  [.] runtime.memclr
   0.03%  crypto_rand_def  crypto_rand_default  [.] runtime.mallocgc
   0.03%  crypto_rand_def  crypto_rand_default  [.] runtime.cas
   0.02%  crypto_rand_def  crypto_rand_default  [.] runtime.xchg
   0.02%  crypto_rand_def  crypto_rand_default  [.] runtime.mSpan_Sweep
   0.02%  crypto_rand_def  crypto_rand_default  [.] syscall.Syscall
   0.01%  crypto_rand_def  crypto_rand_default  [.] main.do.func1
...
```

Now we're getting somewhere. It seems most of the time is spent in the `_raw_spin_unlock_irqrestore`
and `extract_buf` functions in the kernel. What do those do? Well, [extract_buf](https://git.kernel.
org/cgit/linux/kernel/git/torvalds/linux.git/tree/drivers/char/random.c?id=7fdec82af6a9e190e53d07a14
63d2a9ac49a8750#n1090) turns out to be the function that does the heavy lifting in the code that
backs `/dev/random/`, `/dev/urandom/`, and the `getrandom(2)` syscall.

Around the single cryptographically-secure random number generator (CSRNG) core in the kernel is a
[lock](https://git.kernel.org/cgit/linux/kernel/git/torvalds/linux.git/tree/drivers/char/random.c?id
=7fdec82af6a9e190e53d07a1463d2a9ac49a8750#n1113) to prevent parallel requests from modifying state
at the same time. As well, it disables interrupts while work is done. The lock itself is a spinlock,
which is a special kind of lock used is special kinds of situations. The biggest time sink was the
function `_raw_spin_unlock_irqrestore`. This is the architecture dependent function that unlocks a
spinlock. The spinlocks themselves are an interesting piece of the kernel, and are heavily
architecture dependent, so I won't get into them here. Needless to say, [the spinlock
implementation](https://git.kernel.org/cgit/linux/kernel/git/torvalds/linux.git/tree/arch/x86/includ
e/asm/spinlock.h) is fun to read.

Oddly, the unlocking phase takes the most amount of time, but my guess here is that the interrupts
that were stored up while the RNG code was doing work all got serviced immediately after the lock
restored interrupts, and thus the unlock seemingly takes a long time but actually is just
interrupted repeatedly.

## Another Lock Revealed

Again the details below the surface thwart the programmer who wishes to obtain random data. It
makes sense, as before, that there is a lock around the single RNG that is producing numbers. Even
with the high-performance implementation of spinlocks, there is no getting away from the
serialization of requests for data. As well, it's helpful to be mindful of the implications of a
syscall. Even when the request itself is small, the cost of switching into the kernel can be large.

I hope that these couple articles have demystified the `math/rand` and `crypto/rand` packages for
you. If you have any questions, please leave a comment. I'd be happy to hear if it helped.

## Bonus Section: Which Should You Use?

The first version of this article was missing an important point, which is why there are two
different random number sources in the first place in Go. The key difference is in the suitability
of the output of each.

`math/rand` is good if you need some random numbers to be used for testing or to ensure that a part
of the program behaves differently each time. For example, the popular [go-fuzz](https://github.com/
dvyukov/go-fuzz) package uses `math/rand` to generate randomness for fuzz testing. Most use cases
can be satisfied by using `math/rand`.

`crypto/rand` is a different beast entirely. The data returned is guaranteed to be cryptographically
secure, and is usable for security purposes such as generating nonces or encryption keys. The speed
of the call is relevant but is a cost of securely generated numbers, and should not be bypassed for
speed reasons. Please don't ever use `math/rand` for something that needs to be secure.

Thanks to Tim Kagle in the Gophers slack and [dchapes](https://www.reddit.com/user/dchapes) over on
reddit for calling me out for omitting this discussion.

