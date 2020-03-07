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
	"fmt"
	"io"
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/thediveo/gons/reexec"
)

func init() {
	reexec.Register("foo", func() {
		fmt.Println(`"foo done"`)
	})
}

var _ = Describe("coveraging re-execution", func() {

	It("re-executes action foo self-test", func() {
		var result string
		Expect(reexec.ForkReexec("foo", nil, &result)).To(Succeed())
	})

	It("outputs to directory", func() {
		oldod := outputDir
		defer func() { outputDir = oldod }()
		Expect(toOutputDir("foo")).To(Equal("foo"))
		Expect(toOutputDir("/foo")).To(Equal("/foo"))
		outputDir = "bar"
		Expect(toOutputDir("foo")).To(Equal("bar/foo"))
		Expect(toOutputDir("/foo")).To(Equal("/foo"))
	})

	It("parses coverage-related CLI args", func() {
		oldod, oldcp := outputDir, coverProfile
		defer func() { outputDir, coverProfile = oldod, oldcp }()
		arghs := []string{
			"abc",
			"-test.outputdir=bar",
			"-test.coverprofile=foo",
			"-args",
			"-test.outputdir=xxx",
		}
		parseCoverageArgs(arghs)
		Expect(outputDir).To(Equal("bar"))
		Expect(coverProfile).To(Equal("foo"))
	})

	It("merges coverage reports and writes merged report", func() {
		tmpdir, err := ioutil.TempDir("", "covreport")
		Expect(err).NotTo(HaveOccurred())
		defer os.RemoveAll(tmpdir)

		ffrom, err := os.Open("test/cov1.cov")
		Expect(err).NotTo(HaveOccurred())
		defer ffrom.Close()
		fto, err := os.Create(tmpdir + "/main.cov")
		Expect(err).NotTo(HaveOccurred())
		defer fto.Close()
		_, err = io.Copy(fto, ffrom)
		Expect(err).NotTo(HaveOccurred())
		ffrom.Close() // yep, close it now; double closes are alright.
		fto.Close()   // yep, close it now; double closes are alright.

		mergeAndReportCoverages(
			tmpdir+"/main.cov",
			[]string{"test/cov2.cov"})
		actualfinalreport, err := ioutil.ReadFile(tmpdir + "/main.cov")
		Expect(err).NotTo(HaveOccurred())
		finalreport, err := ioutil.ReadFile("test/final.cov")
		Expect(err).NotTo(HaveOccurred())
		Expect(string(actualfinalreport)).To(Equal(string(finalreport)))
	})

})
