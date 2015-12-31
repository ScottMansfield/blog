+++
title = "The Insanity of Generating All Possible GUIDs"
date = "2015-12-26T08:46:53-08:00"
categories = ["Computation", "Algorithms", "Parallelism"]
keywords = ["Stack Overflow", "GUID"]
+++

## Genesis

There was once a StackOverflow [question](http://stackoverflow.com/questions/10029651/fastest-way-in
-c-sharp-to-iterate-through-all-guids-possible) ("was" because it's gone now) title "Fastest way in
C# to iterate through ALL Guids Possible". The premise was that the person wanted to hit a web
server with a single request per GUID to determine if the server had data for it.

Hilarity ensued.

## Suggestions

Some people were legitimately trying to be helpful in this quixotic quest. Some pointed out:

Floating point math mistakes:

> 5316911983139663491615228241121399999d==(5316911983139663491615228241121399999d+‌​1d) is true. Your
loop is going to behave unexpectedly.

The other end mattered too:

> There are so many combinations, and given the lag that you are going to encounter hitting a web
site, I doubt that you will be able to solve it this way, and get responses in any reasonable time
frame.

The choice of GUIDs might be on purpose:

> The fact that the server owners chose to identify their content with guids is an indication that
they do not want you to be able to solve this problem.

The GUIDs might be limited in scope:

>If you look at the [GUID spec](http://www.ietf.org/rfc/rfc4122.txt) you will see that they are
constructed from specific sub-fields. If you can take a look at some of the GUIDs in your data, you
might be able to get an idea of what those sub-fields contain, and you should be able to shorten
your search time.

And that the legal implications might be dire:

> ...you are attempting to mount an attack against a server. I'm trying to talk you out of doing
that, because it is (1) success is impossible, (2) it is morally wrong to even try, and (3) it is
probably illegal to try.

## Math is Hard

And some people used math to prove, though the original poster (OP) was persistent:

Simple math to start, but it was unconvincing:

> If I am not wrong, with 1M/s request it would take 10790283070806014188970529 years

What's a quadrillion? Whatever it is, it doesn't sound large enough to matter:

> You cannot accomplish this task. (Unless there's something you haven't told us. For example, if
you're only dealing with a known, small-ish subset of all possible guids.) Even if you could process
a quadrillion guids per second, it'll still take you almost 11 quadrillion years to get through all
the possibilities

A time limit less than a few quadrillion years might require some more horsepower:

> What's the longest acceptable duration for this task? A minute? An hour? A day, week, month, year?
Let's assume it's a year. If so, you would have to generate and check
10,790,283,070,806,014,188,970,529,155 guids each millisecond.

Counting from 0 to 2^32 - 1 didn't take that long, maybe 2^128 is only, like, 4 times as long?

> See how long this takes to run: `uint x = 0; do { x++; } while (x != 0);`. That was 32 bits; next,
time the 64-bit version: `ulong x = 0; do { x++; } while (x != 0);` How long does that take to run
on your "one of the 50 fastest computers in the world"? It should be about 4 billion times longer
than the 32-bit version. The analogous loop with 96 bits should take 4 billion times longer than
that, and with 128 bits, another 4 billion times longer.

Folding paper more than 7 times is a challenge, but 128 times is ***really*** hard

> Try a spatial analogy: an astronomically large sheet of paper, 0.1 mm thick. Fold it 32 times:
it's now nearly 430 km thick, the distance from New York City to Washington, DC. Fold it 32 more
times; it's now about 0.2 light-years thick, well beyond the outer reaches of the solar system.
Fold it 32 more times; it's 837,460,949 light years thick (roughly 8000 times the diameter of the
milky way galaxy). Fold it 32 more times... wait, you probably can't, because at the 6th additional
fold, it's thickness is larger than the size of the observable universe.

Google might be able to help:

> I just estimated that if Google used all of their servers to attempt this, they could complete it
in as little as 24,874,442,000,000,000,000,000,000,000 years!

But your desktop... probably not:

> Let's say you have a 3 GHz machine. That's 3e+9 clock cycles every second. You have 2^128
iterations to go through. If we were to trim your code down to 1 clock cycle per iteration
(impossible btw), it would take (2^128 / 3e9) = 1.13e+29 seconds to complete. That is 3.15e+25
hours, and 3.59e+21 years. Better get started soon--you're going to be staring at your console for a
LONG time.

Let's destroy entire planets for the enterprise:

> Even if you could generate the GUIDs infinitely fast, your output file is going to be 2^128*16 =
2^132 bytes in size. That is around 10^27 terabytes. One terabyte of storage weighs around 500
grams. The mass of the earth is 10^24 kilograms, so before you run this program, you will need to
acquire 500 earths and convert them all to hard drives.

There's that "quadrillion" again:

> Even if you could process a quadrillion guids per second, it'll still take you almost 11
quadrillion years to get through all the possibilities.

The age of the universe was no matter:

> Suppose... that you can process a quadrillion guids a second per thread, and you have a massively
parallel environment allowing you to run one billion threads at the same time. Your program would
still take 11,000 years to run. Using Stargazer712's more likely figures, with, say 100,000,000
processors in parallel, and you're still looking at 4e+13 years, nearly 3000 times longer than the
age of the universe (1.4e+10: [en.wikipedia.org/wiki/Age_of_the_universe](en.wikipedia.org/wiki/Age_
of_the_universe)).

## Did he ever succeed?

We will probably never know. There was, luckily, a copy of the question archived by another website
that I managed to save before that copy was taken down. It is now hosted over on my brother's
website for all to bask in its glory. [Take a look](http://allguids.cdmansfield.com/Fastest%20way%20
in%20C%23%20to%20iterate%20through%20ALL%20Guids%20Possible.htm) and see how the story unfolded for
yourself.
