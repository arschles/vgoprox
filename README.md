# Welcome To Athens, Gophers!

![Athens Banner](./docs/static/banner.png)

[![Build Status](https://cloud.drone.io/api/badges/gomods/athens/status.svg)](https://cloud.drone.io/gomods/athens)
[![GoDoc](https://pkg.go.dev/github.com/gomods/athens?status.svg)](https://pkg.go.dev/github.com/gomods/athens)
[![Go Report Card](https://goreportcard.com/badge/github.com/gomods/athens)](https://goreportcard.com/report/github.com/gomods/athens)
[![codecov](https://codecov.io/gh/gomods/athens/branch/master/graph/badge.svg)](https://codecov.io/gh/gomods/athens)
[![Docker Pulls](https://img.shields.io/docker/pulls/gomods/athens.svg?maxAge=604800)](https://hub.docker.com/r/gomods/athens/)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](http://makeapullrequest.com)
[![Open Source Helpers](https://www.codetriage.com/gomods/athens/badges/users.svg)](https://www.codetriage.com/gomods/athens)

Welcome to the Athens project! We are a proxy server for the [Go Modules download API](https://docs.gomods.io/intro/protocol/).

See our documentation site [https://docs.gomods.io](https://docs.gomods.io) for more details on the project.

# Project Status

Development teams at several companies are using Athens in their everyday development workflows.

This means that they are running their own Athens servers, hooking them up to their own storage backends (see [here](https://docs.gomods.io/configuration/storage/) for the list of storage backends that Athens supports), and configuring their local Go development environments to use that Athens server.

We encourage you to [try it out](https://docs.gomods.io/install/), consider using it in your development workflow, and letting us know if you are using it by adding a comment to [this GitHub issue](https://github.com/gomods/athens/issues/1323).

# More Details Please!

The proxy implements the [Go modules download protocol](https://docs.gomods.io/intro/protocol/).

There is currently an **experimental** public proxy (at https://athens.azurefd.net) that you can use to try out Athens, without installing anything locally. We also have a community of folks using Athens in production inside their organizations as an internal proxy. In either deployment, developers set their `GOPROXY` environment variable to point to the Athens proxy of their choice. At that point, `go get`, `go build`, and more commands will use the proxy to download dependencies as necessary.

Athens proxies are highly configurable, so they can work for lots of different deployments. For example, proxies support a wide variety of storage drivers including:

- Cloud blob storage services
- MongoDB
- Content distribution networks (CDNs)
- Shared disk
- In-memory

# Development

See [DEVELOPMENT.md](./DEVELOPMENT.md) for details on how to set up your development environment and start contributing code.

Speaking of contributing, read on!

# Contributing

If you use Athens for your development workflow, we hope that you'll consider contributing back to the project. Athens is widely used and has plenty of interesting work to do, from technical challenges to technical documentation to release management. We have a wonderful community that we would love you to be a part of. [Absolutely everyone is welcome](https://arschles.com/blog/absolutely-everybody-is-welcome/).

The quickest way to get involved is by [filing issues](https://github.com/gomods/athens/issues/new/choose) if you find bugs or find that you need Athens to do something it doesn't.

If you'd like to help us tackle some of the technical / code challenges and you're familiar with the GitHub contribution process, you'll probably be familiar with our process for contributions. You can optionally find or submit an issue, and then submit a pull request (PR) to fix that issue. See [here](https://docs.gomods.io/contributing/) for more of the project-specific details.

>If you're not familiar with the standard GitHub contribution process, which Athens mostly follows, please see [this section of our documentation](https://docs.gomods.io/contributing/new/) to learn how to contribute. You can also take advantage of [@bketelsen](https://github.com/bketelsen)'s [great video](https://www.youtube.com/watch?v=bgSDcTyysRc) on how to contribute code. The information in these documents and videos will help you not only with this project, but can also help you contribute to many other projects on GitHub.

If you decide to to contribute but aren't sure what to work on, we have a well maintained [list of good first issues](https://github.com/gomods/athens/contribute) that you should look at. If you find one that you would like to work on, please post a comment saying "I want to work on this", and then it's all yours to begin working on.

>We do recommend that you choose one of the issues on the above list, but you may also consider a different one from our [entire list](https://github.com/gomods/athens/issues). Many of the issues on that list are more complex and challenging.

Before you do start getting involved or contributing, we want to let you know that we follow a general [philosophy](./PHILOSOPHY.md) in how we work together, and we'd really appreciate you getting familiar with it before you start.

It's not too long and it's ok for you to "skim" it (or even just read the first two sections :smile:), just as long as you understand the spirit of who we are and how we work.

# Getting Involved Without Contributing Pull Requests or Issues

If you're not ready to contribute code yet, there are plenty of other great ways to get involved:

- Come talk to us in the `#athens` channel in the [Gophers slack](http://gophers.slack.com/). We’re a really friendly group, so come say hi and join us! Ping me (`@arschles` on slack) in the channel and I’ll give you the lowdown
- Get familiar with the technology. There's lots to read about. Here are some places to start:
    - [Gentle Introduction to the Project](https://medium.com/@arschles/project-athens-c80606497ce1) - the basics of why we started this project
    - [The Download Protocol](https://medium.com/@arschles/project-athens-the-download-protocol-2b346926a818) - the core API that the proxy implements and the `go` CLI uses to download packages
    - [Proxy Design](https://docs.gomods.io/design/proxy/) - what the proxy is and how it works
    - [Go modules wiki](https://github.com/golang/go/wiki/Modules) - context and details on how Go dependency management works in general
    - ["Go and Versioning"](https://research.swtch.com/vgo) - long articles on Go dependency management details, internals, etc...

# Built on the Shoulders of Giants

The Athens project would not be possible without the amazing projects it builds on. Please see
[SHOULDERS.md](./SHOULDERS.md) to see a list of them.

# Coding Guidelines

We all strive to write nice and readable code which can be understood by every person of the team. To achieve that we follow principles described in Brian's talk `Code like the Go team`.

- [Printed version](https://www.brianketelsen.com/slides/gcru18-best/#1)
- [Gophercon RU talk](https://www.youtube.com/watch?v=MzTcsI6tn-0)

# Code of Conduct

This project follows the [Contributor Covenant](https://www.contributor-covenant.org/) (English version [here](./CODE_OF_CONDUCT.md)) code of conduct.

If you have concerns, notice a code of conduct violation, or otherwise would like to talk about something
related to this code of conduct, please reach out `@arschles` on the [Gophers Slack](https://gophers.slack.com/).

---
Athens banner attributed to Golda Manuel
