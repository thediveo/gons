# gons

[![GoDoc](https://godoc.org/github.com/TheDiveO/gons?status.svg)](http://godoc.org/github.com/TheDiveO/gons)
[![GitHub](https://img.shields.io/github/license/thediveo/gons)](https://img.shields.io/github/license/thediveo/gons)
![build and test](https://github.com/TheDiveO/gons/workflows/build%20and%20test/badge.svg?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/thediveo/gons)](https://goreportcard.com/report/github.com/thediveo/gons)

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

## Copyright and License

`gons` is Copyright 2019 Harald Albrecht, and licensed under the Apache
License, Version 2.0.
