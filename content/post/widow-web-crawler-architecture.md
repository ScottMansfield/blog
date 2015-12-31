+++
title = "Widow: Web Crawler Architecture"
date = "2015-12-18T00:09:52-08:00"
categories = ["Java", "Web Crawler", "Widow", "AWS"]
keywords = ["Widow", "web crawler", "web", "crawler", "aws", "Amazon Web Services"]
+++

## Widow

In a [previous post]({{< relref "why-i-decided-to-make-my-own-web-crawler.md" >}}), I went over the
justification for building my own web crawler named [Widow](https://github.com/ScottMansfield/widow).
Here I will explain my alternative method for building a large-scale web crawler. Keep in mind that
the crawler is still a work in progress (as of the end of 2015) so this is not final. There is still
some future work to be done, which will be discussed at the end.

## Architecture Overview

![arch](/img/2015/12/high_level_architecture.svg)

There's three main stages to the online crawler. The first stage is the fetcher, to pull in data,
second is the parser, which pulls important information out of the data, and last is the indexer,
which inserts the data into a database or possibly multiple databases. Offline there is an analysis
server and a UI for end-users to pull, inspect, and report on data gathered by the crawler.

The input to each stage is an SQS queue. The queues hold messages with metadata about each page as
it flows through the system. S3 acts as intermediate blob storage for webpages or other files that
don't fit into the 256k limit of SQS. Dynamo is the current index, however any key-value store could
be plugged in, such as Cassandra, or even an RDBMS like MySQL or Aurora. The different stages are
described in more detail below.

## Fetch

Fetching is a pretty straightforward part of the system. It will reach out to the origin web server
and retireve whatever web resource needed. It then uploads the resource itself to S3 and then sends
the metadata about the resource to the parsing stage. The concept is straightforward, however the
implementation is more complex than it seems at first. How does it handle failures? What about rate
limiting? Sitemaps? The robots.txt standard-but-not-a-standard? These considerations all need to be
built in.

## Parse

The parse stage currently uses [jsoup](http://jsoup.org/) to parse HTML. It is the only supported
type of web resource that Widow can parse at the moment, however the architecture permits changes in
one place to allow broader functionality of the whole. When the parse stage can understand PDF, then
the fetch stage can start pulling information linked from PDFs, and the index stage will start to
index PDF metadata. There is a possibility that I use [Apache Tika](https://tika.apache.org/) in the
future, but not for now. It might be best to use projects that specialize in one format in order to
guard against the "kitchen sink" fallacy, where everything is done adequately instead of doing one
thing very well. Future formats also include [OOXML](https://en.wikipedia.org/wiki/Office_Open_XML)
for Microsoft Word, Excel, and PowerPoint files or [ODF](https://en.wikipedia.org/wiki/OpenDocument)
for OpenOffice files.

## Index

The indexing stage is very straightforward. It takes the metadata from the queue and inserts it into
an index or multiple indices. The stage exists as a separate entity because it might be necessary to
apply some logic for structuring the data or transformations prior to inserting to allow easier
retrieval later. As well, the plan is to have the ability to insert into multiple indexes, such as a
short-term real-time analysis storage and a longer-term archival storage. The current implementation
only inserts into DynamoDB, however an interface is being built to allow multiple indexing
locations. The semantics of the index are yet to be worked out, though there will need to be some
commonality between a key-value store (DynamoDB, Memcached, Redis), a columnar store (Cassandra,
Parquet), or a traditional RDBMS (MySQL, Aurora, Postgres).

## Analyze

The analyze project itself is comprised of two sub-projects. First is the server that retrieves
crawl information from the metadata index, and second is the web UI itself that is a window into
that data.

### Analysis Server

The analysis server is a REST service responsible for retrieving and semantically making
sense of the data, regardless of the datastore it resides in. The same abstraction used in the
indexing stage over different database types can be used here to provide the same ease of use for
different users of Widow. There is no offline or batch processing of data. It is retrieved and
processed as needed by the UI. The same process serves the files for the UI.

### Analysis UI

The UI is an [AngularJS](https://angularjs.org/) application that is bundled into the server. It is
built alongside the server code and bundled into the same JAR. This area is probably the most
lacking as of this post, however the goal is to become a full-fledged analysis tool for the data
retrieved. Analysis means many different things to many different people, so flexibility is key.
Several views will be created for connectedness, where content analysis can be performed by people
managing website content. For example, disconnected graphs can be discovered. Another view will be
for the engineers running the web servers, and will include metrics about website performance, such
as request latency, gzip usage, and total page size over time.

## Auto-scaling

To power each stage is an Auto-Scaling Group in Amazon, which will automatically scale up and down
based on the queue size and other parameters. This means that each stage can independently scale as
needed to meet demand. I expect the parsing stage to have much more than the fetch or index, and
that with a cap on the parsing stage, the fetch and index stages would reach an equilibrium at a
smaller size. Also, if crawling a relatively slow site, the fetch stage can scale up to increase
parallelism (assuming the rate limits are not being hit) and balance out the system. Ideally, no one
stage is the bottleneck.

## Future Work

The crawler itself is not complete, and there's a few big pieces left, all concerning being a good
netizen:

* Global rate limiting to prevent overloading
* Respecting `robots.txt` (assisted by the [Terminator](https://github.com/ScottMansfield/terminator)
sub-project)
* Using `sitemap.xml` as a starting point for crawling (assisted by the [Exo](https://github.com/Sco
ttMansfield/exo) sub-project)

The challenge in particular is the sharing of information between different servers running the same
stage. The parse servers should not have to retrieve the `robots.txt` independently, nor should they
all be parsing each `sitemap.xml`. The solution is a cache that all can use to store the
intermediate form of parsed data. This relies on Terminator and Exo becoming more mature.

As well, the history of visited pages is currently a binary yes/no as to whether a particular box
has seen it before. This is obviously not correct if the crawler was to work as a whole. This
information needs to be shared with yet another cache, however just talking to a single instance of
Redis slowed down crawling dramatically. A fast, distributed cache like [EVCache](https://github.com
/Netflix/EVCache) might be better in this case to store metadata. If persistence is required, then
snapshots will be taken and inserted into a database. This balances speed and availability with
durability.

## Github Repos

All work on Widow is done in public, and the system is completely open source. Feel free to peruse:

* [Widow](https://github.com/ScottMansfield/widow)
* [Terminator](https://github.com/ScottMansfield/terminator)
* [Exo](https://github.com/ScottMansfield/exo)
