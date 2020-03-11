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
	"sort"
	"strings"
)

// Testing (coverage) related CLI arguments picked up from os.Args, which are
// of relevance to coverage profile data handling.
var (
	outputDir    string // "-test.outputdir"
	coverProfile string // "-test.coverprofile"
)

// mergeAndReportCoverages picks up the coverage profile data files created by
// re-executed copies and merges them into this (parent) process' coverage
// profile data.
func mergeAndReportCoverages(maincovprof string, childcovprofs []string) {
	sumcp := coverageProfile{
		Sources: make(map[string]*coverageProfileSource),
	}
	// Prime summary coverage profile data from this parent's coverage profile
	// data...
	mergeCoverageFile(toOutputDir(maincovprof), &sumcp)
	// ...then merge in the re-executed children's coverage profile data, and
	// write the results into a file.
	mergeWithCoverProfileAndReport(&sumcp, childcovprofs, maincovprof)
}

// mergeWithCoverProfileAndReport takes a coverage profile, merges in other
// coverage profile data files, and then writes the summary coverage profile
// data to the specified file.
func mergeWithCoverProfileAndReport(sumcp *coverageProfile, childcovprofs []string, mergedname string) {
	// Merge in other coverage profile data files (typically created by
	// re-executed child processes).
	for _, coverprofilename := range childcovprofs {
		mergeCoverageFile(toOutputDir(coverprofilename), sumcp)
	}
	// Finally dump the summary coverage profile data onto the parent's
	// coverage profile data, overwriting it.
	f, err := os.Create(toOutputDir(mergedname))
	if err != nil {
		panic("cannot report summary coverage profile data: " + err.Error())
	}
	defer f.Close()
	fmt.Fprintf(f, "mode: %s\n", sumcp.Mode)
	// To make testing deterministic, we need to deterministically sort the
	// source filename keys, as otherwise the map may iterate in arbitrary
	// order over the sources.
	sourcenames := make([]string, len(sumcp.Sources))
	idx := 0
	for sourcename := range sumcp.Sources {
		sourcenames[idx] = sourcename
		idx++
	}
	sort.Strings(sourcenames)
	for _, sourcename := range sourcenames {
		for _, block := range sumcp.Sources[sourcename].Blocks {
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
