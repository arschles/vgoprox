---
title: "Intro"
date: 2018-12-07T11:44:36+00:00
---

<img src="/banner.png" width="600" alt="Athens Logo"/>

## Athens is a Proxy Server for Your Go Modules

Welcome, Gophers! Athens is a 100% open source proxy server for your public _and private_ Go modules. You should consider using it if:

- You are using [Go modules](https://github.com/golang/go/wiki/Modules) for dependencies in your project
- Your codebase is private

## Getting Started

There are many ways to [install](/install) Athens. This section shows the easiest way to get running.

>Make sure to remove the `$` on all of the commands in this section before you run them

### Run Athens

We recommend using Docker to run Athens for the first time. Do so with this command:

```console
$ docker run -p '3000:3000' gomods/athens:latest
```

You can run this command on any machine that supports Docker and that you can access over the network, including your local machine! 

### Set up `GOPROXY`

Next, set up your `GOPROXY` environment variable. This tells your `go` CLI to use the new Athens to get dependencies:

```console
$ export GOPROXY=http://localhost:3000
```

>If you're not running Athens on your local machine, change the above address to the machine you're running on

### Build Your App!

Make sure you're in a directory that has code you want to build. When you're there, you're ready to build:

```console
$ go build
```

## More Details on Athens

Let's look under the covers of the demo we just did.

When you started it up, Athens started serving the [standard Go modules download API](https://docs.gomods.io/intro/protocol/). This API is compatible with any Go version 1.9 or above (we recommend using the latest stable release, however!).

As long as you set your `GOPROXY` environment variable to the Athens server host and post, you'll automatically be using Athens to do your builds.

### Why You'd Run Your Own Server

You might _need_ to run your own module server if your app is private and needs to depend on private code. Since you control your own Athens installation, you can configure it to fetch code from both public sources and your private repositories alike.

Although the hosted solutions like `proxy.golang.org` or `gocenter.io` are convenient, you may _want_ to run your own Athens server if you:

- Can't access them
    - For example, you work at a company that has policies against using external sources to fetch code
- Are uncomfortable using a closed source, hosted solution

There are several more technical reasons why Athens might be a good choice for you. Read our [Why It Matters](/intro/why) document for more.

### Other Options for Using Athens

Athens is easy to run yourself. Here are the most popular options:

- You can run it as a [Docker](https://www.docker.com/) image (see [here](./install/shared-team-instance/) for how to do that)
- You can run it on [Kubernetes](https://kubernetes.io) (see [here](./install/install-on-kubernetes/) for how to do that)

We also run an experimental version of Athens at `https://athens.azurefd.net`. If you set your `GOPROXY` environment variable to that address, you can get started without installing anything.

>This is not a production-ready proxy deployment, though. Please deploy your own Athens instance for your own production usage.

**[Like what you hear? Try Athens Now!](/try-out)**

## Get Involved

Here are some other ways to get involved:

- Read the full [walkthrough](/walkthrough) with setting up, running and testing the Athens proxy
explores this in greater depth.
* Join our [office hours](/contributing/community/office-hours/)! It's a great way to meet folks working on the project, ask questions or just hang out. All are welcome to join and participate.
* Check out our issue queue for [good first issues](https://github.com/gomods/athens/issues?q=is%3Aopen+is%3Aissue+label%3A%22good+first+issue%22)
* Join us in the `#athens` channel on the [Gophers Slack](https://invite.slack.golangbridge.org/)

---
Athens banner attributed to Golda Manuel
