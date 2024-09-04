package persi_acceptance_test

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

const printErrorsOn = true
const printErrorsOff = false
const instanceNameBase = "pats-volume-instance"
const appNameBase = "pats-pora"

func get(uri string, printErrors bool) (body string, status int, err error) {
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return "", status, err
	}

	response, err := (&http.Client{}).Do(req)
	if err != nil {
		return "", status, err
	}

	bodyBytes, err := io.ReadAll(response.Body)
	defer response.Body.Close()

	if printErrors && response.StatusCode >= http.StatusInternalServerError {
		fmt.Printf("Request: [[%v]]\nResponse: [[%v]] [[%s]]\n", req, response, string(bodyBytes))
	}

	return string(bodyBytes[:]), response.StatusCode, err
}

func eventuallyExpect(endpoint string, expectedSubstring string) string {
	EventuallyWithOffset(1, func() int {
		_, status, err := get(endpoint, printErrorsOn)
		Expect(err).NotTo(HaveOccurred())
		return status
	}, 5*time.Second, 1*time.Second).Should(Equal(http.StatusOK))

	var body string
	Eventually(func() string {
		var err error
		body, _, err = get(endpoint, printErrorsOn)
		Expect(err).NotTo(HaveOccurred())
		return body
	}, 5*time.Second, 1*time.Second).Should(ContainSubstring(expectedSubstring))

	return body
}

func enableServiceAccess(serviceName, org string) {
	workflowhelpers.AsUser(cfTestSuiteSetup.AdminUserContext(), DEFAULT_TIMEOUT, func() {
		publishService := cf.Cf("enable-service-access", serviceName, "-o", org).Wait(DEFAULT_TIMEOUT)
		Expect(publishService).To(Exit(0))
	})
}

func pushPoraNoStart(a string, dockerApp bool) {
	workflowhelpers.AsUser(cfTestSuiteSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
		if dockerApp {
			Eventually(cf.Cf("push", a, "--docker-image", "cfpersi/pora", "--no-start"), DEFAULT_TIMEOUT).Should(Exit(0))
		} else {
			Eventually(cf.Cf("push", a, "-p", appPath, "-f", appPath+"/manifest.yml", "--no-start"), DEFAULT_TIMEOUT).Should(Exit(0))
		}
	})

	workflowhelpers.AsUser(cfTestSuiteSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
		marketplaceItems := cf.Cf("apps").Wait(DEFAULT_TIMEOUT)
		Expect(marketplaceItems).To(Exit(0))
		Expect(marketplaceItems).To(Say(a))
	})
}

func createService(s, c string) {
	workflowhelpers.AsUser(cfTestSuiteSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
		Eventually(func() *Session {
			createService := cf.Cf("create-service", pConfig.ServiceName, pConfig.PlanName, s, "-c", c, "-b", pConfig.BrokerName).Wait(DEFAULT_TIMEOUT)
			Expect(createService).To(Exit(0))

			serviceDetails := cf.Cf("service", s).Wait(DEFAULT_TIMEOUT)
			Expect(serviceDetails).To(Exit(0))
			return serviceDetails
		}, LONG_TIMEOUT, POLL_INTERVAL).Should(Say("create succeeded"))
	})
}

func bindAppToService(a, s, c string) {
	workflowhelpers.AsUser(cfTestSuiteSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
		bindResponse := cf.Cf("bind-service", a, s, "-c", c).Wait(DEFAULT_TIMEOUT)
		Expect(bindResponse).To(Exit(0))

		services := cf.Cf("services").Wait(DEFAULT_TIMEOUT)
		Expect(services).To(Exit(0))
		Expect(services).To(Say(s + "[^\\n]+" + pConfig.ServiceName + "[^\\n]+" + a))
	})
}

func startApp(a string) {
	workflowhelpers.AsUser(cfTestSuiteSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
		bindResponse := cf.Cf("start", a).Wait(LONG_TIMEOUT)
		Expect(bindResponse).To(Exit(0))
	})
}

func stopApp(a string) {
	workflowhelpers.AsUser(cfTestSuiteSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
		bindResponse := cf.Cf("stop", a).Wait(LONG_TIMEOUT)
		Expect(bindResponse).To(Exit(0))
	})
}

func generateTestNames() (instanceName, appName, appURL string) {
	parallelNode := strconv.Itoa(GinkgoParallelProcess())

	uuid := uuid.NewString()

	instanceName = uuid + "-" + instanceNameBase + parallelNode
	appName = uuid + "-" + appNameBase + parallelNode

	appHost := appName + "." + cfConfig.AppsDomain
	appURL = "http://" + appHost

	return instanceName, appName, appURL
}
