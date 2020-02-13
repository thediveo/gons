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

package reexec

import (
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func init() {
	Register("action", func() {
		fmt.Fprint(os.Stdout, `"done"`)
	})
	Register("sleepy", func() {
		fmt.Fprint(os.Stdout, `"sleeping"`)
		// Just keep this re-executed child action sleeping; we will be killed
		// by our parent when the test is done. What a lovely family.
		select {}
	})
	Register("unintelligible", func() {
		// Return something the parent process didn't expect.
		println(42)
	})
}

var _ = Describe("reexec", func() {

	It("runs action and decodes answer", func() {
		var s string
		Expect(ForkReexec("action", []Namespace{}, &s)).NotTo(HaveOccurred())
		Expect(s).To(Equal("done"))
	})

	It("doesn't accept registering the same action name twice", func() {
		Expect(func() { Register("foo", func() {}) }).NotTo(Panic())
		Expect(func() { Register("foo", func() {}) }).To(Panic())
	})

	It("doesn't run the child for a non-preregistered action", func() {
		// Note how registering the bar action here will cause the re-executed
		// package test child to fail, because this will trigger CheckAction()
		// without the bar action being registered early enough in the child.
		Expect(func() { Register("bar", func() {}) }).NotTo(Panic())
		Expect(ForkReexec("bar", []Namespace{}, nil)).To(
			MatchError(MatchRegexp(`ForkReexec: child failed:`)))
	})

	FIt("terminates a hanging re-executed child", func() {
		var s string
		done := make(chan error)
		go func() {
			Expect(ForkReexec("sleepy", []Namespace{}, &s)).ToNot(HaveOccurred())
			Expect(s).To(Equal("sleeping"))
			done <- nil
		}()
		select {
		case <-time.After(5 * time.Second):
			Fail("ForkReexec failed to terminate sleeping re-executed child in time")
		case <-done:
		}
	})

})
