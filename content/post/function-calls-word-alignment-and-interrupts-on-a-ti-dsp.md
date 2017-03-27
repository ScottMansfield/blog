+++
title = "Function Calls, Word Alignment, and Interrupts on a TI DSP"
date = "2011-10-13T16:23:00-05:00"
keywords = []
categories = ["Assembly", "DSP"]
+++

## Background

I was taking the microprocessors class last fall and started writing this post... now here it is! I
had a couple of labs that were giving me problems. The particular DSP I was using for my lab (the
Texas Instruments TMS320F28335) has a word size of 16 bits. When writing a ton of assembly code for
a couple of the labs, I came across an error that cropped up concerning word alignment that was
particularly obvious when using an interrupt routine set to go off every millisecond or so.

## The Problem With Stack Misalignment (No Interrupts)

When a stack becomes misaligned, it creates problems when pushing or popping 32 bit values. Since
the DSP I was using addresses memory by words, a 32 bit push will write to two places in memory.
When writing to an address like `0x1000`, the write will place the 32 bit value to addresses
`0x1000` and `0x1001`. Order is unimportant because I use a 32 bit pop as the reverse. Here is the
meat of the problem: if the stack pointer is misaligned, that is it points to an **odd** address in
memory, the 32 bit push will overwrite the previously placed value. This is very bad when attempting
to return one word from a function call or when an interrupt suspends the main "thread" (for lack of
a better term) in between two 16 bit pushes. Note that this may not be the *exact* way the stack
works on the DSP, but conceptually it is the same.

The first time I encountered this problem was when I had a keypad scanning function that would
return a single word, the key code. When calling the function (at least in the initial lab) there
were no problems. The return address of the function call was placed on the stack and the function
ran. It is important to note that the address is 32 bits, or 2 words, long. The function would then
scan the key pad for any depressed buttons and return a key code regardless. This key code was 16
bits, or 1 word, long. The reason I removed the return address from the stack is because it is
required to be at the top of the stack when the RET instruction is executed. That is where the DSP
will take the address from. In a mirror fashion, a 2 word, 32 bit read from memory at an odd
location will read that location and the one below, not the ones expected. If a read is performed
at 0x1001, I would expect `0x1001` and `0x1002` to be given back. Not so. The DSP returns `0x1000`
and `0x1001`. Effectively, a 32 bit read from `0x1000` is the same as a 32 bit read from `0x1001`.
The picture below illustrates what happens when this problem occurs.

![TI DSP Stack Misalignment](/img/2011/10/TI-DSP-Stack-Misalignment.png)

As the picture shows, this situation eventually leads to the function call returning to an unknown
location, which is generally random and contains uninitialized memory. This causes all sorts of fun
things to happen, like flashing LEDs and a general dysfunctional state in which the board must be
reset and reprogrammed.

## The Solution (No Interrupts)

The solution to this problem was simple, in retrospect (hindsight is 20/20, after all). I simply
pushed a second word with a dummy value, in this case 0, so the stack would stay aligned. This
required two push instructions in a row but it made the program work. The stack pointer always gets
incremented or decremented by 2 memory locations, ensuring that it stays on an even value. The
graphic below shows the execution of the program after this fix is applied.

![TI DSP Stack Misalignment fix](/img/2011/10/TI-DSP-Stack-Misalignment-fix.png)

## The Problem With Stack Misalignment (With Interrupts)

The previous solution is just fine considering the circumstances it was in. There are no
asynchronous tasks that could interrupt the execution of the main "thread." This is unrealistic,
however, because real systems will have interrupts on regular or irregular bases which can interrupt
the code execution at any time. These essentially execute as a function call, pushing the return
value on the stack (as a 2 word value) and going to the interrupt handler body. These
pseudo-function-calls must be assumed to occur between any two instructions in the main program.
This causes the same stack misalignment error as before in some situations.

The previous solution would place two words on to the stack, one word at a time. This means that two
separate instructions were required, providing a kink in the armor in which an interrupt can wreak
havoc. The program may even execute just fine for awhile, but eventually an interrupt will execute
between two of the push instructions, creating the same situation as before. The graphic
representation of this would be the same as above.

## The Solution (With Interrupts)

The solution that I decided to use was to pass everything through a 32 bit register on its way to
the stack. I would place a 16 bit dummy value side by side with a 16 bit value to be pushed on the
stack and use one push instruction. The DSP may need to actually make two trips to memory, but at
least it is guaranteed to not be interrupted between writing both words. This solution would
graphically look the same as the solution above. After fixing this, the program proceeded to run
just fine, even with interrupts.

## Conclusion

When using a DSP that operates on a stack, misalignment can occur. Misalignment places the stack
pointer at an odd position in the stack. Upon writing a 32 bit value, or 2 words, to the stack, the
data will overwrite the singular 16 bit value in the even position just below the stack pointer and
the stack pointer will point to uninitialized data. When the function or interrupt returns, it may
jump to uninitialized memory. The solution to both of these problems is to push and pop only 32 bit
values on and off the stack, packing and unpacking data as necessary.
