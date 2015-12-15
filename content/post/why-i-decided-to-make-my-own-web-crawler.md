+++
title = "Why I Decided to Make My Own Web Crawler"
date = "2015-12-11T00:39:54-08:00"
categories = ["Java", "Web Crawler", "Widow", "AWS"]
keywords = ["Widow", "web crawler", "web", "crawler", "Apache", "Nutch", "Heritrix", "archive.org",
"aws", "Amazon Web Services"]
+++

## Widow

The web crawler I am making is named Widow, and is freely available [on GitHub](https://github.com/S
cottMansfield/widow). Along with Widow, there are a couple of other sub-projects that were, in my
mind, necessary to have a decent crawler. It must also be able to parse the `robots.txt` and
`sitemap.xml` files that websites use to direct web crawler behavior. The projects to perform both
of those functions are [terminator](https://github.com/ScottMansfield/terminator) and [exo](https://
github.com/ScottMansfield/exo). They are also available on Maven Central under the
[com.widowcrawler](https://repo1.maven.org/maven2/com/widowcrawler/) namespace. A future post will
describe the crawler in a lot more detail.

## Why build one?

The two main open source projects that existed before I started were [Heritrix](http://crawler.archi
ve.org) from [archive.org](https://archive.org/) and [Apache Nutch](http://nutch.apache.org/). When
Widow was conceived in 2013, the projects  did not fit what I thought a real, distributed,
high-scale web crawler could be. Both have been in development for quite a long time, and have very
clearly defined areas of use. There is also a third, much older project named [Grub](http://sourcefo
rge.net/projects/grub/). These comparisons are not meant to be dismissive, but are meant to show the
other projects in the field.

### Heritrix

Heritrix is used by archive.org to pull down content for archiving. The goal of the project is to
retrieve content in archival quality and keep it for archival use. It runs on one machine in one JVM
and is not able to scale beyond one java process and heap. The typical use case is to pull down a
whole website, so most of the time this will be sufficient. For large websites, I assume they get a
bigger box.

This architecture eliminates a whole class of problems that a distributed crawler has, including
configuration management and tracking how good of an internet citizen the crawler is being. Rate
limiting is hard to begin with, so keeping it all in one JVM helps. As well, there's no network
overhead simply to function, and nearly any box will do as long as it can hold your data.

The drawback is, of course, that it is limited by how much a single JVM can achieve. Given the
characteristics of a web crawler (I/O bound sometimes, CPU bound others) it's possible to achive a
a fair amount of work in one process. The amount is still not enough to be a truly large-scale
crawler because true large scale implies more than one server's worth of work for a reasonable time
span. Many hands make light work.

### Apache Nutch

Nutch is a project which actually has similar goals to Widow. There are still new releases coming
out, so the project is definitely not dead. At first, I thought it was another single-box crawler,
because their [FAQ page](http://wiki.apache.org/nutch/FAQ#Will_Nutch_use_a_distributed_crawler.2C_li
ke_Grub.3F) states that they are not considering going to a distributed model. It took me a while to
find the [Hadoop instructions](https://wiki.apache.org/nutch/NutchHadoopTutorial), which tell how to
run Nutch in a distributed fashion on top of Hadoop and HDFS.

There is a 1.x branch and a 2.x branch, with the latter using more Apache projects to decrease its
dependence on specific technologies (e.g. Hadoop). The 2.x branch includes Apache Tika for parsing
of the many file formats and Apache Gora for abstracting the persistence layer. Tika is an awesome
project because there's so much specialized knowledge about so many formats that is concentrated in
one place. The Gora project I am not entirely convinced is a good idea, mostly because projects that
try to abstract the data layer too much either hide any advantage of one data system over another,
so all of them are as good as the least capable one, or require you to write a lot of glue code
behind them anyway, and thus become a leaky and almost useless as an abstraction.

In all, the project is still moving along, but documentation is sparse, with many pages being marked
(on the wiki) as old a few years ago. Updates seem to never happen even if development is. What does
seem apparent is that the project updates mostly consist of bugfixes, and no new technologies are
being tried. For example, many [SPA](https://en.wikipedia.org/wiki/Single-page_application) websites
are not indexable unless you can run a headless browser and run the entire javascript payload. Only
then can it be indexed properly.

The final gripe I have is the attachment to Hadoop. The 2.x branch changed this to Gora, but that,
like I said before, I am not entirely convinced is a worthwhile exercise. Hadoop is a solid
technology, but is an operational overhead that I am not willing to take on in order to run a
distributed crawler. AWS has many of the distributed fundamentals exposed as a service already, so
taking advantage of that is better than running on Hadoop. 

### Grub

The Grub project seems to have very similar goals to Widow, but the last commit was in March 2002.
The documentation, if it did exist at some point, has been lost to the sands of time. The only
remaining text is

> Grub is a distributed internet crawler/indexer designed to run on multi-platform systems,
interfacing with a central server/database.

and the website grub.org only shows the default "It works!" page. I think it's safe to say Grub is
dead.

As well, the project began and ended before cloud computing was a real product. Amazon released its
first web service (SQS, or Simple Queue Service) as generally available [in 2006](https://aws.amazon
.com/blogs/aws/amazon_simple_q/) after a preview period of about a year and a half. The preview
started after the last public release of Grub.

## The New Hotness

When starting a new project, one has to consider the technology landscape as it exists now. I chose:

* Java as the language
* Gradle for building
* AWS as a **hard** dependency

The language is nothing new, but is different than Grub, which is written in C++. I chose to start
with Java 8 as a requirement because there's no reason to run in even Java 7 anymore, except for
inertia. It is my belief that anyone who decides to use Widow will be free enough to run Java 8.

Build systems can make or break a project because they are how the developer interacts with the
software. Gradle is a robust build and dependency management system that is much easier to deal with
than Ivy / Ant. The benefits are well spelled out elsewhere, but suffice it to say it's [better](htt
p://gradle.org/maven_vs_gradle/) than the alternatives.

Amazon Web Services is an awesome platform for building distributed applications, so building a
distributed web crawler on top of that makes sense given the hard parts are mostly done for you.
Widow uses SQS and S3 extensively for passing data around between different stages, which allows me
to keep my mind on the application itself, and not building a distributed queue. Given my earlier
sentiments, it should be no surprise that I don't find the exercise of trying to abstract S3 or SQS
out useful, given that it's unlikely I would want to run it anywhere else. As well, many places run
OpenStack as an alternative cloud infrastructure, which has a compatible API for object storage.

## Next

Next up will be a post about the design and internals of Widow, which will go into more detail on
how data flows through the system. It will also talk about the software itself; what pieces perform
what functions and how the whole thing fits together.
