package testing

import (
	"os"
	"testing"
	gotesting "testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestMain(m *gotesting.M) {
	//os.Exit(M{M: m}.Run())
	os.Exit(m.Run())
}

func TestPackage(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "gons/reexec/testing package")
}
