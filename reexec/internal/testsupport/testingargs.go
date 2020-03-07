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
	"fmt"
)

// TestingEnabled is set to true when we're under testing; gathering coverage
// profile data might be enabled.
var TestingEnabled = false

// CoverageOutputDir is the directory in which to create profile files and the
// like. When run from "go test", our binary always runs in the source
// directory for the package under test. The CLI argument "-test.outputdir"
// corresponding with this variable lets "go test" tell our binary to write
// the files in the directory where the "go test" command is run.
var CoverageOutputDir = ""

// CoverageProfile is the name of a coverage profile data file; if empty, then
// no coverage profile is to be saved. This variable corresponds with the
// "-test.coverprofile" CLI argument.
var CoverageProfile = ""

// EnableTesting is a module-internal function used by the gons/reexec/testing
// (sub) package; it tells this reexec package when we're in testing mode, and
// also passes coverage profiling-related test parameters to us. We need these
// parameters when re-executing child processes and in order to allocate
// coverage profile data files to these children.
func EnableTesting(outputdir, coverprofile string) {
	TestingEnabled = true
	CoverageOutputDir = outputdir
	CoverageProfile = coverprofile
}

// CoverageProfiles is a list of coverage profile data filenames created by
// re-executed child processes when under test.
var CoverageProfiles = []string{}

// TestingArgs returns additional testing arguments while under test;
// otherwise it returns an empty slice of arguments.
func TestingArgs() []string {
	testargs := []string{}
	if TestingEnabled {
		if CoverageProfile != "" {
			name := CoverageProfile +
				fmt.Sprintf("_%d", len(CoverageProfiles))
			CoverageProfiles = append(CoverageProfiles, name)
			if CoverageProfile != "" {
				testargs = append(testargs,
					"-test.coverprofile="+name)
			}
			if CoverageOutputDir != "" {
				testargs = append(testargs,
					"-test.outputdir="+CoverageOutputDir)
			}
		}
		// Let's suppose for a moment that no sane developer will ever use the
		// following name for one of her/his tests ... except for "THEM" :p
		testargs = append(testargs,
			"-test.run=nadazilchnixdairgendwoimnirvanavonbielefeld",
		)
	}
	return testargs
}
