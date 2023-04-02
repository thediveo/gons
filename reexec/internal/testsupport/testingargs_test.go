// Copyright 2020 Harald Albrecht.
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

package testsupport

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("testing-related args", func() {

	It("correctly generates child's testing-related args", func() {
		defer func(et bool) { TestingEnabled = et }(TestingEnabled)
		defer func() { CoverageProfiles = []string{} }()
		EnableTesting("/foo", "bar")
		tstargs := TestingArgs()
		Expect(tstargs).To(ContainElement("-test.coverprofile=bar_0"))
		Expect(tstargs).To(ContainElement("-test.outputdir=/foo"))
		Expect(tstargs).To(ContainElement(MatchRegexp("-test.run=.+")))
	})

})
