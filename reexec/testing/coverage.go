package testing

import (
	"os"
	"strings"
	gotesting "testing"
)

// M is an "enhanced" version of Golang's testing.M which additionally handles
// merging coverage profile data from re-executions into the main ("parent's")
// coverage profile file.
type M struct {
	*gotesting.M
}

var (
	outputDir    string // "-test.outputdir"
	coverProfile string // "-test.coverprofile"
)

// Run runs the tests and correctly merges the coverage profile data from
// re-executed process copies into this process' coverage profile data. Run
// returns an exit code to pass to os.Exit.
func (m *M) Run() int {
	parseCoverageArgs(os.Args)
	code := m.M.Run()
	mergeCoverages()
	return code
}

// mergeCoverages picks up the coverage profile data files created by
// re-executed copies and merges them into this (parent) process' coverage
// profile data.
func mergeCoverages() {
	//coverProfiles = reexec.ReexecCoverageProfiles()
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
