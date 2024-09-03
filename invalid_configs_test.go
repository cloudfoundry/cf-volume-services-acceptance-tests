package persi_acceptance_test

import (
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Invalid Configs", func() {
	var (
		appName, instanceName string
	)

	BeforeEach(func() {
		instanceName, appName, _ = generateTestNames()

		By("Enabling service-access")
		enableServiceAccess(pConfig.ServiceName, cfTestSuiteSetup.RegularUserContext().Org)

		By("Pushing an app")
		pushPoraNoStart(appName, false)
	})

	AfterEach(func() {
		workflowhelpers.AsUser(cfTestSuiteSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
			cf.Cf("purge-service-instance", instanceName, "-f").Wait(DEFAULT_TIMEOUT)
			cf.Cf("unbind-service", appName, instanceName).Wait(DEFAULT_TIMEOUT)
			cf.Cf("delete-service", instanceName, "-f").Wait(DEFAULT_TIMEOUT)
			Eventually(func() *Session {
				serviceDetails := cf.Cf("services").Wait(DEFAULT_TIMEOUT)
				Expect(serviceDetails).To(Exit(0))
				return serviceDetails
			}, LONG_TIMEOUT, POLL_INTERVAL).Should(Not(Say(instanceName)))
			cf.Cf("delete", appName, "-r", "-f").Wait(DEFAULT_TIMEOUT)
		})
	})

	Context("with a bind config with invalid keys", func() {
		BeforeEach(func() {
			By("Creating a valid service")
			createService(instanceName, testValues.validCreateServiceConfig)
		})

		It("fails to bind", func() {
			workflowhelpers.AsUser(cfTestSuiteSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
				bindResponse := cf.Cf("bind-service", appName, instanceName, "-c", testValues.bindConfigWithInvalidKeys).Wait(DEFAULT_TIMEOUT)
				Expect(bindResponse).NotTo(Exit(0))
				Eventually(bindResponse.Err).Should(Say(testValues.bindConfigWithInvalidKeysFailure))
			})
		})
	})

	// TODO test more invalid configurations
	Context("with a bind config with valid keys, but invalid values", func() {
		BeforeEach(func() {
			By("Creating an invalid service")
			createService(instanceName, testValues.createServiceConfigWithInvalidShare)

			By("Binding the service to the app")
			bindAppToService(appName, instanceName, testValues.validBindConfigs[0])
		})

		It("fails to start", func() {
			workflowhelpers.AsUser(cfTestSuiteSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
				startResponse := cf.Cf("start", appName).Wait(LONG_TIMEOUT)
				Expect(startResponse).To(Exit(1))

				Eventually(cf.Cf("logs", appName, "--recent").Wait(DEFAULT_TIMEOUT)).Should(Say("failed to mount volume, errors:"))
			})
		})
	})
})
