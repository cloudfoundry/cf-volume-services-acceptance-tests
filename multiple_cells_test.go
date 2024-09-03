package persi_acceptance_test

import (
	"bytes"
	"strconv"
	"sync"
	"time"

	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("when there are multiple Diego Cells", func() {
	var (
		readWriteAppURL, appName, instanceName string
	)

	BeforeEach(func() {
		if !pConfig.IncludeMultiCell {
			Skip("skipping multi cell")
		}

		instanceName, appName, readWriteAppURL = generateTestNames()

		By("Pushing an app bound to a service")
		enableServiceAccess(pConfig.ServiceName, cfTestSuiteSetup.RegularUserContext().Org)
		pushPoraNoStart(appName, false)
		createService(instanceName, testValues.validCreateServiceConfig)
		bindAppToService(appName, instanceName, testValues.validBindConfigs[0])
		startApp(appName)
	})

	It("should keep the data across multiple stops and starts", func() {
		fname := eventuallyExpect(readWriteAppURL+"/create", "pora")
		// start a bunch of simultaneous requests to do file io
		var wg sync.WaitGroup
		stop := make(chan bool)
		wg.Add(10)
		for i := 0; i < 10; i++ {
			go func() {
				defer wg.Done()
				for {
					get(readWriteAppURL+"/loadtest", printErrorsOff)
					time.Sleep(100 * time.Millisecond)
					select {
					case <-stop:
						return
					default:
					}
				}
			}()
		}

		workflowhelpers.AsUser(cfTestSuiteSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
			for i := 0; i < 20; i++ {
				stopResponse := cf.Cf("stop", appName).Wait(DEFAULT_TIMEOUT)
				Expect(stopResponse).To(Exit(0))
				startResponse := cf.Cf("start", appName).Wait(LONG_TIMEOUT)
				Expect(startResponse).To(Exit(0))
			}
		})

		// signal our background load to stop and then wait for it
		close(stop)
		wg.Wait()

		eventuallyExpect(readWriteAppURL+"/read/"+fname, "Hello Persistent World")
		eventuallyExpect(readWriteAppURL+"/delete/"+fname, fname)

		// clean up any load test files that got left behind on the mount due to apps stopping
		// and starting
		get(readWriteAppURL+"/loadtestcleanup", printErrorsOn)
	})

	Context("when the app is scaled across cells", func() {
		const appScale = 5
		BeforeEach(func() {
			workflowhelpers.AsUser(cfTestSuiteSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
				bindResponse := cf.Cf("scale", appName, "-i", strconv.Itoa(appScale)).Wait(LONG_TIMEOUT)
				Expect(bindResponse).To(Exit(0))

				Eventually(func() int {
					apps := cf.Cf("app", appName).Wait(DEFAULT_TIMEOUT)
					Expect(apps).To(Exit(0))
					return bytes.Count(apps.Out.Contents(), []byte("running"))
				}, LONG_TIMEOUT, POLL_INTERVAL).Should(Equal(appScale))
			})
		})

		It("should be able to create a test file then read it from any instance", func() {
			fname := eventuallyExpect(readWriteAppURL+"/create", "pora")

			responses := map[string]int{}
			for i := 0; i < appScale*10000; i++ {
				body := eventuallyExpect(readWriteAppURL+"/read/"+fname, "Hello Persistent World")
				responses[body] = 1
				if len(responses) >= appScale {
					break
				}
			}
			eventuallyExpect(readWriteAppURL+"/delete/"+fname, fname)

			Expect(len(responses)).To(Equal(appScale))
		})
	})
})
