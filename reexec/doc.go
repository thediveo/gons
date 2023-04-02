/*
Package reexec allows to fork and then re-execute the current application
(process) in order to only invoke a specific action function.

Why, because the Golang runtime sucks at fork() and switching Linux kernel
mount namespaces. The go runtime spins up multiple threads, but Linux really
doesn't like changing mount namespaces when a process has become
multi-threaded.

# Example

The following example code registers an action called "action" to be run when
forking and re-executing the example process. As early as possible in main()
we check for a pending action using CheckAction(). It will either execute an
action and then terminate the process, or return control flow in order to run
the process as usual.

The registered example action simply prints its result to os.Stdout and then
returns, which immediately terminates the re-executed process. This result is
returned to the parent process which initiated the re-execution.

	import (
	    "fmt"
	    "os"
	    "github.com/thediveo/gons/reexec"
	}

	func init() {
	    reexec.Register("action", func() {
	      fmt.Fprintln(os.Stdout, `"done"`)
	    })
	}

	func main() {
	    reexec.CheckAction()
	    var result string
	    _ = reexec.RunReexecAction(
	      "action",
	      reexec.Result(&result))
	}

Please note that reexec.RunReexecAction() optionally accepts the namespaces to
run the action in, as well as a parameter and/or environment variables. The
result is picked up in the variable specified using reexec.Result().
*/
package reexec
