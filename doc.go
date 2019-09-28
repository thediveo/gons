// Package gons ("go [into] namespaces") selectively switches your Go
// application into other already existing Linux namespaces. This must happen
// before the Go runtime spins up, blocking certain namespace changes, such as
// changing into a different mount namespace.
//
// Code Usage
//
// Simply import the gons package into your application, and you're set.
//
//   package main
//
//   import _ "github.com/thediveo/gons"
//
//   func main() {
//       // ...
//   }
//
// Runtime Usage
//
// The existing namespaces to join/switch into are referenced by their paths
// in the filesystem (such as `/proc/123456/ns/mnt`), and are specified using
// environment variables. Set only the environment variables for those
// namespaces that should be switched at startup. These variables need to be
// set before your application is started. The names of the environment
// variables are as follows and must be all lowercase:
//
//   cgroupns=...
//   ipcns=...
//   mntns=...
//   netns=...
//   pidns=... # see note below
//   userns=...
//   utsns=...
//
// Notes
//
// Setting pidns= does not switch your application's own PID namespace, but
// rather controls the PID namespace any child processes of your application
// will be put into.
//
// If a given namespace path is invalid, or if there are insufficient rights
// to access the path or switch to the specified namespace, then an error
// message is printed to stderr and the application aborted with error code 1.
//
// The `gons` package requires cgo (https://golang.org/cmd/cgo/): the required
// namespace switches can only safely be done while your application is still
// single-threaded and that's only the case before the Go runtime is spinning
// up.
//
package gons
