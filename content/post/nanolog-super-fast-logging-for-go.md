+++
title = "Nanolog: Super Fast Logging for Go"
date = "2017-04-28T00:00:00-07:00"
categories = ["Go", "Performance", "Logging"]
keywords = ["nanolog", "go", "golang", "performance", "logging"]

+++

I've released a new package called [nanolog](https://github.com/ScottMansfield/nanolog)!

Nanolog is super fast, as low as 70ns to log a line, and works with concurrent loggers. It uses a
somewhat different model than most people are used to for the API; you can think of it like prepared
statements for logging. It is currently at version 0.1.0.

This package was inspired by a [project of the same name](https://github.com/PlatformLab/NanoLog)
for C++.

## Why?

Because.

I actually don't have a driving use case where I need logging to be as fast as possible. I am also
not trying to thumb my nose at other loggers (Uber's [Zap](https://github.com/uber-go/zap) has
gotten a lot of attention for its speed). This was more of a technical itch that I just *had* to
scratch.

And, just for bragging rights, this logger actually is 0 allocs, except for the conversion to an
interface to pass in the arguments.

```go
func BenchmarkLogSequential(b *testing.B) {
	w = bufio.NewWriter(ioutil.Discard)
	h := AddLogger("foo thing bar thing %i64. Fubar %s foo. sadfasdf %u32 sdfasfasdfasdffds %u32.")
	args := []interface{}{int64(1), "string", uint32(2), uint32(3)}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Log(h, args...)
	}
}
```

```
BenchmarkLogSequential-8       10000000       114 ns/op       0 B/op       0 allocs/op
```

## Usage

### At runtime (logging)

The following code is from the [test program](https://github.com/ScottMansfield/nanolog/tree/master/test)
in the package that tests out the package in a real program type scenario.

As I said above, the API and, actually, the inner workings are very similar to the way a database
has prepared statements where only the changing bits get sent. In any file, log lines can be created
like so:

```go
var (
    logWorking = nanolog.AddLogger("Worker %u8, working on task %i, attempt %i.")
)
```

Well, that looks a little different.

Nanolog has its own syntax for the format strings, but for good reason. It uses the interpolation
tokens (e.g. `%u16`) to map that gap in the string to a specific `reflect.Kind`. This is what
enables it to be so fast; it is able to write just the data that changes in each log line and knows
what types come out on the other end.

The `nanolog.AddLogger` method returns a `nanolog.Handle` to internal data structures that hold the
metadata about that particular log line. New log lines can be created at any time, but normally,
this should be at program init and happen only once. As log lines are created, their metadata is
serialized into the output stream for the logger.

In the main file of the program, the main function should set up the nanolog package with an
`io.Writer` that is set to write to the stream you want as your log data. This could be `os.Stdout`,
a file handle, or any other implementation of `io.Writer`. Ideally, this is a raw stream and not
intermediated.

```go
var (
	logWorking nanolog.Handle
)

func init() {
	logWorking = nanolog.AddLogger("Worker %u8, working on task %i, attempt %i.")

	// Set up nanolog writer
	nanologout, err := os.Create("foo.log")
	if err != nil {
		panic(err)
	}
	nanolog.SetWriter(nanologout)
}

func main() {
    //...
}
```

When a new writer is set, nanolog will flush the data it has stored up and switch to the new stream.
At initialization time, this data is actually all stored in memory, which is why `main` should set
up the I/O as soon as it starts. The init data will get written to the first writer passed in.

This also enables a program to possibly do log rotation by replacing the writer every so often. This
would be a fun secondary project to do as a wrapper package. It would, however, lose the "context"
for determining which log line matches which actual log entry. I don't have a solution for this as
of yet.

Later on, of course, you'll want to actually log a line during the course of your program's
execution:

```go
// id, i, and j are in context here. See example code.
nanolog.Log(logWorking, id, i, j)
```

This line will serialize the log line to the internal buffer and may write to the underlying writer.
Eventually, your program will want to shut down. The nanolog package currently does not have a way
to detect when the program is shutting down, so if you need the last log lines to be written to the
log before the program exits (which you normally would), you need to ensure that you call
`nanolog.Flush()` immediately before exiting.

```go
func main() {
    //...

    nanolog.Flush()
}
```

### After runtime (reading the logs)

The log itelf is a binary format that is not human-readable. These files, especially for programs
that have longer log lines, will be much smaller than the typical log file. In order to read the
logs, they need to be "inflated." (Side note: if anyone has a better term for this, let me know.)
This requires using the `github.com/ScottMansfield/nanolog/cmd/inflate` command on the file. The
output files are self-contained and are append-able, meaning you don't need any additional metadata
and you can append to the same file and the `inflate` program will work just fine.

In the examples given above, if would be:

```
$ go build github.com/ScottMansfield/nanolog/cmd/inflate
$ ./inflate -f foo.log
```

The `inflate` command will dump the inflated log to its standard output, so you can direct it to a
file, `grep` through it, or do anything else you want to.

### Caveats

Because of the way the logger is built, there is a maximum number of log lines that can be created
with `AddLogger`. This number is 10240, however it can be changed by simply changing the
`MaxLoggers` constant in the package if you have a vendored version. There's no real technical
reason it couldn't be higher, 10240 just seemed like a reasonable upper bound.

## Future Work

At this point, most of the time is spent in the lock and unlock of the mutex protecting the writes.
I'd like to see if I can get a multiple-producer single-consumer blocking ring buffer set up such
that both sides can operate independently and both sides can scale better. This might end up being
assisted by sharding these buffers to reduce contention on the atomic ints, or I may be barking up
the wrong tree :).

There's a few more things I could do, such as:

* Add support for types that implement the [`fmt.Stringer`](https://golang.org/pkg/fmt/#Stringer)
interface such that it can log anything that has a string representation.
* Support arrays / slices of a particular type. This would need some kind of annotation like `%as`
to signal an array of strings, for example.
* Some ability to detect shutdown of the program such that the writer can be flushed.
* A log rotation feature / wrapper package that would allow logs to be rotated without losing their
context such that every log file is standalone and inflatable.

Of course, PR's are welcome. I'll be trying to make sure that I use issues to capture ideas so
others can follow along (if anyone were so inclined).

## Links

* [Github Repo](https://github.com/ScottMansfield/nanolog)
* [Godoc](https://godoc.org/github.com/ScottMansfield/nanolog)

Please let me know what you thought of this article in the comments below or
[@sgmansfield](https://twitter.com/sgmansfield) on twitter. Thank you for stopping by!