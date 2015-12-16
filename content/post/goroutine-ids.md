+++
title = "Goroutine IDs"
date = "2015-12-16T01:50:55-08:00"
categories = ["Go", "Hacks"]
keywords = ["go", "golang", "goroutine", "id"]
+++

## Do Goroutine IDs Exist?

Yes, of course. The runtime has to have some way to track them.

## Should I use them?

No.
[No.](https://groups.google.com/forum/#!topic/golang-nuts/Nt0hVV_nqHE)
[No.](https://groups.google.com/forum/#!topic/golang-nuts/0HGyCOrhuuI)
[No.](http://stackoverflow.com/questions/19115273/looking-for-a-call-or-thread-id-to-use-for-logging
)

## Are There Packages I Can Use?

There exist packages from people on the Go team, with colorful descriptions like
"[If you use this package, you will go straight to hell.](https://godoc.org/github.com/davecheney/ju
nk/id)"

There are also packages to do goroutine local storage built on top of goroutine IDs, like
[github.com/jtolds/gls](https://github.com/jtolds/gls) and
[github.com/tylerb/gls](https://github.com/tylerb/gls)
that use dirty knowledge from the above to create an environment contrary to Go design principles.

## Minimal Code

So you like pain? Here's how to get the current goroutine ID:

### The Hacky Way in "Pure" Go

This is adapted from Brad Fitzpatrick's [http/2 library](https://github.com/golang/net/blob/master/h
ttp2/gotrack.go)
that is being rolled into Go 1.6. It was used solely for debugging purposes, and is not used during
regular operation

```Go
package main

import (
    "bytes"
    "fmt"
    "runtime"
    "strconv"
)

func main() {
    fmt.Println(getGID())
}

func getGID() uint64 {
    b := make([]byte, 64)
    b = b[:runtime.Stack(b, false)]
    b = bytes.TrimPrefix(b, []byte("goroutine "))
    b = b[:bytes.IndexByte(b, ' ')]
    n, _ := strconv.ParseUint(string(b), 10, 64)
    return n
}

```

#### How does this work?

It is possible to parse debug information (meant for human consumption, mind you) and retrieve the
goroutine ID. There's even **debug** code in the http/2 library that uses this to track ownership of
connections. It was used sparingly, however, for debugging purposes only. It's not meant to be a
high-performance operation.

The debug information itself was retrieved by calling [`runtime.Stack(buf []byte, all bool) int`](ht
tps://golang.org/pkg/runtime/#Stack) which will print a stacktrace into the buffer it's given in
textual form. The very first line in the stacktrace is the text "goroutine #### [..." where #### is
the actual goroutine ID. The rest is just text manipulation to extract and parse the number. I took
out some error handling, but if you want the goroutine ID in the first place, I assume you want to
live dangerously already.

### The Legitimate CGo Version

The C version is from [github.com/davecheney/junk/id](https://github.com/davecheney/junk/tree/master
/id) where the C code directly accesses the `goid` property of the current goroutine and just
returns that. The code below was copied verbatim from Dave Cheney's repo, so all credit to him.

File `id.c`
```C
#include "runtime.h"

int64 ·Id(void) {
	return g->goid;
}
```

File `id.go`
```Go
package id

func Id() int64
```

## Where to go from here?

Running screaming away from goroutine IDs. Forget they exist, and you will be much happier. The use
of them is a red flag from a design standpoint, because nearly all uses (speaking from research, not
tacit knowledge) try to create something to do goroutine-local state, which violates the "[Share
Memory By Communicating](https://blog.golang.org/share-memory-by-communicating)" tenet of Go
programming.
