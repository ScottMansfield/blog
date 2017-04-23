package bucket

import (
	"testing"
	"testing/quick"
)

const numBuckets = 276

var powerOf4Index = []int{
	0, 3, 14, 23, 32, 41, 50, 59, 68, 77, 86, 95, 104,
	113, 122, 131, 140, 149, 158, 167, 176, 185, 194,
	203, 212, 221, 230, 239, 248, 257, 266, 275,
}

func getBucket(n uint64) uint64 {
	if n <= 15 {
		return n
	}

	rshift := 64 - lzcnt(n) - 1
	lshift := rshift

	if lshift&1 == 1 {
		lshift--
	}

	prevPowerOf4 := (n >> rshift) << lshift
	delta := prevPowerOf4 / 3
	offset := int((n - prevPowerOf4) / delta)
	pos := offset + powerOf4Index[lshift/2]

	if pos >= numBuckets-1 {
		return numBuckets - 1
	}

	return uint64(pos + 1)
}

//go:noinline
func getBucketNoInline(n uint64) uint64 {
	if n <= 15 {
		return n
	}

	rshift := 64 - lzcnt(n) - 1
	lshift := rshift

	if lshift&1 == 1 {
		lshift--
	}

	prevPowerOf4 := (n >> rshift) << lshift
	delta := prevPowerOf4 / 3
	offset := int((n - prevPowerOf4) / delta)
	pos := offset + powerOf4Index[lshift/2]

	if pos >= numBuckets-1 {
		return numBuckets - 1
	}

	return uint64(pos + 1)
}

var Sink uint64

func BenchmarkGetBucket(b *testing.B) {
	for i := uint64(0); i < uint64(b.N); i++ {
		Sink = getBucket(i)
	}
}

func BenchmarkGetBucketNoInline(b *testing.B) {
	for i := uint64(0); i < uint64(b.N); i++ {
		Sink = getBucketNoInline(i)
	}
}

func BenchmarkGetBucketASM(b *testing.B) {
	for i := uint64(0); i < uint64(b.N); i++ {
		Sink = getBucketASM(i)
	}
}

func TestSpecialFailingCase(t *testing.T) {
	i := uint64(0xdf5b0412ffd341c0)
	// 1101 1111 0101 1011 0000 0100 0001 0010
	// 1111 1111 1101 0011 0100 0001 1100 0000
	t.Logf("Trying input %d", i)

	a := getBucket(i)
	b := getBucketASM(i)

	if a != b {
		t.Fatalf("Results don't match: a: %d b: %d | input was %d", a, b, i)
	} else {
		t.Log("Results match")
	}
}

func TestCompareStandardAndAssembly(t *testing.T) {
	i := uint64(1)
	for i < 0x8000000000000000 {
		t.Logf("Trying input %d", i)

		a := getBucket(i)
		b := getBucketASM(i)

		if a != b {
			t.Fatalf("Results don't match: a: %d b: %d | input was %d", a, b, i)
		} else {
			t.Log("Results match")
		}

		i <<= 1
	}
}

func TestQuickCompareStandardAndAssembly(t *testing.T) {
	f := func(n uint64) bool {
		a := getBucket(n)
		b := getBucketASM(n)

		if a != b {
			t.Logf("Results don't match: a: %d b: %d", a, b)
		}

		return a == b
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}
