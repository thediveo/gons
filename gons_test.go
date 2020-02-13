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
	"strings"
	"syscall"

	"github.com/moby/moby/pkg/reexec"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("gons", func() {

	// Basic self-check that reexecution is working and doesn't trigger an
	// infinite loop.
	It("re-executes itself", func() {
		cmd := reexec.Command("foo", "-ginkgo.focus=NOTESTS")
		out, err := cmd.Output()
		Expect(err).NotTo(HaveOccurred())
		Expect(string(out)).To(ContainSubstring("Running Suite: gons suite"))
	})

	// Re-execute with an invalid namespace reference.
	It("aborts re-execution for invalid namespace reference", func() {
		cmd := reexec.Command("foo", "-ginkgo.focus=NOTESTS")
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, "gons_net=/foo")
		_, err := cmd.Output()
		Expect(err).To(HaveOccurred())
		ee, ok := err.(*exec.ExitError)
		Expect(ok).To(BeTrue())
		Expect(string(ee.Stderr)).To(ContainSubstring(
			"package gons: invalid gons_net reference \"/foo\": "))
	})

	// Re-execute and switch into other namespaces especially created for this
	// test.
	It("switches namespaces when re-executing", func() {
		if os.Geteuid() != 0 {
			Skip("needs root")
		}
		// Re-execute ourselves and tell (re)exec to create some new namespaces
		// for our clone. The purpose of our clone is to just sleep in order
		// to keep those pesky little namespaces open for as long as we need
		// them for testing. By creating a new user namespace, we are allowed
		// to do this without being root, and in consequence, we're also
		// allowed to create new mount and network namespaces also without
		// needing root. It's okay that our re-executed child will be nobody,
		// so we skip setting up UID and GID mappings.
		//
		// As for useful references about using Linux namespaces in Go, please
		// refer to:
		// https://medium.com/@teddyking/namespaces-in-go-basics-e3f0fc1ff69a
		// and
		// https://medium.com/@teddyking/namespaces-in-go-user-a54ef9476f2a
		sleepy := reexec.Command("sleepingunbeauty", "-ginkgo.focus=NOTESTS")
		sleepy.SysProcAttr.Cloneflags =
			syscall.CLONE_NEWUSER | syscall.CLONE_NEWNS | syscall.CLONE_NEWNET
		Expect(sleepy.Start()).To(Succeed())
		// Ensure to terminate the sleeping re-execed child when this test
		// finishes. The sleeping child should not have exited by itself by
		// the end of the test, so we consider any issues in killing it a
		// failure -- that test description rather sounds like a really bad
		// movie script (SCHLEFAZ, anyone???). Anyway, waiting for the child
		// to terminate should then be without problems, and here we consider
		// the child being killed as "no problem". Oh, I just notice there are
		// a lot of likes by King Herodes and his henchmen.
		defer func() {
			Expect(sleepy.Process.Kill()).To(Succeed())
			err := sleepy.Wait()
			if exiterr, ok := err.(*exec.ExitError); ok {
				if !exiterr.Sys().(syscall.WaitStatus).Signaled() {
					Expect(err).To(Succeed())
				}
			} else {
				Expect(err).To(Succeed())
			}
		}()
		// Now we re-execute another child, but this time we tell it to join
		// the newly created namespaces and then check to see if it succeeded.
		// We tell the child which namespaces to join through a set of
		// environment variables passed to it upon start.
		joiner := reexec.Command("sleepingunbeauty", "-ginkgo.focus=NOTESTS")
		joiner.Env = os.Environ()
		var out strings.Builder
		joiner.Stdout = &out
		joiner.Stderr = &out
		namespaces := []string{"user", "mnt", "net"}
		for _, ns := range namespaces {
			joiner.Env = append(joiner.Env,
				fmt.Sprintf("gons_%s=/proc/%d/ns/%s",
					ns, sleepy.Process.Pid, ns))
		}
		Expect(joiner.Start()).To(Succeed())
		// Ensure to terminate the re-executed child under test after we've
		// passed the main checks.
		defer func() {
			Expect(joiner.Process.Kill()).To(Succeed())
		}()
		// Wait for reexeced child to terminate so that we can check for early
		// child fails while running our test here.
		go func() { _ = joiner.Wait() }()
		// Now check the reexeceuted child to use the changed namespaces as we
		// told it to do...
		for _, ns := range namespaces {
			newnsid, err := os.Readlink(fmt.Sprintf(
				"/proc/%d/ns/%s", sleepy.Process.Pid, ns))
			Expect(err).To(Succeed())
			// Since we might well to early yet the reexecuted child might not
			// have yet switched its namespaces ... so we need to give it a
			// little bit of time to settle things. Also, the child might have
			// terminated already due to fatal failures, so we want to catch
			// this situation here too.
			Eventually(func() string {
				Expect(joiner.ProcessState).To(BeNil(),
					"reexecuted joiner child terminated prematurely: "+out.String())
				joinersnsid, err := os.Readlink(fmt.Sprintf(
					"/proc/%d/ns/%s", joiner.Process.Pid, ns))
				Expect(err).To(Succeed())
				return joinersnsid
			}, "2s", "20ms").Should(Equal(newnsid))
		}
	})

})
