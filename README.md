# gons

`gons` ("go [*into*] namespaces") is a Go package that selectively switches
your Go application to other already existing Linux namespaces. This must
happen before the Go runtime spins up, blocking certain namespace changes,
such as changing into a different mount namespace.

- `gons` switches the Go application *itself* it is linked to into a set of
  already existing namespaces.
- `gons` does *neither fork nor re-execute* in order to switch namespaces.
- `gons` *does not* create new namespaces.

In consequence, `gons` is a package tackling a different usecase than
`runc/libcontainer`'s famous
[*nsenter*](https://github.com/opencontainers/runc/tree/master/libcontainer/nsenter)
package.

The existing namespaces to join/switch into are referenced by their paths in
the filesystem (such as `/proc/123456/ns/mnt`), and are specified using
environment variables. Set only the environment variables for those namespaces
that should be switched at startup. The names of the environment variables are
as follows:

- `cgroupns=...`
- `ipcns=...`
- `mntns=...`
- `netns=...`
- `pidns=...` (*please note that this does not switch your applications own
  PID namespace, but rather controls the PID namespace any child processes of
  your application will be put into.*)
- `userns=...`
- `utsns=...`

> **Note:** if a given namespace path is invalid, or if there are insufficient
> rights to access the path or switch to the specified namespace, then an
> error message is printed to stderr and the application aborted with error
> code 1.

The `gons` package requires [cgo](https://golang.org/cmd/cgo/): the required
namespace switches can only safely be done while your application is still
single-threaded and that's only the case before the Go runtime is spinning up.

## Copyright and License

`gons` is Copyright 2018 Harald Albrecht, and licensed under the Apache
License, Version 2.0.
