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
	"strings"
	gotesting "testing"

	"github.com/thediveo/gons/reexec/internal/testsupport"
)

// M is an "enhanced" version of Golang's testing.M which additionally handles
// merging coverage profile data from re-executions into the main ("parent's")
// coverage profile file.
type M struct {
	*gotesting.M
}

// Testing-related CLI arguments picked up from os.Args, which are of
// relevance to coverage profile data handling.
var (
	outputDir    string // "-test.outputdir"
	coverProfile string // "-test.coverprofile"
)

// Run runs the tests and for the parent process then correctly merges the
// coverage profile data from re-executed process copies into this parent
// process' coverage profile data. Run returns an exit code to pass to
// os.Exit.
func (m *M) Run() (exitcode int) {
	// If necessary, run the action first, as this gathers the coverage
	// profile data during re-execution, which we are interested in. Please
	// note that we cannot use gons.reexec.RunAction() directly, as this would
	// result in an import cycle. To break this vicious cycle we use
	// testsupport's RunAction instead, which gons.reexec will initialize to
	// point to its real implementation of RunAction.
	reexeced := testsupport.RunAction()
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
		exitcode = m.M.Run()
		// For the parent we finally need to gather the coverage profile data
		// written by the individual re-executed child processes, and merge it
		// with our own coverage profile data. Our data has been written at the
		// end of the (empty) m.M.Run(), so we can only now do the final merge.
		if coverProfile != "" && exitcode == 0 {
			mergeAndReportCoverages()
		}
	} else {
		// Run the empty test set when we're an re-executed child, so that the
		// Go testing package creates a coverage profile data report.
		pritiPratel(func() {
			exitcode = m.M.Run()
		})
	}
	return
}

// mergeAndReportCoverages picks up the coverage profile data files created by
// re-executed copies and merges them into this (parent) process' coverage
// profile data.
func mergeAndReportCoverages() {
	sumcp := coverageProfile{
		Sources: make(map[string]*coverageProfileSource),
	}
	// Prime summary coverage profile data from this parent's coverage profile
	// data...
	mergedname := toOutputDir(coverProfile)
	mergeCoverageFile(mergedname, &sumcp)
	// ...and then merge in the re-executed children's coverage profile data.
	for _, coverprofilename := range testsupport.CoverageProfiles {
		fname := toOutputDir(coverprofilename)
		mergeCoverageFile(fname, &sumcp)
	}
	// Finally dump the summary coverage profile data onto the parent's
	// coverage profile data, overwriting it.
	f, err := os.Create(mergedname)
	if err != nil {
		panic("cannot report summary coverage profile data: " + err.Error())
	}
	defer f.Close()
	fmt.Fprintf(f, "mode: %s\n", sumcp.Mode)
	for sourcename, source := range sumcp.Sources {
		for _, block := range source.Blocks {
			fmt.Fprintf(f, "%s:%d.%d,%d.%d %d %d\n",
				sourcename,
				block.StartLine, block.StartCol,
				block.EndLine, block.EndCol,
				block.NumStmts,
				block.Counts)
		}
	}
}

// parseCoverageArgs gathers the output directory and cover profile file from
// the CLI arguments.
func parseCoverageArgs(args []string) {
	for idx := 0; idx < len(args); idx++ {
		arg := args[idx]
		if strings.HasPrefix(arg, "-test.outputdir=") {
			outputDir = strings.SplitN(arg, "=", 2)[1]
		} else if strings.HasPrefix(arg, "-test.coverprofile=") {
			coverProfile = strings.SplitN(arg, "=", 2)[1]
		} else if arg == "-args" || arg == "--args" {
			break
		}
	}
}

// toOutputDir is a Linux-only variant of testing's toOutputDir: it returns
// the specified filename relocated, if required, to outputDir.
func toOutputDir(path string) string {
	if outputDir == "" || path == "" {
		return path
	}
	// If the name of the coverage profile data file is already an absolute
	// path, then simply return it.
	if os.IsPathSeparator(path[0]) {
		return path
	}
	// Otherwise return the coverage profile data filename relative to the
	// specified output directory path ... the latter might be relative or
	// absolute, but we don't care here.
	return fmt.Sprintf("%s%c%s", outputDir, os.PathSeparator, path)
}
