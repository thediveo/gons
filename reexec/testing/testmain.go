package testing

import (
	"os"
	"sync/atomic"
	gotesting "testing"
	_ "unsafe" // needed in order to use "go:linkname".

	"github.com/thediveo/gons/reexec/internal/testsupport"
)

// In order to get complete coverage of our M.Run() during our own tests, we
// have to resort to dirty tricks by accessing the package private
// testing.cover variable which contains the complete coverage profile data
// gathered.

//go:linkname cover testing.cover
var cover gotesting.Cover

// coverageProfileFromCover returns the profile data from testing.cover, but
// in our own coverage profile format.
func coverageProfileFromTestingCover() *coverageProfile {
	cp := newCoverageProfile()
	cp.Mode = cover.Mode
	var count uint32
	for sourcename, counts := range cover.Counters {
		source := &coverageProfileSource{
			Blocks: make([]coverageProfileBlock, len(counts)),
		}
		cp.Sources[sourcename] = source
		blocks := cover.Blocks[sourcename]
		for idx := range counts {
			count = atomic.LoadUint32(&counts[idx])
			source.Blocks[idx] = coverageProfileBlock{
				StartLine: blocks[idx].Line0,
				StartCol:  blocks[idx].Col0,
				EndLine:   blocks[idx].Line1,
				EndCol:    blocks[idx].Col1,
				NumStmts:  blocks[idx].Stmts,
				Counts:    count,
			}
		}
	}
	return cp
}

// TestMainWithCoverage is only for our own testing, in order to gather "more
// complete" coverage profile data including our M.Run()/M.run() methods.
//
// We achieve this with an unfortunate hack: we update the already written
// coverage data after the fact, that is, after mm.run() (or its public
// mm.Run() facade) has called gotesting.M.Run() which in turns writes the
// coverage profile data. This way, we can also get coverage of the code parts
// of ours M.run() which run after gotesting.M.Run().
func TestMainWithCoverage(m *gotesting.M) {
	mm := &M{M: m, skipCleanup: true}
	exitcode, reexeced := mm.run()
	if coverProfile != "" {
		// Take the final coverage profile data as our starting point, ignoring
		// whatever mm.run() wrote to the final coverage file. We need to write a
		// new version of it with the most recent coverage profile data.
		cp := coverageProfileFromTestingCover()
		var merges []string
		if !reexeced {
			merges = testsupport.CoverageProfiles
		}
		mergeWithCoverProfileAndReport(cp, merges, coverProfile)
		for _, coverprof := range merges {
			_ = os.Remove(toOutputDir(coverprof))
		}
	}
	os.Exit(exitcode)
}
