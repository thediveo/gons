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

package testing

import (
	"fmt"
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// While running function f(), captures f's output to stdout.and returns it.
func capturestdout(f func()) (stdout string) {
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	defer func() {
		os.Stdout = origStdout
		r.Close()
		w.Close()
	}()
	os.Stdout = w
	// Run function f() only on this thread, as Gomego doesn't like it
	// otherwise. So we have to run the Stdout replacement pipe reader on a
	// separate go routine. When it has read all that was in the pipe, it will
	// set the return value and signal that it's done.
	done := make(chan struct{})
	go func() {
		b, _ := ioutil.ReadAll(r)
		stdout = string(b)
		close(done)
	}()
	f()
	// Shut down the writer end, so the pipe reader knows that capturing
	// stdout is finished, and can retrieve the complete captured output. We
	// wait for the pipe reader to be finally done before returning.
	w.Close()
	<-done
	return
}

var _ = Describe("stdout processing", func() {

	It("passes test harness self-test", func() {
		Expect(capturestdout(func() { fmt.Print("foo") })).To(Equal("foo"))
	})

	It("correctly passes on normal output", func() {
		Expect(capturestdout(func() {
			noUglyTruthsToStdout(func() {
				fmt.Print("some test")
			})
		})).To(Equal("some test"))
		Expect(capturestdout(func() {
			noUglyTruthsToStdout(func() {
				fmt.Print("coverage is meh\ntest\n")
			})
		})).To(Equal("coverage is meh\ntest\n"))
	})

	It("hides unwanted truths about coverage: and testing:", func() {
		Expect(capturestdout(func() {
			noUglyTruthsToStdout(func() {
				fmt.Print("some test\ncoverage: foo\nbar\ntesting: foo\nbar")
			})
		})).To(Equal("some test\nbar\nbar"))
	})

})
