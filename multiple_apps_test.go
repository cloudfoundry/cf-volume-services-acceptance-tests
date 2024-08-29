package persi_acceptance_test

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("multiple apps uses volume services", func() {
	var (
		readWriteAppURL, appName, instanceName string
		app2Name, fname, app2URL               string
	)

	BeforeEach(func() {
		instanceName, appName, readWriteAppURL = generateTestNames()
		app2Name = appName + "-2"
		app2URL = "http://" + app2Name + "." + cfConfig.AppsDomain
	})

	Context("when a second app is bound with a different uid and gid", func() {
		BeforeEach(func() {
			if pConfig.ServiceName != "nfs" {
				Skip("skip when using smb")
			}

			By("Enabling serivice-access")
			enableServiceAccess(pConfig.ServiceName, cfTestSuiteSetup.RegularUserContext().Org)

			By("Pushing the first app")
			pushPoraNoStart(appName, false)

			By("Creating a service")
			createService(instanceName, testValues.validCreateServiceConfig)

			By("Binding the service to the first app")
			bindAppToService(appName, instanceName, testValues.validBindConfig)

			By("Starting the first app")
			startApp(appName)

			By("Writing something to the mount with the first app")
			fname = eventuallyExpect(readWriteAppURL+"/create", "pora")
		})

		AfterEach(func() {
			eventuallyExpect(fmt.Sprintf("%s/delete/%s", readWriteAppURL, fname), fname)
			workflowhelpers.AsUser(cfTestSuiteSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
				cf.Cf("unbind-service", appName, instanceName).Wait(DEFAULT_TIMEOUT)
				cf.Cf("unbind-service", app2Name, instanceName).Wait(DEFAULT_TIMEOUT)
				cf.Cf("delete-service", instanceName, "-f").Wait(DEFAULT_TIMEOUT)
				cf.Cf("delete", appName, "-r", "-f").Wait(DEFAULT_TIMEOUT)
				cf.Cf("delete", app2Name, "-r", "-f").Wait(DEFAULT_TIMEOUT)
			})
		})

		DescribeTable("can be read by the 2nd app",
			func(testDocker bool) {
				By("Pushing a 2nd app")
				app2Name = appName + "-2"
				pushPoraNoStart(app2Name, testDocker)

				By("Binding the service to the 2nd app")
				bindAppToService(app2Name, instanceName, testValues.secondAppValidBindConfig)

				By("Starting the 2nd app")
				startApp(app2Name)
				app2URL = "http://" + app2Name + "." + cfConfig.AppsDomain

				By("Testing that the 2nd app can read the file")
				eventuallyExpect(fmt.Sprintf("%s/read/%s", app2URL, fname), "Hello Persistent World")

				By("Testing that the 2nd app can't delete the file")
				body, status, _ := get(fmt.Sprintf("%s/delete/%s", app2URL, fname), printErrorsOn)
				Expect(body).NotTo(ContainSubstring("deleted"))
				Expect(status).NotTo(Equal(http.StatusOK))
			},
			Entry("When using a docker app", true),
			Entry("When using a non-docker app", false),
		)
	})

	Context("when a second app is bound with a readonly mount", func() {
		BeforeEach(func() {
			// if os.Getenv("TEST_READ_ONLY") == "true" { } // TODO: leaving for now as a reminder for other configs
			By("Enabling serivice-access")
			enableServiceAccess(pConfig.ServiceName, cfTestSuiteSetup.RegularUserContext().Org)

			By("Pushing the first app")
			pushPoraNoStart(appName, false)

			By("Creating a service")
			createService(instanceName, testValues.validCreateServiceConfig)

			By("Binding the service to the first app")
			bindAppToService(appName, instanceName, testValues.validBindConfig)

			By("Starting the first app")
			startApp(appName)

			By("Writing something to the mount with the first app")
			fname = eventuallyExpect(readWriteAppURL+"/create", "pora")
		})

		AfterEach(func() {
			eventuallyExpect(fmt.Sprintf("%s/delete/%s", readWriteAppURL, fname), fname)
			workflowhelpers.AsUser(cfTestSuiteSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
				cf.Cf("unbind-service", appName, instanceName).Wait(DEFAULT_TIMEOUT)
				cf.Cf("unbind-service", app2Name, instanceName).Wait(DEFAULT_TIMEOUT)
				cf.Cf("delete-service", instanceName, "-f").Wait(DEFAULT_TIMEOUT)
				cf.Cf("delete", appName, "-r", "-f").Wait(DEFAULT_TIMEOUT)
				cf.Cf("delete", app2Name, "-r", "-f").Wait(DEFAULT_TIMEOUT)
				cf.Cf("disable-service-access", pConfig.ServiceName, "-o", cfTestSuiteSetup.RegularUserContext().Org).Wait(DEFAULT_TIMEOUT)
			})
		})

		DescribeTable("should include the volume mount as read only in the second application's environment",
			func(testDocker bool) {
				By("Pushing a 2nd app")
				pushPoraNoStart(app2Name, testDocker)

				By("Binding the service to the 2nd app")
				readOnlyBindConfig := strings.Replace(testValues.validBindConfig, "}", `,"readonly":true}`, 1)
				bindAppToService(app2Name, instanceName, readOnlyBindConfig)

				By("Starting the 2nd app")
				startApp(app2Name)

				By("Testing that the 2nd app cannot write to the file system")
				body, _, _ := get(app2URL+"/create", printErrorsOff)
				Expect(body).To(ContainSubstring("read-only file system"))

				By("Testing that when the first app creates a file, should be readable by the 2nd app")
				fname := eventuallyExpect(readWriteAppURL+"/create", "pora")
				eventuallyExpect(fmt.Sprintf("%s/read/%s", app2URL, fname), "Hello Persistent World")
			},
			Entry("When using a docker app", true),
			Entry("When using a non-docker app", false),
		)
	})
})
