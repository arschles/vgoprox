# Contributing to Athens
Hurray! We are glad that you want to contribute to our project! 👍

If this is your first contribution, not to worry! We have a great [tutorial](https://www.youtube.com/watch?v=bgSDcTyysRc) to help you get started, and you can always ask us for help in the `#athens` channel in the [gopher slack](https://invite.slack.golangbridge.org/). We'll give you whatever guidance you need. Another great resource for first time contributors can be found [here](https://github.com/firstcontributions/first-contributions/blob/master/README.md).

## Claiming an issue
If you see an issue that you'd like to work on, please just post a comment saying that you want to work on it. Something like "I want to work on this" is fine.

## Verify your work
Run `make verify test-unit test-e2e` to run all the same validations that our CI process runs, such
as checking that the standard go formatting is applied, linting, etc.

## Setup your dev environment

Run `make setup-dev-env` to install local developer tools and run necessary
services, such as mongodb, for the end-to-end tests.

## Unit Tests
For further details see [DEVELOPMENT.md](DEVELOPMENT.md#L84)

## End-to-End Tests
End-to-End tests (e2e) are tests from the user perspective that validate that
everything works when running real live servers, and using `go` with GOPROXY set.

Run `make test-e2e` to run the end-to-end tests.

The first time you run the tests,
you must run `make setup-dev-env` first, otherwise you will see errors like the one below:

```
error connecting to storage (no reachable servers)
```

## Helm Chart

This repository comes with a [Helm](https://helm.sh) [chart](https://github.com/gomods/athens/tree/master/charts/athens-proxy) to make it easier for anyone to deploy their own instance of Athens to their Kubernetes cluster.

Our CI/CD system will look to ensure that the `version` field in [Chart.yaml](https://github.com/gomods/athens/blob/master/charts/athens-proxy/Chart.yaml) is updated if any part of the chart is changed. We do that to make sure that there is a new version for every change to the chart.

_If you're planning to submit a pull request (PR) that updates any part of the Helm chart, please make sure you _increase_ the version number in that field._

>Version numbers follow the [semantic versioning convention](https://semver.org/).
>If you're not sure which part of the version number to change, increment the PATCH one.

Also, please keep in mind that there may be other PRs open that update the chart, so we may need to ask you to change that field one or more times if other changes to the chart are merged.

We want our Helm chart to stay up to date, but we're going to work as hard as we can to help you get your pull request merged!

## Next Steps

After you get your code working, submit a Pull Request (PR) following 
[Github's PR model](https://help.github.com/articles/about-pull-requests/).

If you're interested, take a look at [REVIEWS.md](REVIEWS.md) to learn how
your PR will be reviewed and how you can help get it merged.
