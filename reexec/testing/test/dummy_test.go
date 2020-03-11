package test

import (
	"os"
	"testing"

	_ "github.com/thediveo/gons/reexec" // needed, otherwise mm.Run() bombs...
	rxtst "github.com/thediveo/gons/reexec/testing"
)

// This just tests the "standard" coverage-enabled mm.Run() method.
func TestMain(m *testing.M) {
	mm := &rxtst.M{M: m}
	os.Exit(mm.Run())
}
