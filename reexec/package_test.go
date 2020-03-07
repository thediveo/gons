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
	"io"
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/thediveo/gons"
	rxtst "github.com/thediveo/gons/reexec/testing"
)

// As we need to do some pre-test checks in order to run actions on
// re-execution, wo go for TestMain instead of an ordinary TextXxx function
// when unit-testing this package.
func TestMain(m *testing.M) {
	// If there was a failure in switching namespaces during initial startup,
	// then report this and end the process with a non-zero status. We do this
	// regardless of whether we're the original test or a re-executed child.
	if err := gons.Status(); err != nil {
		_, _ = io.WriteString(os.Stderr, err.Error())
		_, _ = io.WriteString(os.Stderr, "\n")
		os.Exit(1)
	}
	// We eat our own dog (testing) food here...
	mm := &rxtst.M{M: m}
	os.Exit(mm.Run())
}

func TestPackage(t *testing.T) {
	// Okay, we're a real test suite, and there was no re-executed child
	// handler triggering... :)
	RegisterFailHandler(Fail)
	RunSpecs(t, "reexec package")
}
