+++
title = "10k Concurrent Connections"
date = "2016-08-05T01:09:22-07:00"
categories = ["Go", "Performance"]
keywords = ["go", "golang", "concurrency", "c10k", "web server", "parallelism", "rend", "netflix"]
+++

After my [recent appearance](https://changelog.com/gotime-9/) on the Go Time podcast I thought it
would be fun to write a post on a few performance bits in [Rend](https://github.com/netflix/rend),
the software project I hack on at [work](https://netflix.com). 

This topic comes up frequently enough that I felt I should write about it. The c10k problem has many
solutions already. This is just my take.

Fortunately for people in 2016, handling 10 thousand concurrent connections is not really all that
difficult anymore. Servers have long been able to handle that many connections, so why is it a
topic? Mostly because 10 thousand is a nice round number to aim for, but realy I think it's because
most servers do not have to handle that many in the first place. You likely won't see a brand new
Rails or Django app or fresh Wordpress install that can work. This is perfectly fine. C10k is really
about I/O bound apps managing the traffic between point A and point B.

Now, with some of the philosophy out of the way, let's look at how we can solve the c10k problem
fairly easily using Go.

## One Goroutine per Connection

A common pattern for a TCP-based server in Go is to have a connection accept loop spin off a
separate goroutine per accepted connection. This works well for a variety of reasons:

#### Simplifies Code

As the author you have less to worry about in a single function. It applies to the connection you're
currently servicing and nothing else. Of course, there's some small exceptions.

#### I/O Events and Continuation are Transparent

When a goroutine performs a blocking action, it is placed into a waiting state while other
goroutines that can run are run. Once the conditions are met for the goroutine to run again, it will
be placed back into its work queue. This is an essential part of the green thread / multiplexing
model that Go uses. It also means you get some fo the advantages of non-blocking IO while not 
descending into callback or asynchronous `.and_then` hell.

Dave Chenety wrote an article that referenced the c10k problem and why IO polling is not an issue:
http://dave.cheney.net/2015/08/08/performance-without-the-event-loop

#### Downside: Memory

Memory increases linearly with the number of connections. At 10k connections the overhead of buffers
and the like should be in the low hundreds of megabytes. There is a nonzero cost to having buffers
and I/O structs per connection, but in general this does not matter. Even for Rend, which runs on
boxes that are, essentially, memory as a service, the overhead is not enough to matter.

## Resources are per-connection

Expanding a bit on the point above: Most of the code to handle a request can be straightline code,
not worrying about any other connections. Even a panic only affects the one incoming and related
outgoing connections. In Rend, even the connections are isolated, where once incoming external
connection is tied to a specific set of outgoing connections to the backends. If there is a problem
anywhere in Rend that panics, the grouped connections are closed and the rest of the connections
live on.

To make things concrete, this is how Rend is structured:

1. One main listener `accept` loop - ok because connections are long-lived and reconnects are rare
2. Each new connections gets its own `server` - basically a REPL
3. Each `server` owns an `orca`, the request orchestrator
4. Each `orca` owns one or more `handler`s that are able to communicate to the backend
5. Each `handler` owns the connection to their backend

## Prefer Atomic Operations for Simple Cases
a.k.a. minimize (the existence of) critical sections in locks

Now this comes with a HUGE caveat: `sync/atomic` can be hard to get right, and weird bugs can result
if it is used improperly. 

https://groups.google.com/d/msg/golang-nuts/AoO3aivfA_E/zFjhu8XvngMJ


This could be considered sacrilege, given some commentary on golang-nuts points to a desire for
`sync/atomic` to be used less

- counters and gauges require no lock
- histograms have grouped data so they require a sync.RWMutex BUT that's kept after the response has been sent

in this case im referring to the metrics package in rend
everything in there is atomic or has a very short critical section
reactive only - metrics endpoint does work only

## Pools are OK

If you *know* you will need billions (or trillions) of a reusable item, then you might want to
consider using a `sync.Pool` to pool objects. This may or may not be worth it to you depending on
your performance requirements. Another detail that took me a while to understand is that every
instance of `sync.Pool` is [cleared at every garbage collection](https://golang.org/src/sync/pool.go#L187),
meaning that your pool is only good between GCs (if the line number changes in the future,
look for `func poolCleanup()` in `sync/pool.go`).
