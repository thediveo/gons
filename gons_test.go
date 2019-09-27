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

	It("aborts on reexecution for invalid namespace reference", func() {
		cmd := reexec.Command("foo")
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, "netns=/foo")
		_, err := cmd.Output()
		Expect(err).To(HaveOccurred())
		ee, ok := err.(*exec.ExitError)
		Expect(ok).To(BeTrue())
		Expect(string(ee.Stderr)).To(Equal(
			"initns: invalid netns reference \"/foo\"\n"))
	})

})
