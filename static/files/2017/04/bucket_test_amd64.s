// asm for finding the proper atlas bucket

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

// func getBucketASM(n uint64) uint64
//
// R8 contains the input number n throughout the function
// R9 contains the previous power of 4
// R10 is the left shift amount, then shamt/2
//
// NOSPLIT == 4
TEXT ·getBucketASM(SB), 4, $0
        // if n <= 15, return n
        MOVQ x+0(FP), R8
        CMPQ R8, $16

        // Use jump if carry flag set here instead of JLT because we are dealing with unsigned ints
        JC underSixteen

sixteenAndOver:
        // The more complicated procedure begins
        // The next few sections get the previous power of 2

        // Get the base shift amount, also called the previous power of two
        // The shift amount is 63 - lzcnt(R8)
        // We can ignore the case where the input to BSRQ is zero (and the output is undefined)
        // because the test above guarantees it's at least 16 
        BSRQ R8, BX
        SUBQ $63, BX
        NEGQ BX            // BX now has the leading zero count
        MOVQ $63, R10
        SUBQ BX, R10

        // R10 now contains the base shift amount
        // From here we are getting the previous power of 4 from the previous power of 2
        // Test if R10 (shift amount) is even or odd by doing R10 & 1
        // if even, we shift right and left the same amount since this power of 2 is a power of 4
        // if odd, we shift right then shift back one less to get previous power of 4
        // From here we will need BX to be the right shift amount and R10 to be the left shift amount
        // So proactively move R10 to BX, then subtract 1 from R10 only if it's odd
        MOVQ R10, BX

        // Test if R10 (left shift amount) is odd, if so subtract 1
        MOVL $1, CX
        ANDQ R10, CX
        JEQ powerOfFour

        // subtract 1 from R10 (left shift) in place because it was odd
        SUBQ $1, R10

powerOfFour:
        // Move R8 (input number n) to R9 (prev power of 4) to be shifted around
        // then shift right then back left to get prev power of 4
        // CX is the only allowed input register for a variable shift amount 
        MOVQ R8, R9
        MOVQ BX, CX
        SHRQ CX, R9
        MOVQ R10, CX
        SHLQ CX, R9


        // The next few sub sections implement this code
        // delta = ppo4 / 3
        // temp = n - ppo4
        // offset = temp / delta
 
        // Get the delta by dividing the previous power of 4 by 3
        MOVQ $0x5555555555555556, AX // magic constant 
        MULQ R9
        MOVQ DX, BX
        // BX now contains the quotient from ppo4 / 3

        // get temp, first copy input to AX, then subtract R9 (prev power of 4)
        MOVQ R8, AX
        SUBQ R9, AX

        // Divide temp (in AX:DX) by delta (in BX)
        // This will overwrite AX with the quotient and DX with the remainder
        // but first clear DX for the upcoming DIVQ
        MOVQ $0, DX 
        DIVQ BX
        // AX now contains offset

        // pos := offset + powerOf4Index[shift/2]

        // Shift R10 right one to get index into powerOf4Index
        // Then immediately shift left 3 to get the index in bytes
        // So it ends up being a shift left by 2
        // This works because we know the shift amount is even already, it was forced
        // to be so earlier
        SHLQ $2, R10

        // need to access powerOf4Index[R10]
        LEAQ powerOf4Index<>(SB), DX
        MOVQ (DX)(R10*1), BX

        // Add the offset plus the previous power of 4 index
        ADDQ BX, AX
        // And now AX almost contains the bucket

        // numBuckets-1 = 275
        // Check for array bounds, max index is 275
        CMPQ AX, $275
        JGE bucketOverflow
        
        // Return the calculated bucket + 1
        ADDQ $1, AX
        MOVQ AX, ret+8(FP)
        RET

bucketOverflow:
        // AX was greater than 275, so return 275
        MOVQ $275, ret+8(FP)
        RET

underSixteen:
        MOVQ R8, ret+8(FP)
        RET
