package testing

import (
	"os"
	gotesting "testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestMain(m *gotesting.M) {
	// We eat our own dog food here...
	mm := &M{M: m}
	os.Exit(mm.Run())
}

func TestPackage(t *gotesting.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "gons/reexec/testing package")
}
