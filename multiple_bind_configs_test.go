package persi_acceptance_test

import (
	"fmt"

	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Multiple bind configs", func() {
	var (
		readWriteAppURL, appName, instanceName string
		validBindConfigs                       []string
	)

	BeforeEach(func() {
		validBindConfigs = testValues.validBindConfigs
		Expect(validBindConfigs).To(HaveLen(6)) // hardcoded value for nfs

		instanceName, appName, readWriteAppURL = generateTestNames()

		By("Enabling serivice-access")
		enableServiceAccess(pConfig.ServiceName, cfTestSuiteSetup.RegularUserContext().Org)

		By("Pushing the first app")
		pushPoraNoStart(appName, false)

		By("Creating a service")
		createService(instanceName, testValues.validCreateServiceConfig)
	})

	AfterEach(func() {
		workflowhelpers.AsUser(cfTestSuiteSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
			cf.Cf("logs", appName, "--recent").Wait(DEFAULT_TIMEOUT)
			cf.Cf("stop", appName).Wait(DEFAULT_TIMEOUT)
			cf.Cf("unbind-service", appName, instanceName).Wait(DEFAULT_TIMEOUT)
			cf.Cf("delete-service", instanceName, "-f").Wait(DEFAULT_TIMEOUT)
			cf.Cf("delete", appName, "-r", "-f").Wait(DEFAULT_TIMEOUT)
		})
	})

	It("works", func() {
		pushPoraNoStart(appName, false)

		for _, config := range validBindConfigs {
			By(fmt.Sprintf("Binding the service using config: %s", config))
			bindAppToService(appName, instanceName, testValues.validBindConfig)
			startApp(appName)
			eventuallyExpect(readWriteAppURL+"/write", "Hello Persistent World")
			stopApp(appName)
		}
	})
})
