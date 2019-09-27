package main

import (
	"fmt"
	"os"

	"github.com/moby/moby/pkg/reexec"

	_ "github.com/thediveo/gons"
)

// Thanks to
// https://jiajunhuang.com/articles/2018_08_28-how_does_golang_implement_fork_syscall.md.html
// for the example of how to use Moby's reexec to restart ourself in order to
// be able to switch into prickly namespaces, such as the mount namespace: it
// cannot be changed after the Go runtime has spun up and created additional
// OS threads.
func init() {
	reexec.Register("switch-namespaces", SwitchedNamespaces)
}

func SwitchedNamespaces() {
	fmt.Printf("Restarted with switched namespaces.\n")

}

func main() {
	if reexec.Init() {
		fmt.Printf("Restarted child started.\n")
		for _, ns := range []string{"cgroup", "ipc", "mnt", "net", "pid", "user", "uts"} {
			if nsref, err := os.Readlink(fmt.Sprintf("/proc/self/ns/%s", ns)); err == nil {
				fmt.Printf("%s\n", nsref)
			}
		}
		fmt.Printf("Restarted child done.\n")
	} else {
		fmt.Printf("Started.\n")

		if len(os.Args) >= 2 {
			os.Setenv("netns", fmt.Sprintf("/proc/%s/ns/net", os.Args[1]))
		}

		cmd := reexec.Command("switch-namespaces")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			panic(err)
		} else if err := cmd.Wait(); err != nil {
			panic(err)
		}
		fmt.Printf("Stopped.\n")
	}
}
