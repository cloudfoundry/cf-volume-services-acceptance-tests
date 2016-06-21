package persi_acceptance_tests_test

import (
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Sample", func() {
	It("cf version", func() {
		version := cf.Cf("version").Wait(patsConfig.DefaultTimeout)
		Expect(version).To(Exit(0))
		Expect(version).To(Say("abc"))
	})
})
