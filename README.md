# gons

[![GoDoc](https://godoc.org/github.com/TheDiveO/gons?status.svg)](http://godoc.org/github.com/TheDiveO/gons)

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

- `cgroupns=...`
- `ipcns=...`
- `mntns=...`
- `netns=...`
- `pidns=...` (*please note that this does not switch your application's own
  PID namespace, but rather controls the PID namespace any child processes of
  your application will be put into.*)
- `userns=...`
- `utsns=...`

> **Note:** if a given namespace path is invalid, or if there are insufficient
> rights to access the path or switch to the specified namespace, then an
> error message is stored which you need to pick up later in your application
> using `gons.Status()`. Please see the package documentation for details.

The `gons` package requires [cgo](https://golang.org/cmd/cgo/): the required
namespace switches can only safely be done while your application is still
single-threaded and that's only the case before the Go runtime is spinning up.

## Copyright and License

`gons` is Copyright 2019 Harald Albrecht, and licensed under the Apache
License, Version 2.0.
