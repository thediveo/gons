package testing

import (
	gotesting "testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMain(m *gotesting.M) {
	TestMainWithCoverage(m)
}

func TestPackage(t *gotesting.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "gons/reexec/testing package")
}
