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
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func init() {
	Register("action", func() {
		fmt.Fprintln(os.Stdout, `"done"`)
	})
	Register("sleepy", func() {
		fmt.Fprintln(os.Stdout, `"sleeping"`)
		// Just keep this re-executed child action sleeping; we will be killed
		// by our parent when the test is done. What a lovely family.
		select {}
	})
	Register("unintelligible", func() {
		// Return something the parent process didn't expect.
		fmt.Fprintln(os.Stdout, `42`)
	})
	Register("envvar", func() {
		fmt.Fprintf(os.Stdout, "%q\n", os.Getenv("foobar"))
	})
	Register("withparam", func() {
		var param string
		if err := json.NewDecoder(os.Stdin).Decode(&param); err != nil {
			fmt.Fprint(os.Stderr, err.Error())
			return
		}
		fmt.Fprintf(os.Stdout, "%q\n", "xx"+param)
	})
	Register("reexec", func() {
		_ = ForkReexec("reexec", []Namespace{}, nil)
	})
	Register("silent", func() {})
}

var _ = Describe("reexec", func() {

	It("runs action and exits", func() {
		ec := -1
		osExit = func(code int) { ec = code }
		defer func() {
			osExit = os.Exit
			os.Setenv(magicEnvVar, "")
		}()
		os.Setenv(magicEnvVar, "silent")
		CheckAction()
		Expect(ec).To(Equal(0))
	})

	It("runs action and decodes answer", func() {
		var s string
		Expect(ForkReexec("action", []Namespace{}, &s)).NotTo(HaveOccurred())
		Expect(s).To(Equal("done"))
	})

	It("runs action and passes env var", func() {
		var s string
		Expect(ForkReexecEnv("envvar", []Namespace{}, []string{"foobar=baz!"}, &s)).NotTo(HaveOccurred())
		Expect(s).To(Equal("baz!"))
	})

	It("runs action with parameter", func() {
		var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
		p := make([]rune, 8192)
		for i := range p {
			p[i] = letters[rand.Intn(len(letters))]
		}
		var r string
		Expect(RunReexecAction("withparam", Param(string(p)), Result(&r))).NotTo(HaveOccurred())
		Expect(r).To(Equal("xx" + string(p)))
	})

	It("panics when re-execution wasn't properly enabled", func() {
		defer func(old bool) { reexecEnabled = old }(reexecEnabled)
		reexecEnabled = false
		Expect(func() { _ = ForkReexec("action", []Namespace{}, nil) }).To(Panic())
	})

	It("doesn't accept registering the same action name twice", func() {
		Expect(func() { Register("foo", func() {}) }).NotTo(Panic())
		Expect(func() { Register("foo", func() {}) }).To(Panic())
	})

	It("doesn't accept triggering a non-registered action", func() {
		Expect(func() { _ = ForkReexec("xxx", []Namespace{}, nil) }).To(Panic())
	})

	It("panics the child for a non-preregistered action", func() {
		// Note how registering the bar action here will cause the re-executed
		// package test child to fail, because this will trigger CheckAction()
		// without the bar action being registered early enough in the child.
		Expect(func() { Register("barx", func() {}) }).NotTo(Panic())
		err := ForkReexec("barx", []Namespace{}, nil)
		Expect(err).To(MatchError(MatchRegexp(
			`.* ReexecAction.Run: child failed with stderr message ` +
				`"unregistered .* action .*\\"barx\\""`)))
	})

	It("panics the child for invalid namespace", func() {
		// Note that it is not possible to re-enter the current user
		// namespace, because that would otherwise give us full privileges. We
		// use this to check that the re-executed child correctly panics when
		// there are problems entering namespaces.
		Expect(ForkReexec("action", []Namespace{
			{Type: "user", Path: "/proc/self/ns/user"},
		}, nil)).To(MatchError(MatchRegexp(`ReexecAction.Run: child failed with stderr message \".* cannot join`)))
	})

	It("doesn't re-execute from a re-executed child", func() {
		Expect(ForkReexec("reexec", []Namespace{}, nil)).To(
			MatchError(MatchRegexp(`ReexecAction.Run: child failed with stderr message \".* tried to re-execute`)))
	})

	It("panics on un-decodable child result", func() {
		var s string
		Expect(ForkReexec("unintelligible", []Namespace{}, &s)).To(
			MatchError(MatchRegexp(`ReexecAction.Run: cannot decode child result`)))
	})

	It("terminates a hanging re-executed child", func() {
		var s string
		done := make(chan error)
		go func() {
			defer GinkgoRecover()
			done <- nil
			select {
			case <-time.After(5 * time.Second):
				Fail("ForkReexec failed to terminate sleeping re-executed child in time")
			case <-done:
			}
		}()
		//
		Expect(ForkReexec("sleepy", []Namespace{}, &s)).ToNot(HaveOccurred())
		Expect(s).To(Equal("sleeping"))
	})

})
