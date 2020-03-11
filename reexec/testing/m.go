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
	"os"
	gotesting "testing"

	"github.com/thediveo/gons/reexec/internal/testsupport"
)

// M is an "enhanced" version of Golang's testing.M which additionally handles
// merging coverage profile data from re-executions into the main ("parent's")
// coverage profile file.
type M struct {
	*gotesting.M
	skipCleanup bool
}

// Run runs the tests and for the parent process then correctly merges the
// coverage profile data from re-executed process copies into this parent
// process' coverage profile data. Run returns an exit code to pass to
// os.Exit.
func (m *M) Run() (exitcode int) {
	exitcode, _ = m.run()
	return
}

// run is the internal implementation of the public Run() method, and
// additionally returns an indication of whether we were running as the parent
// process or a re-executed child process. This indication is used by
// TestMainWithCoverage() to correctly update coverage data to also include
// almost complete coverage of our M.run() code.
func (m *M) run() (exitcode int, reexeced bool) {
	// If necessary, run the action first, as this gathers the coverage
	// profile data during re-execution, which we are interested in. Please
	// note that we cannot use gons.reexec.RunAction() directly, as this would
	// result in an import cycle. To break this vicious cycle we use
	// testsupport's RunAction instead, which gons.reexec will initialize to
	// point to its real implementation of RunAction.
	var recovered interface{}
	func() {
		// RunAction() panics when it is asked to run a non-registered action.
		// But we still want to write coverage profile data, so we need to
		// wrap the call to RunAction(), so that we can recover.
		defer func() {
			if recovered = recover(); recovered != nil {
				// RunAction panics only when trying to re-execute, never
				// otherwise.
				reexeced = true
			}
		}()
		reexeced = testsupport.RunAction()
	}()
	// If we're in coverage mode and we're the parent test process, then pass
	// the required test argument settings to the gons/reexec package, so that
	// it can correctly re-execute child processes under test.
	parseCoverageArgs(os.Args)
	if !reexeced {
		testsupport.EnableTesting(outputDir, coverProfile)
	}
	// Run the tests: for the parent this will be an ordinary test run, but
	// for a re-executed child the passed "-test.run" argument will ensure
	// that actually no tests are run at all, because that would result in
	// tests executed multiple times and panic when hitting a recursive
	// reexec.ForkReexec() call.
	if !reexeced {
		// testing's M.Run() will write the coverage report even when a test
		// panics. And since tests might have used reexec.ForkReexec() we
		// should merge any child coverage profile data results with what
		// M.Run() reported.
		func() {
			defer func() {
				recovered = recover()
			}()
			exitcode = m.M.Run()
		}()
		// For the parent we finally need to gather the coverage profile data
		// written by the individual re-executed child processes, and merge it
		// with our own coverage profile data. Our data has been written at the
		// end of the (empty) m.M.Run(), so we can only now do the final merge.
		if coverProfile != "" && exitcode == 0 {
			mergeAndReportCoverages(coverProfile, testsupport.CoverageProfiles)
			// Now clean up!
			if !m.skipCleanup {
				for _, coverprof := range testsupport.CoverageProfiles {
					_ = os.Remove(toOutputDir(coverprof))
				}
			}
		}
		if recovered != nil {
			// Recover panic!!!
			panic(recovered)
		}
	} else {
		// Run the empty test set when we're an re-executed child, so that the
		// Go testing package creates a coverage profile data report.
		pritiPratel(func() {
			exitcode = m.M.Run()
		})
		// If RunAction() panicked, we "recover our panic", but this way the
		// coverage data has been generated and can later be merged.
		if recovered != nil {
			fmt.Fprint(os.Stderr, recovered)
		}
	}
	return
}
