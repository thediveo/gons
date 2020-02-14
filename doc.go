/*

Package gons ("go [into] namespaces") selectively switches your Go application
into other already existing Linux namespaces. This must happen before the Go
runtime spins up, because the Go runtime later unintentionally blocks certain
namespace changes, especially changing into a different mount namespace when
running with multiple OS threads.

Using gons in Your Code

Simply import the gons package into your application, and you're almost set.
In your application's main() you should check that there were no errors
switching namespaces.

  package main

  import _ "github.com/thediveo/gons"

  func main() {
    if err := Status(); err != nil {
      panic(err.Error())
    }
    // ...
  }

Telling Your Program Which Namespaces to Enter

The existing namespaces to join/switch into are referenced by their paths in
the filesystem (such as "/proc/123456/ns/mnt"), and are specified using
environment variables. Set only the environment variables for those namespaces
that should be switched at startup. These variables need to be set before your
application is started. The names of the environment variables are as follows
and must be all lowercase:

  gons_cgroup=...
  gons_ipc=...
  gons_mnt=...
  gons_net=...
  gons_pid=... # see note below
  gons_user=...
  gons_uts=...

Controlling the Sequence in Which to Enter Namespaces

Additionally, you can specify the order in which the namespaces should be
switched, as well as when the namespace paths are to be opened: if not
overridden by the optional environment variable gons_order=..., then the
default order is "!user,!mnt,!cgroup,!ipc,!net,!pid,!uts" (see below for the
meaning of "!"). It's not necessary to specify all 7 namespace types when you
don't intend to switch them all. For instance, if you just switch the net and
IPC namespaces, then "gons_order=net,ipc" is sufficient.

When a namespace type name is preceded by a bang "!", such as "!user", then
the its path will be opened before the first namespace switch takes place.
Without a bang, the namespace path is opened just right before switching into
this namespace. This is mostly of importance when switching the mount
namespace, as this can also change the filesystem and thus how the namespace
paths are resolved.

Reexec to the Rescue

In case your Go application wants to fork and then restart itself in order to
be able to switch namespaces, you might find the subpackage
"github.com/thediveo/gons/reexec" useful. It simplifies the overall process and
takes care of correctly setting the environment variables.

Technical Notes

Setting "gons_pid=..."" does not switch your application's own PID namespace, but
rather controls the PID namespace any child processes of your application will
be put into.

If a given namespace path is invalid, or if there are insufficient rights to
access the path or switch to the specified namespace, then an error message is
printed to stderr and the application aborted with error code 1.

The gons package requires cgo (https://golang.org/cmd/cgo/): the required
namespace switches can only safely be done while your application is still
single-threaded and that's only the case before the Go runtime is spinning up.

*/
package gons
