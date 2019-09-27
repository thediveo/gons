// Copyright 2019 Harald Albrecht.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gons

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/moby/moby/pkg/reexec"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("gons", func() {

	// Basic self-check that reexecution is working and doesn't trigger an
	// infinite loop.
	It("reexecutes itself", func() {
		cmd := reexec.Command("foo", "-ginkgo.focus=NOTESTS")
		out, err := cmd.Output()
		Expect(err).NotTo(HaveOccurred())
		Expect(string(out)).To(ContainSubstring("Running Suite: gons suite"))
	})

	// Reexecute with an invalid namespace reference.
	It("aborts reexecution for invalid namespace reference", func() {
		cmd := reexec.Command("foo", "-ginkgo.focus=NOTESTS")
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, "netns=/foo")
		_, err := cmd.Output()
		Expect(err).To(HaveOccurred())
		ee, ok := err.(*exec.ExitError)
		Expect(ok).To(BeTrue())
		Expect(string(ee.Stderr)).To(ContainSubstring(
			"gonamespaces: invalid netns reference \"/foo\": "))
	})

	It("switches namespaces when reexecuting", func() {
		cmd := reexec.Command("reexecutee", "-ginkgo.focus=NOTESTS")
		out, err := cmd.Output()
		Expect(err).To(HaveOccurred())
		ee, ok := err.(*exec.ExitError)
		Expect(ok).To(BeTrue())
		Expect(ee.ExitCode()).To(Equal(42))
		Expect(string(out)).To(ContainSubstring("net:["))
	})

})

// Make sure that we have a reexecution handler installed for some of our
// tests: it will be run inside the reexecuted child, and its output is then
// checked in the parent running the test cases.
func init() {
	reexec.Register("reexecutee", func() {
		// Dump all namespace identifiers to allow checks on what really
		// happened...
		for _, ns := range []string{"cgroup", "ipc", "mnt", "net", "pid", "user", "uts"} {
			if nsref, err := os.Readlink(fmt.Sprintf("/proc/self/ns/%s", ns)); err == nil {
				fmt.Printf("%s\n", nsref)
			}
		}
		os.Exit(42)
	})
	// Ensure that the registered handler is run in the reexecuted child. This
	// won't trigger the handler while we're in the parent, because the
	// parent's Arg[0] won't match the name of our handler.
	reexec.Init()
}
