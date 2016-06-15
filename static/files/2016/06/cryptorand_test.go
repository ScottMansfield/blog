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
