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
	var (
		readWriteAppURL, appName, instanceName string
	)

	BeforeEach(func() {
		instanceName, appName, readWriteAppURL = generateTestNames()
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

	DescribeTable("given one valid bind config, it can mount volumes",
		func(testDocker bool) {
			By("Enabling serivice-access")
			enableServiceAccess(pConfig.ServiceName, cfTestSuiteSetup.RegularUserContext().Org)

			By("Pushing an app")
			pushPoraNoStart(appName, testDocker)

			By("Creating a service")
			createService(instanceName, testValues.validCreateServiceConfig)

			By("Binding the service to the app")
			bindAppToService(appName, instanceName, testValues.validBindConfigs[0])

			By("Starting the app")
			startApp(appName)

			By("Verifying that it responds to http requests")
			eventuallyExpect(readWriteAppURL, "instance index:")

			By("Verifying that the volume mount path is included in the application's environment")
			workflowhelpers.AsUser(cfTestSuiteSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
				env := cf.Cf("env", appName).Wait(DEFAULT_TIMEOUT)
				Expect(env).To(Exit(0))
				Expect(env).To(Say(pConfig.ServiceName))
				Expect(env).To(Say(instanceName))
				Expect(env).To(Or(Say("container_path"), Say("container_dir")))
			})

			By("Verifying that the app is able to write to the volume")
			eventuallyExpect(readWriteAppURL+"/write", "Hello Persistent World")
		},
		Entry("When using a docker app", true),
		Entry("When using a non-docker app", false),
	)
})
