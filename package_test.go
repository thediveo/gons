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

package gons_test

import (
	"testing"

	"github.com/thediveo/gons/reexec"
	rxtst "github.com/thediveo/gons/reexec/testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMain(m *testing.M) {
	// There were no namespace switching errors, so we next register this
	// generic re-execution handler that helps our test procedures. It simply
	// puts the re-executed child to sleep, waiting to be killed. This allows
	// the parent test to examine the child's namespaces taking all the time
	// it needs in order to figure out if all went well.
	reexec.Register("sleepingunbeauty", func() {
		// Just keep this re-executed child sleeping; we will be killed by our
		// parent when the test is done. What a lovely family.
		select {}
	})
	// Do NOT USE rxtst.TestMainWithCoverage in your own tests. Use instead:
	//   mm := &rxtst.M{M: m}
	//   os.Exit(mm.Run())
	rxtst.TestMainWithCoverage(m)
}

func TestPackage(t *testing.T) {
	// Okay, we're a real test suite, and there was no re-executed child
	// handler triggering... :)
	RegisterFailHandler(Fail)
	RunSpecs(t, "gons package")
}
