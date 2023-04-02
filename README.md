# gons

[![Go Reference](https://pkg.go.dev/badge/godoc.org/github.com/TheDiveO/gons.svg)](https://pkg.go.dev/github.com/TheDiveO/gons)
[![GitHub](https://img.shields.io/github/license/thediveo/gons)](https://img.shields.io/github/license/thediveo/gons)
![build and test](https://github.com/TheDiveO/gons/workflows/build%20and%20test/badge.svg?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/thediveo/gons)](https://goreportcard.com/report/github.com/thediveo/gons)
![Coverage](https://img.shields.io/badge/Coverage-82.9%25-brightgreen)

## gons

`gons` ("go [*into*] namespaces") is a small Go package that selectively
switches your Go application into other already existing Linux namespaces.
This must happen before the Go runtime spins up, blocking certain namespace
changes, such as changing into a different mount namespace.

- `gons` switches the Go application *itself* it is linked to into a set of
  already existing namespaces, and only so at *startup*.
- `gons` does *neither fork nor re-execute* in order to switch namespaces.
- `gons` *does not* create new namespaces.

In consequence, `gons` is a package tackling a different usecase than
`runc/libcontainer`'s famous
[*nsenter*](https://github.com/opencontainers/runc/tree/master/libcontainer/nsenter)
package.

The existing namespaces to join/switch into are referenced by their paths in
the filesystem (such as `/proc/123456/ns/mnt`), and are specified using
environment variables. Set only the environment variables for those namespaces
that should be switched at startup. These variables need to be set before your
application is started. The names of the environment variables are as follows
and must be all lowercase:

- `gons_cgroup=...`
- `gons_ipc=...`
- `gons_mnt=...`
- `gons_net=...`
- `gons_pid=...` (*please note that this does not switch your application's
  own PID namespace, but rather controls the PID namespace any child processes
  of your application will be put into.*)
- `gons_user=...`
- `gons_uts=...`

Additionally, you can specify the order in which the namespaces should be
switched, as well as when the namespace paths are to be opened:

- if not overridden by the optional environment variable `gons_order=...`,
  then the default order is `!user,!mnt,!cgroup,!ipc,!net,!pid,!uts` (see
  below for the meaning of "`!`"). It's not necessary to specify all 7
  namespace types when you don't intend to switch them all. For instance, if
  you just switch the net and IPC namespaces, then `gons_order=net,ipc` is
  sufficient.
- when a namespace type name is preceded by a bang "`!`", such as `!user`,
  then the its path will be opened before the first namespace switch takes
  place. Without a bang, the namespace path is opened just right before
  switching into this namespace. This is mostly of importance when switching
  the mount namespace, as this can also change the filesystem and thus how the
  namespace paths are resolved.

> **Note:** if a given namespace path is invalid, or if there are insufficient
> rights to access the path or switch to the specified namespace, then an
> error message is stored which you need to pick up later in your application
> using `gons.Status()`. Please see the package documentation for details.

The `gons` package requires [cgo](https://golang.org/cmd/cgo/): the required
namespace switches can only safely be done while your application is still
single-threaded and that's only the case before the Go runtime is spinning up.

## gons/reexec

`gons/reexec` helps with forking and re-executing an application in order to
switch namespaces, run some action, sending back intelligence to the parent
application process, and then terminating the re-executed child. The parent
process (or rather: go routine) then continues, working on the intelligence
gathered.

A very simplistic example:

```go
package main

import (
  "fmt"
  "github.com/thediveo/gons/reexec"
)

func init() {
  reexec.Register("answertoeverything", func() {
    fmt.Fprintln(os.Stdout, `42`)
  })
}

func main() {
  var answer int
  reexec.RunReexecAction("answertoeverything", reexec.Result(&s))
  fmt.Printf("answer: %d\n", answer)
}
```

- `reexec.Register` registers an action with its name and the code to execute.
- `reexec.RunReexecAction` forks and re-executes itself as a child process,
  triggering the named action. It then picks up the result, which the action
  has to print to `os.Stdout` in JSON format, and prints the result.

## gons/reexec/testing

So you want to get code coverage data even across one or several
re-executions? Then you'll need to add a `TestMain` as follows:

```go
package foobar

import (
    "testing"

    rxtst "github.com/thediveo/gons/reexec/testing"
)

func TestMain(m *testing.M) {
    mm := &rxtst.M{M: m}
    os.Exit(mm.Run())
}
```

As you can see from this code above, `TestMain` takes the usual `m *testing.M`
parameter. But instead of directly calling `m.Run()` we have to wrap it into a
`txtst.M` instead, and call `Run()` only on the wrapper. This wrapper contains
the magic to cause re-executed actions to write coverage profile data and to
merge it with the main process' coverage profile data.

## Copyright and License

`gons` is Copyright 2019-23 Harald Albrecht, and licensed under the Apache
License, Version 2.0.
