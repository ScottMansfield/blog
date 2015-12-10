+++
title = "CUDA Timing Woes"
date = "2010-09-25T22:03:00-05:00"
tags = ["timing", "cuda", "c", "nvidia", "cudaEvent"]
categories = ["C", "CUDA"]
+++

I have come to realize that programming using Nvidia's CUDA can be a major PITA pit when a problem
crops up. Generally there are few resources besides the CUDA Programming Guide that are helpful.
Most of the answers that one would find in their searches are on the forums at Nvidia, and even then
many searches turn up unanswered questions.

My particular problem was in the realm of timing the execution of a kernel on the device. In the
example code that was provided to me in my class, they used several undocumented functions from the
`cutil.h` (or, in my case, the `cutil_inline.h`) header. These functions included:

* cutCreateTimer
* cutStartTimer
* cutStopTimer
* cutGetTimerValue
* cutDeleteTimer

All of these functions are not officially supported by Nvidia. The exact error that I was getting
was this:

> ld: cannot find -lcutil

Basically, the linker could not find the static cutil library to link against. I used the command
line

    nvcc example.cu -Iinclude -lcutil -L/opt/cuda-sdk/lib

where the path `/opt/cuda-sdk/lib` contained the file `libcutil.a`, which is the shared library
version of cutil. I searched high and low and couldn't find a `libcutil.so` anywhere inside the SDK
file structure. As it turns out, it doesn't exist, at least not in the SDK version that I am using.

The way you are supposed to time something is by using **events**. I will get to events in a minute,
but first I want to describe the journey I took to get there.

## Google, thou hast failed me

I spent a total of probably 4 hours of Google searching, interspersed between other classes and
homework, which got me basically nowhere. I found various sources all telling me something
different. I tried many of these different approaches, including:

* looking in cutil.h for clues on what the function declarations looked like to see if maybe I could
add a function prototype to help out the linker (wrong direction... I think. No effect anyway)
* adding the phrase 'extern "C"' before the prototype declaration (no effect)
* decompiling the shared library and recompiling it into a static library (which didn't work anyway)

The above are just a few of the many different things I tried while searching. Most of the
suggestions I tried were from Nvidia forums. In the forums also were about 20 threads that I came
across that said to just reference the library using `-lcutil` and `-L/path/to/library`, which
didn't work in the first place.

## Nvidia to the rescue

In the end I gave up and started looking through the [Nvidia CUDA Programming Guide](http://develope
r.download.nvidia.com/compute/cuda/3_1/toolkit/docs/NVIDIA_CUDA_C_ProgrammingGuide_3.1.pdf)(warning:
large PDF. [source](http://developer.nvidia.com/object/gpucomputing.html)) page by page. I came
across an example in the middle (Page 39, Section 3.2.7.6) that focuses on events and how they can
be used to time execution. Here is the basic structure for timing something in your program (in C):

    cudaEvent_t start, stop;
    cudaEventCreate(&amp;start);
    cudaEventCreate(&amp;stop);
    cudaEventRecord(start, 0);
    
    // Whatever needs to be timed goes here
    
    cudaEventRecord(stop, 0);
    cudaEventSynchronize(stop);
    float elapsedTime;
    cudaEventElapsedTime(&amp;elapsedTime, start, stop);

## So, there you have it

If you are trying to time something using any of the `cutXxXxTimer` functions, stop using those and
start using events. The functions are officially supported, aren't any harder to use, and, best of
all, they actually link correctly when you go to compile your program.
