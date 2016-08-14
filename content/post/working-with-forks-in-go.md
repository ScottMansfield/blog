+++
title = "Working With Forks in Go"
date = "2016-06-18T21:17:27-07:00"
categories = ["Go", "Git"]
keywords = ["golang", "git", "github", "open source"]
+++

This been written about before in the Go universe, but I felt it was worth
reviewing as it has come up many times in the [Go slack](https://gophers.slack.c
om/) ([invite link](https://invite.slack.golangbridge.org/)).

When developing on a separate remote from the original in Go, there is a common
point of confusion when a developer forks on Github (for example) and then
proceeds to `go get` the fork. This places the fork in the wrong spot in their
`$GOPATH` and none of the imports work. This post will detail how to work with a
fork, or really any repository where the working remote is different than the
original, `go get`able remote.

## Setup

Say I wanted to work on a pull request for the [netns](https://github.com/jfraze
lle/netns) repository from [Jessie Frazelle](https://blog.jessfraz.com/) and I
had imported a package of it into another project. If I download my fork with
`go get`, the repository will be in the wrong place on my system at 
`$GOPATH/src/github.com/ScottMansfield/netns` and not in the proper place at
`$GOPATH/src/github.com/jfrazelle/netns`.

## Solution

Follow the steps below to properly develop on a fork. The example uses the same
repository as above, though it generally applies to any repository. This set of
instructions is optimized for git workflow during development.

1. Pull *the original package* from the canonical place with the standard
`go get` command
```
go get github.com/jfrazelle/netns
```

1. Fork the repository on Github or set up whatever other remote git repo you
will be using. In this case, I would go to Github and fork the repository.

1. Navigate to the top level of the repository on your computer. Note that this
might not be the specific package you're using.
```
cd $GOPATH/src/github.com/jfrazelle/netns
```

1. Rename the current `origin` remote to `upstream`
```
git remote rename origin upstream
```

1. Add your fork as `origin`
```
git remote add origin https://github.com/ScottMansfield/netns
```

At this point I'm ready to develop, branch, push, and everything else I would
normally do with my code, but with the package names of the other repository.
From here, the best approach would be to create a feature branch and work from
there.

```
git checkout -t -b awesome-feature
// create awesome feature commits
git push origin awesome-feature
```

Don't forget to pull the changes from upstream before sending in a PR. This
helps avoid merge conflicts.

```
git fetch upstream
git merge upstream/master
``` 

## Alternative Remote Names

If you are not as comfortable with using git commands directly on your go repo
to pull in the latest upstream changes, you can instead add your fork as a
remote named after your username:

```
git remote add ScottMansfield https://github.com/ScottMansfield/netns
```

Then you would work on branches on the username remote instead of `origin`. This
preserves the behavior of `go get` but is a slightly more awkward git workflow.
You would be working on a remote not called `origin` and the `go get -u` command
might end up causing merge conflicts if you were developing on the `master`
branch. This is why feature branches are so important.

## Not so bad

This is a common pain point with new open source contributors who want to be
able to start working on Go projects. Hopefully this helps with someone looking
to get started with their first OSS contributions.

If this post was helpful, please leave a comment below or let me know
[@sgmansfield](https://twitter.com/sgmansfield).