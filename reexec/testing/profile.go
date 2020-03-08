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
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
)

// coverageProfile represents coverage profile data for a specific coverage
// profile data file.
type coverageProfile struct {
	// Mode of coverage profile: "atomic", "count", or "set".
	Mode string
	// Sources with block coverage data, indexed by source file name.
	Sources map[string]*coverageProfileSource
}

// newCoverageProfile returns a new and correctly initialized coverageProfile.
func newCoverageProfile() *coverageProfile {
	return &coverageProfile{
		Sources: map[string]*coverageProfileSource{},
	}
}

// coverageProfileSource represents the coverage blocks of a single source
// file.
type coverageProfileSource struct {
	Blocks []coverageProfileBlock // coverage blocks per source file.
}

// coverageProfileBlockByStart is a type alias for sorting slices of
// coverageProfileBlocks.
type coverageProfileBlockByStart []coverageProfileBlock

func (b coverageProfileBlockByStart) Len() int      { return len(b) }
func (b coverageProfileBlockByStart) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b coverageProfileBlockByStart) Less(i, j int) bool {
	bi, bj := b[i], b[j]
	return bi.StartLine < bj.StartLine ||
		(bi.StartLine == bj.StartLine && bi.StartCol < bj.StartCol)
}

// coverageProfileBlock represents a single block of coverage profiling data.
type coverageProfileBlock struct {
	StartLine uint32 // line number for block start.
	StartCol  uint16 // column number for block start.
	EndLine   uint32 // line number for block end.
	EndCol    uint16 // column number for block end.
	NumStmts  uint16 // number of statements included in this block.
	Counts    uint32 // number of times this block was executed.
}

// modeRe specifies the format of the first "mode:" text line of a coverage
// profile data file.
var modeRe = regexp.MustCompile(`^mode: ([[:alpha:]]+)$`)

// lineRe specifies the format of the block text lines in coverage profile
// data files.
var lineRe = regexp.MustCompile(`^(.+):([0-9]+).([0-9]+),([0-9]+).([0-9]+) ([0-9]+) ([0-9]+)$`)

// mergeCoverageFile reads coverage profile data from the file specified in
// the path parameter and merges it with the summary coverage profile in
// sumcp.
func mergeCoverageFile(path string, sumcp *coverageProfile) {
	cpf, err := os.Open(toOutputDir(path))
	if err != nil {
		if os.IsNotExist(err) {
			// Silently skip the situation when a re-execution did not create
			// a coverage profile data file.
			return
		}
		panic(fmt.Sprintf(
			"unable to merge coverage profile data file %q: %s",
			toOutputDir(path), err.Error()))
	}
	defer cpf.Close()
	scan := bufio.NewScanner(cpf)
	if !scan.Scan() {
		return
	}
	// Phase I: read in the specified coverage profile data file, before we
	// can attempt to merge it.
	cp := newCoverageProfile()
	// The first line of a coverage profile data file is the mode how
	// coverage data was gathered; either "atomic", "count", or "set".
	line := scan.Text()
	m := modeRe.FindStringSubmatch(line)
	if m == nil {
		panic(fmt.Sprintf(
			"line %q doesn't match expected mode: line format", line))
	}
	cp.Mode = m[1]
	// The remaining lines contain coverage profile block data. We optimize
	// here on the basis that Go's testing/coverage.go writes coverage profile
	// data files where the coverage block data for the same source file is
	// continuous (instead of being scattered around). However, the code
	// blocks are not sorted.
	var srcname string                // caches most recent source filename.
	var source *coverageProfileSource // caches most recent source data.
	for scan.Scan() {
		line = scan.Text()
		m := lineRe.FindStringSubmatch(line)
		if m == nil {
			panic(fmt.Sprintf(
				"line %q doesn't match expected block line format", line))
		}
		if m[1] != srcname {
			// If we haven't seen this source filename yet, allocate a
			// coverage data source element and put into the map of known
			// sources.
			srcname = m[1]
			source = &coverageProfileSource{}
			cp.Sources[srcname] = source
		}
		// Append the block data from the coverage profile data file line, the
		// sequence of blocks is yet unsorted.
		source.Blocks = append(source.Blocks, coverageProfileBlock{
			StartLine: toUint32(m[2]),
			StartCol:  toUint16(m[3]),
			EndLine:   toUint32(m[4]),
			EndCol:    toUint16(m[5]),
			NumStmts:  toUint16(m[6]),
			Counts:    toUint32(m[7]),
		})
	}
	// Phase II: check for the proper coverage profile mode first...
	if sumcp.Mode == "" {
		sumcp.Mode = cp.Mode
	} else if cp.Mode != sumcp.Mode {
		panic(fmt.Sprintf("expected mode %q, got mode %q", sumcp.Mode, cp.Mode))
	}
	// Phase III: for each source, sort the source's coverage blocks and then
	// merge the coverage block data into the summary coverage profile.
	setmode := sumcp.Mode == "set"
	for srcname, source := range cp.Sources {
		sort.Sort(coverageProfileBlockByStart(source.Blocks))
		// Look up the corresponding source in the summary coverage profile,
		// or create a new one, if not already present.
		var sumsource *coverageProfileSource
		var ok bool
		if sumsource, ok = sumcp.Sources[srcname]; !ok {
			sumsource = &coverageProfileSource{}
			sumcp.Sources[srcname] = sumsource
		}
		// Now merge...
		sumblkidx := 0
	NextBlock:
		for _, block := range source.Blocks {
			for sumblkidx < len(sumsource.Blocks) {
				sumblock := &sumsource.Blocks[sumblkidx]
				sumblkidx++ // yes, increment anyway, as no block appears twice.
				if sumblock.StartLine == block.StartLine &&
					sumblock.StartCol == block.StartCol &&
					sumblock.EndLine == block.EndLine &&
					sumblock.EndCol == block.EndCol {
					// We've found a matching code block, so update its
					// coverage data.
					if setmode {
						sumblock.Counts |= block.Counts
					} else {
						sumblock.Counts += block.Counts
					}
					continue NextBlock
				}
			}
			// No matching block found, append it. Since the coverage profile
			// data files come from parent and children runs on the same
			// snapshot of sources, they contain the same blocks. And since
			// we've sorted the blocks, we can simply append and it should
			// still keep sorting order.
			sumsource.Blocks = append(sumsource.Blocks, block)
		}
	}
}

// toUint32 converts a textual int value into its binary uint32
// representation. If the specified text doesn't represent a valid uint32
// value, toUint32 panics.
func toUint32(s string) uint32 {
	if v, err := strconv.ParseUint(s, 10, 32); err != nil {
		panic(err.Error())
	} else {
		return uint32(v)
	}
}

// toUint16 converts a textual int value into its binary uint16
// representation. If the specified text doesn't represent a valid uint16
// value, toUint16 panics.
func toUint16(s string) uint16 {
	if v, err := strconv.ParseUint(s, 10, 16); err != nil {
		panic(err.Error())
	} else {
		return uint16(v)
	}
}
