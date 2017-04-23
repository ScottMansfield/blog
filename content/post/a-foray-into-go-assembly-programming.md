+++
title = "A Foray Into Go Assembly Programming"
date = "2017-04-21T03:37:23-07:00"
categories = ["Go", "ASM"]
keywords = ["code", "organization", "software development", "github", "golang", "go"]
+++

This blog post started last August when I was integrating the [Spectator](https://github.com/Netflix/spectator)
[PercentileTimer](http://netflix.github.io/spectator/en/latest/javadoc/spectator-api/com/netflix/spectator/api/histogram/PercentileTimer.html)
concept into the [metrics library](https://github.com/Netflix/rend/tree/master/metrics) in
[Rend](https://github.com/Netflix/rend) so we could get better cross-fleet latency percentiles.

As a part of doing this, I had to port the
[code that selects which bucket](https://github.com/Netflix/spectator/blob/ba7811588eb035f98d161ab5e179fee9432438a8/spectator-api/src/main/java/com/netflix/spectator/api/histogram/PercentileBuckets.java#L64)
(counter) to increment inside the `PercentileTimer` distribution. A `PercentileTimer` is implemented
as a large array of counters, each of which represent a bucket. They are incremented whenever the
observation lands in that bucket. The [Atlas](https://github.com/Netflix/atlas) backend can then use
these (properly tagged) counters as a group to derive cross-fleet percentiles within a couple
percent error. The bucketing scheme divides the range of `int64` into powers of 4, which are then
subdivided linearly by 3 for the set of final buckets. This code is farly quick to run and compact,
if a bit obtuse at first.

### Side note: optimizations

When I saw a divide by 3, I shuddered a little bit because I assumed it would be less efficient to
do the division as a `DIV` instruction instead of as a shift operation like a divide by 4 or 2 would
be. Little did I know that people had solved this problem before. It's a common compiler
optimization to apply [Montgomery Division](https://en.wikipedia.org/wiki/Montgomery_modular_multiplication)
when the division is by an integer constant. In this case, a divide by 3 is equivalent to
multiplying by `0x55555556` and then taking the top half of the output.
[Thanks, StackOverflow](http://stackoverflow.com/questions/171301/whats-the-fastest-way-to-divide-an-integer-by-3?rq=1)

## The descent

At this point I was also looking for an excuse to program something in assembly with Go. I actually
already had a couple of other files in Rend that had assembly in them, but they were borrowed from
other places and not my own original work. I wanted to translate this bucket selection code into
assembly to see how fast I could make it.

The first step was to translate it into Go code so I had something to compare my assembly code
against. This was easy as it was pretty straightforward to change the Java code into Go. The only
hangup was recreating the indexes array that is part of the static initialization in the Java code.

From there, it was time to create the function and just get it set up to be called. This is where
the trickery begins. From here on this post is mostly a list of things that I ran into that I had to
do some research to solve.

## Go version

This is the entire Go version of the code below for reference. This is the original code that I am
working off of throughout this process.

```go
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
```

## The setup

Declaring an assembly function is more complex than a standard function. There is the equivalent of
a C function declaration in Go code and then the actual assembly implementation in another `.s` file.

There's a few things that are sticky when making a new function:

### "Bridging" Go and ASM

This is something I had a little difficulty with because the method was not entirely clear at first.

In order to be able to compile the program, you need to create a Go version of the function
declaration in a `.go` file alongside the `.s` file that contains the assembly implementation:

```go
func getBucketASM(n uint64) uint64
```

The way to think about this, at least in my mind, is that this is your interface definition in Go
and the implementation is in assembly in another file. Go code uses the interface.

### The middot and (SB)

Function names in Go assembly files start with a middot character (`·`). The function declaration
starts like this:

```armasm
TEXT ·getBucketASM(SB)
```

`TEXT` means that the following is meant for the text section of the binary (runnable code). Next
comes the middot `·` then the name of the function. Immediately after the name the extra `(SB)` is
required. This means "stack base" and is an artifact of the Plan9 assembly format. The real reason,
from the [Plan9 ASM documentation](http://plan9.bell-labs.com/sys/doc/asm.html), is that functions
and static data are located at offsets relative to the beginning of the start address of the
program.

I literally copy-paste the middot every time. Who has a middot key?

### To NOSPLIT or not to NOSPLIT

The asm doc says this about `NOSPLIT`:

> NOSPLIT = 4 
> (For TEXT items.) Don't insert the preamble to check if the stack must be split. The frame for the
> routine, plus anything it calls, must fit in the spare space at the top of the stack segment. Used
> to protect routines such as the stack splitting code itself.

In this function's case, we can add `NOSPLIT` because the function doesn't use any stack space at
all beyond the arguments it receives. The annotation was probably not strictly necessary, but it's
fine to use in this case. At this point I'm still not sure how the "spare space" at the top of the
stack works and I haven't found a good bit of documentation to tell me.

### How much stack space?

If your function requires more space than you have registers, you may need to spill on to the stack
temporarily. In this case, you need to tell the compiler how much extra space you need in bytes.
This function doesn't need that many temporary variables, so it doesn't spill out of the registers.
That means we can use `$0` as our stack space. The stack space is the last thing we need on the
declaration line.

At this point we have one line done!

```armasm
TEXT ·getBucketASM(SB), 4, $0
```

I have `4` written instead of `NOSPLIT` because I wasn't quite doing things right. I'll get to that
in the next section.

## Static array

In order to make the algorithm work, I needed to declare a static array of numbers that represent
offsets into the array of counters. First, I derived a declaration from the standard library AES GCM
code:

```armasm
GLOBL powerOf4Index<>(SB), (NOPTR+RODATA), $256
> unexpected NOPTR evaluating expression
```

This didn't work, however, because the `NOPTR` and `RODATA` symbols were undefined. I tried each on
their own:

```armasm
GLOBL powerOf4Index<>(SB), RODATA, $256
> illegal or missing addressing mode for symbol RODATA

GLOBL powerOf4Index<>(SB), NOPTR, $256
> illegal or missing addressing mode for symbol NOPTR
```

Again, same result. To be expected, because they weren't defined before. I didn't know this at the
time, though, because I was flailing about in the dark. I tried it without the annotation at all:

```armasm
GLOBL powerOf4Index<>(SB), $256
> missing Go type information for global symbol
```

Again, no dice. It needs something there to tell the compiler how to treat the data.

It took me a while to find, but the [asm documentation on the official Go website](https://golang.org/doc/asm)
was actually the most helpful here. For "some unknown reason," my code was unable to compile with
the mnemonics so I just replaced `RODATA` and `NOPTR` with the numbers that represent them:

```go
GLOBL powerOf4Index<>(SB), (8+16), $256
```

Aha! These two symbols tell the compiler to treat the "array" as a constant and not having any
pointers.

Of course hindsight is 20/20, meaning that after this entire exercise was over I found the
[proper header file](https://github.com/golang/go/blob/964639cc338db650ccadeafb7424bc8ebb2c0f6c/src/runtime/textflag.h)
to include to get these symbols. I didn't figure out how to actually compile my code with this
header file in place for this post, but the assembly files in the Go codebase all include it right
at the top:

```
#include "textflag.h"
```

It's also important to note how the data is laid out. Each `DATA` line is declaring a value for a
given 8 byte chunk of the static data. The name of the "array" is first, followed by the offset (the
type of which is defined in [this picture](http://plan9.bell-labs.com/sys/doc/asm0.png) from Plan9)
and the size, then finally the value. After all of the `DATA` lines are complete, the `GLOBL` symbol
`powerOf4Index` is declared along with some flags and its total size.

Now the Go array

```go
var powerOf4Index = []int{
	0, 3, 14, 23, 32, 41, 50, 59, 68, 77, 86, 95, 104,
	113, 122, 131, 140, 149, 158, 167, 176, 185, 194,
	203, 212, 221, 230, 239, 248, 257, 266, 275,
}
```

has become this block of assembly `DATA` declarations:

```armasm
DATA powerOf4Index<>+0x00(SB)/8, $0
DATA powerOf4Index<>+0x08(SB)/8, $3
DATA powerOf4Index<>+0x10(SB)/8, $14
DATA powerOf4Index<>+0x18(SB)/8, $23
DATA powerOf4Index<>+0x20(SB)/8, $32
DATA powerOf4Index<>+0x28(SB)/8, $41
DATA powerOf4Index<>+0x30(SB)/8, $50
DATA powerOf4Index<>+0x38(SB)/8, $59
DATA powerOf4Index<>+0x40(SB)/8, $68
DATA powerOf4Index<>+0x48(SB)/8, $77
DATA powerOf4Index<>+0x50(SB)/8, $86
DATA powerOf4Index<>+0x58(SB)/8, $95
DATA powerOf4Index<>+0x60(SB)/8, $104
DATA powerOf4Index<>+0x68(SB)/8, $113
DATA powerOf4Index<>+0x70(SB)/8, $122
DATA powerOf4Index<>+0x78(SB)/8, $131
DATA powerOf4Index<>+0x80(SB)/8, $140
DATA powerOf4Index<>+0x88(SB)/8, $149
DATA powerOf4Index<>+0x90(SB)/8, $158
DATA powerOf4Index<>+0x98(SB)/8, $167
DATA powerOf4Index<>+0xa0(SB)/8, $176
DATA powerOf4Index<>+0xa8(SB)/8, $185
DATA powerOf4Index<>+0xb0(SB)/8, $194
DATA powerOf4Index<>+0xb8(SB)/8, $203
DATA powerOf4Index<>+0xc0(SB)/8, $212
DATA powerOf4Index<>+0xc8(SB)/8, $221
DATA powerOf4Index<>+0xd0(SB)/8, $230
DATA powerOf4Index<>+0xd8(SB)/8, $239
DATA powerOf4Index<>+0xe0(SB)/8, $248
DATA powerOf4Index<>+0xe8(SB)/8, $257
DATA powerOf4Index<>+0xf0(SB)/8, $266
DATA powerOf4Index<>+0xf8(SB)/8, $275

// RODATA == 8
// NOPTR == 16

GLOBL powerOf4Index<>(SB), (8+16), $256
```

If you properly import `textflag.h` then you could just change the declaration to be like it was
originally:

```armasm
GLOBL powerOf4Index<>(SB), (NOPTR+RODATA), $256
```

As for most of my struggles, careful reading of the [Go ASM doc](https://golang.org/doc/asm#directives)
would have explained this to me.

## Troubleshooting the shifts

> Illegal instruction 

This is what I was faced with. Not much information there. I did manage to isolate the error to the
most recent bit of code I had added, which had both a `SHRQ` and a `SHLQ`, which shift a quadword
(64 bits) right and left, respectively. These can shift by a fixed amount or by a dynamic amount.
I needed to use the dynamic amount in this case. The same mnemonic actually produces two different
encodings at the binary level because the instructions for dynamic and static shift amounts are
different.

I had written the code to use some arbitrary register because I hadn't thought anything of it at the
time. It so turns out that the assembler was smart enough to recognize that the instruction was not
actually encodeable, which is what the error message meant in the first place.

I dug around a little bit, not really knowing exactly where to start. Eventually, I looked at the
[SSA code](https://github.com/golang/go/blob/3572c6418b5032fbd7e888e14fd9ad5afac85dfc/src/cmd/compile/internal/ssa/opGen.go#L1964)
in the Go compiler to see what kind of logic they had around the instruction. Jackpot.

The SSA code showed me only CX can be used as a variable shift amount:

```go
//...
{
	name:         "SHRQ",
	argLen:       2,
	resultInArg0: true,
	asm:          x86.ASHRQ,
	reg: regInfo{
		inputs: []inputInfo{
			{1, 2},     // CX
			{0, 65519}, // AX CX DX BX BP SI DI R8 R9 R10 R11 R12 R13 R14 R15
		},
		clobbers: 8589934592, // FLAGS
		outputs: []regMask{
			65519, // AX CX DX BX BP SI DI R8 R9 R10 R11 R12 R13 R14 R15
		},
	},
},
//...
```

After this I had a bit of an epiphany: I could have just looked at the
[Intel manuals](https://software.intel.com/en-us/articles/intel-sdm) the whole time. In the manual
(page 4-582 Volume 2B) it shows only the CL register being usable as the shift amount for the
dynamic versions of `SHR` and `SHL`.

## Final code

I've reproduced the entire assembly version of the code here without comments for brevity. If you
want to see the entire thing, you can take a look at the source files below. This code depends on
the data declaration above.

```x86asm
TEXT ·getBucketASM(SB), 4, $0
        MOVQ x+0(FP), R8
        CMPQ R8, $16
        JC underSixteen
sixteenAndOver:
        BSRQ R8, BX
        SUBQ $63, BX
        NEGQ BX
        MOVQ $63, R10
        SUBQ BX, R10
        MOVQ R10, BX
        MOVL $1, CX
        ANDQ R10, CX
        JEQ powerOfFour
        SUBQ $1, R10
powerOfFour:
        MOVQ R8, R9
        MOVQ BX, CX
        SHRQ CX, R9
        MOVQ R10, CX
        SHLQ CX, R9
        MOVQ $0x5555555555555556, AX
        MULQ R9
        MOVQ DX, BX
        MOVQ R8, AX
        SUBQ R9, AX
        MOVQ $0, DX 
        DIVQ BX
        SHLQ $2, R10
        LEAQ powerOf4Index<>(SB), DX
        MOVQ (DX)(R10*1), BX
        ADDQ BX, AX
        CMPQ AX, $275
        JGE bucketOverflow
        ADDQ $1, AX
        MOVQ AX, ret+8(FP)
        RET
bucketOverflow:
        MOVQ $275, ret+8(FP)
        RET
underSixteen:
        MOVQ R8, ret+8(FP)
        RET
```

## Benchmarks

So what did all this struggle get me?

I wrote up some benchmarks to compare three versions of the code:

1. The original Go version above
1. The same Go code above but with `//go:noinline` to try to control for inlining
1. The assembly version

```
$ go test -run asdsdf -bench . -count 100 | tee -a benchdata
...
$ benchstat benchdata
name                 time/op
GetBucket-8          17.4ns ± 2%
GetBucketNoInline-8  17.4ns ± 2%
GetBucketASM-8       12.8ns ± 1%
```

The answer is 4.6ns. 

If it took me 4 hours to write this code, it would have to run 3.13043478 × 10<sup>12</sup> times to
be worthwhile. Luckily, this code would be run that many times in about a day or so in production at
Netflix.

However, I didn't use it.

I have a couple reasons:

1. **Maintenance.** If this code needs to be changed in the future by me or anyone else, I (or they)
would need to brush up on assembly on x86, Go assembly quirks, etc. in order to do so. There's quite
a lot of overhead in that.
1. The time difference **wasn't worth optimizing** in this case. I only did it for fun. The
latency of this code is in the low dozens of microseconds, so saving 15 or 20 nanoseconds per
request is not useful. There are bigger fish to fry that won't introduce the programmer overhead of
assembly code.

For those of you wondering, I did actually benchmark the ported Go code before starting this whole
process.

I know this may sound rather disappointing, doing all this work without putting it into production,
but it was a fantastic exercise in learning Go assembly idiosyncrasies. Hopefully this chronicle of
my struggles helps you overcome some small hurdle in your assembly ventures.

Please let me know what you think about this article in the comments or [@sgmansfield](https://twitter.com/sgmansfield)
on twitter.

## Files

If you would like to peruse the files that made up this work in their entirety, you can find them here:

* [Consolidated Java code](/files/2017/04/BucketTest.java) to get the static data
* [bucket_test.go](/files/2017/04/bucket_test.go)
* [bucket_test_asm.go](/files/2017/04/bucket_test_asm.go)
* [bucket_test_amd64.s](/files/2017/04/bucket_test_amd64.s)

You can also see them at the [GitHub repository for this blog](https://github.com/ScottMansfield/blog).

## References

Sites / documents:

* https://golang.org/doc/asm
* http://plan9.bell-labs.com/sys/doc/asm.html
* http://plan9.bell-labs.com/sys/doc/asm0.png
* https://software.intel.com/en-us/articles/intel-sdm
* https://goroutines.com/asm

Code:

* https://golang.org/src/crypto/aes/gcm_amd64.s
* https://golang.org/src/runtime/sys_windows_amd64.s
* https://github.com/golang/go/blob/3572c6418b5032fbd7e888e14fd9ad5afac85dfc/src/cmd/compile/internal/ssa/opGen.go#L1964
