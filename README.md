# automaxprocs [![GoDoc][doc-img]][doc] [![Build Status][ci-img]][ci] [![Coverage Status][cov-img]][cov]

# automaxprocs

Importing the base package will automatically set `GOMAXPROCS` to match Linux container
CPU quota.

## Quick Start

To automatically set `GOMAXPROCS`:
```go
import _ "go.uber.org/automaxprocs"

func main() {
  // Your application logic here.
}
```

To get the current CPU quota
```go
import "go.uber.org/automaxprocs/cpuquota"

allCGroups, err := cpuquota.NewCGroupsForCurrentProcess()
if err != nil {
	// handle err
}
quota, defined, err := allCGroups.CPUQuota()
if !defined || err != nil {
	// handle err
}

fmt.Println("quota:", quota)
```

## Development Status: Stable

All APIs are finalized, and no breaking changes will be made in the 1.x series
of releases. Users of semver-aware dependency management systems should pin
automaxprocs to `^1`.

## Contributing

We encourage and support an active, healthy community of contributors &mdash;
including you! Details are in the [contribution guide](CONTRIBUTING.md) and
the [code of conduct](CODE_OF_CONDUCT.md). The automaxprocs maintainers keep
an eye on issues and pull requests, but you can also report any negative
conduct to oss-conduct@uber.com. That email list is a private, safe space;
even the automaxprocs maintainers don't have access, so don't hesitate to hold
us to a high standard.

<hr>

Released under the [MIT License](LICENSE).

[doc-img]: https://godoc.org/go.uber.org/automaxprocs?status.svg
[doc]: https://godoc.org/go.uber.org/automaxprocs
[ci-img]: https://travis-ci.com/uber-go/automaxprocs.svg?branch=master
[ci]: https://travis-ci.com/uber-go/automaxprocs
[cov-img]: https://codecov.io/gh/uber-go/automaxprocs/branch/master/graph/badge.svg
[cov]: https://codecov.io/gh/uber-go/automaxprocs
