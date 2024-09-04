package persi_acceptance_test

import (
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Cloud Foundry Persistence", func() {

	Context("given a service broker", func() {
		AfterEach(func() {
			workflowhelpers.AsUser(cfTestSuiteSetup.AdminUserContext(), DEFAULT_TIMEOUT, func() {
				publishService := cf.Cf("disable-service-access", pConfig.ServiceName, "-o", cfTestSuiteSetup.RegularUserContext().Org).Wait(DEFAULT_TIMEOUT)
				Expect(publishService).To(Exit(0))
			})
		})

		It("testing service-access enablement", func() {
			By("Testing that a service-broker already exists")
			workflowhelpers.AsUser(cfTestSuiteSetup.AdminUserContext(), DEFAULT_TIMEOUT, func() {
				serviceBrokers := cf.Cf("service-brokers").Wait(DEFAULT_TIMEOUT)
				Expect(serviceBrokers).To(Exit(0))
				Expect(serviceBrokers).To(Say(brokerName))
			})

			By("Testing that by default there is no service-access to all")
			// If this fails it might be for test pollution reasons
			workflowhelpers.AsUser(cfTestSuiteSetup.AdminUserContext(), DEFAULT_TIMEOUT, func() {
				serviceAccess := cf.Cf("service-access").Wait(DEFAULT_TIMEOUT)
				Expect(serviceAccess).To(Exit(0))
				Expect(serviceAccess).To(Say(brokerName))
				Expect(serviceAccess).NotTo(Say(pConfig.ServiceName + ".*" + pConfig.PlanName + ".*" + "all"))
			})

			By("Testing that there is no service-access to this org")
			workflowhelpers.AsUser(cfTestSuiteSetup.AdminUserContext(), DEFAULT_TIMEOUT, func() {
				serviceAccess := cf.Cf("service-access").Wait(DEFAULT_TIMEOUT)
				Expect(serviceAccess).To(Exit(0))
				Expect(serviceAccess).To(Say(brokerName))
				Expect(serviceAccess).To(Say(pConfig.ServiceName + ".*" + pConfig.PlanName + ".*"))
				Expect(serviceAccess).NotTo(Say(cfTestSuiteSetup.RegularUserContext().Org))
			})

			By("Testing enabling service-access")
			workflowhelpers.AsUser(cfTestSuiteSetup.AdminUserContext(), DEFAULT_TIMEOUT, func() {
				publishService := cf.Cf("enable-service-access", pConfig.ServiceName, "-o", cfTestSuiteSetup.RegularUserContext().Org, "-b", pConfig.BrokerName).Wait(DEFAULT_TIMEOUT)
				Expect(publishService).To(Exit(0))
			})

			By("Testing that service-access is enabled")
			workflowhelpers.AsUser(cfTestSuiteSetup.AdminUserContext(), DEFAULT_TIMEOUT, func() {
				serviceAccess := cf.Cf("service-access").Wait(DEFAULT_TIMEOUT)
				Expect(serviceAccess).To(Exit(0))
				Expect(serviceAccess).To(Say(brokerName))
				Expect(serviceAccess).To(Say(pConfig.ServiceName + ".*" + pConfig.PlanName + ".*limited.*" + cfTestSuiteSetup.RegularUserContext().Org))
			})

			By("Testing that the service is in the marketplace")
			workflowhelpers.AsUser(cfTestSuiteSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
				marketplaceItems := cf.Cf("marketplace").Wait(DEFAULT_TIMEOUT)
				Expect(marketplaceItems).To(Exit(0))
				Expect(marketplaceItems).To(Say(pConfig.ServiceName))
				Expect(marketplaceItems).To(Say(pConfig.PlanName))
			})
		})
	})
})
