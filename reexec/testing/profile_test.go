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

package testing

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("coverage profile data", func() {

	It("rejects invalid coverage profile data files", func() {
		cp := newCoverageProfile()
		Expect(func() { mergeCoverageFile("test/nonexisting.cov", cp) }).NotTo(Panic())
		Expect(cp.Sources).To(BeEmpty())

		Expect(func() { mergeCoverageFile("/root", cp) }).To(Panic())
		Expect(cp.Sources).To(BeEmpty())

		Expect(func() { mergeCoverageFile("test/empty.cov", cp) }).NotTo(Panic())
		Expect(cp.Sources).To(BeEmpty())

		Expect(func() { mergeCoverageFile("test/modeless.cov", cp) }).To(Panic())
		Expect(cp.Sources).To(BeEmpty())

		Expect(func() { mergeCoverageFile("test/broken1.cov", cp) }).To(Panic())
		Expect(cp.Sources).To(BeEmpty())

		Expect(func() { mergeCoverageFile("test/broken2.cov", cp) }).To(Panic())
		Expect(cp.Sources).To(BeEmpty())

		Expect(func() { mergeCoverageFile("test/broken3.cov", cp) }).To(Panic())
		Expect(cp.Sources).To(BeEmpty())
	})

	It("reads coverage profile data", func() {
		cp := newCoverageProfile()
		Expect(func() { mergeCoverageFile("test/cov1.cov", cp) }).NotTo(Panic())
		Expect(cp.Mode).To(Equal("atomic"))
		Expect(cp.Sources).To(HaveLen(2))
		Expect(cp.Sources).To(HaveKey("a/b.go"))
		Expect(cp.Sources).To(HaveKey("a/c.go"))
		Expect(cp.Sources["a/b.go"].Blocks).To(HaveLen(2))
		Expect(cp.Sources["a/b.go"].Blocks[0]).To(Equal(coverageProfileBlock{
			StartLine: 1,
			StartCol:  0,
			EndLine:   2,
			EndCol:    42,
			NumStmts:  3,
			Counts:    456,
		}))
	})

	It("rejects merging different modes", func() {
		cp := newCoverageProfile()
		Expect(func() { mergeCoverageFile("test/cov1.cov", cp) }).NotTo(Panic())
		Expect(func() { mergeCoverageFile("test/set.cov", cp) }).To(Panic())
	})

	It("merges mode \"atomic\"", func() {
		cp := newCoverageProfile()
		Expect(func() { mergeCoverageFile("test/cov1.cov", cp) }).NotTo(Panic())
		Expect(func() { mergeCoverageFile("test/cov2.cov", cp) }).NotTo(Panic())
		Expect(cp.Sources).To(HaveLen(3))
		Expect(cp.Sources["a/b.go"].Blocks).To(HaveLen(2))
		Expect(cp.Sources["a/c.go"].Blocks).To(HaveLen(1))
		Expect(cp.Sources["a/d.go"].Blocks).To(HaveLen(2))
		Expect(cp.Sources["a/b.go"].Blocks[0]).To(Equal(coverageProfileBlock{
			StartLine: 1,
			StartCol:  0,
			EndLine:   2,
			EndCol:    42,
			NumStmts:  3,
			Counts:    4560,
		}))
	})

	It("merges mode \"set\"", func() {
		cp := newCoverageProfile()
		Expect(func() { mergeCoverageFile("test/set.cov", cp) }).NotTo(Panic())
		Expect(func() { mergeCoverageFile("test/set.cov", cp) }).NotTo(Panic())
		Expect(cp.Sources).To(HaveLen(2))
		Expect(cp.Sources["a/b.go"].Blocks[0]).To(Equal(coverageProfileBlock{
			StartLine: 1,
			StartCol:  0,
			EndLine:   2,
			EndCol:    42,
			NumStmts:  3,
			Counts:    1,
		}))
	})

})
