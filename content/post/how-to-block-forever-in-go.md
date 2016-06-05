+++
title = "How to Block Forever in Go"
date = "2016-06-05T00:17:35-07:00"
categories = ["Go"]
keywords = ["Go", "golang", "block", "forever", "select", "infinite loop"]
+++

## Let me count the ways

There seems to be quite a few ways to block forever in Go. This post is part
code golf and part practical advice. All of the code below is simply to find a
way to block the current goroutine from progressing while still allowing others
to continue. The versions that block forever will actually cause a panic saying
that all goroutines are asleep, but the ones that "busy block" will just time
out on the playground.

## Why Not an Infinite Loop?

An infinite loop will simply use 100% of a CPU an not allow the Go runtime to
schedule any other work on that core. All of the solutions below will either
block forever or very quickly yield the processor for other work.

(thanks [@bradleyfalzon](https://twitter.com/bradleyfalzon))

## Empty Select

This function tries to select the first case that returns, so it will happily
wait forever for nothing to return.

```
func blockForever() {
    select{ }
}
```

## Waiting on Itself

A `sync.WaitGroup` is a way to coordinate multiple goroutines by reporting
completion. Since there's nobody to report completion, it never returns.

```
import "sync"

func blockForever() {
    wg := sync.WaitGroup{}
    wg.Add(1)
    wg.Wait()
}
```

## Double Locking

A `sync.Mutex` provides exclusive use of another associated resource. What if
it's requested twice? Oops, blocked.

```
import "sync"

func blockForever() {
    m := sync.Mutex{}
    m.Lock()
    m.Lock()
}
```
The same trick works with a `sync.RWMutex` that has been `RLock()`'d.

## Reading from an Empty Channel

Empty unbuffered channels will block until there's something to receive.

```
func blockForever() {
    c := make(chan struct{})
    <-c
}
```

This also works with a nil channel not made with `make()` (H/T: @cgilmour on the
Gophers Slack).

## "Busy Blocking"

This method technically doesn't block, it just constantly defers to other work
if it's available.

```
import "runtime"

func blockForever() {
    for {
        runtime.Gosched()
    }
}
```

## The Decomposed Loop

Same as before, not technically blocking but looping and deferring to other
work.

```
import "runtime"

func blockForever() {
    foo: runtime.Gosched()
    goto foo
}
```

## Shaking my Own Hand

This is a bit like shaking your own hand. This function will continually send a
useless message to itself until the end of time. The channel send operations are
opportunities for the runtime to schedule other goroutines, so this method would
not monopolize a single processor.

```
func blockForever() {
    c := make(chan struct{}, 1)
    for {
        select {
        case <-c:
        case c <- struct{}{}:
        }
    }
}
```

## Sleeping for a Looooong Time

Sleeping to the maximum time allowed takes quite a long time, roughly 292.4
years. Likely, your code won't run that long, so I'd consider this equivalent to
`select{}`.

```
import (
    "math"
    "time"
)

func blockForever() {
    <-time.After(time.Duration(math.MaxInt64))
}
```

This comes from @hydroflame (on the Gophers Slack) of [Lux](https://github.com/l
uxengine) fame.

## Are These Useful?

The first one is likely the only one you'll ever need. For example, if you have
a single server with multiple http listeners on different ports, you can run
them all as separate goroutines in your `main()` and just block forever with a
`select{}`. It's likely that any failure of a server ought to end your program,
so having errors `panic()` will end the process.

If for some reason you *don't* want your program to end on a single server
failing (or stopping), you can create a `*sync.WaitGroup` to pass in to each
wrapper goroutine and wait on it before ending the `main()` function.

```
import (
    "log"
    "net/http"
    "sync"
)

func main() {
    wg := &sync.WaitGroup{}
    wg.Add(4)
    
    for i := 1111; i < 1115; i++ {
        go func(port int, wg *sync.WaitGroup) {
            err := http.ListenAndServe(fmt.Sprintf(":%d", port), someHandlerFunction)
            log.Print(err)
            wg.Done()
        }(i, wg)
    }
    
    wg.Wait()
}
```

The rest of the methods are just for fun. if you have any other methods that you
know of that will not monopolize a CPU, leave a comment and let me know.
Enjoy :)