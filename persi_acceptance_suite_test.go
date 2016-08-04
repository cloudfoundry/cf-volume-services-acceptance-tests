package persi_acceptance_tests_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"testing"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"

	"time"
)

var (
	patsConfig       helpers.Config
	patsSuiteContext helpers.SuiteContext

	DEFAULT_TIMEOUT = 30 * time.Second
	LONG_TIMEOUT    = 300 * time.Second

	brokerName = "pats-broker"

	serviceName = "pats-service"
	planName    = "pats-plan"
	brokerUrl   string
)

func TestPersiAcceptance(t *testing.T) {
	RegisterFailHandler(Fail)

	patsConfig = helpers.LoadConfig()
	defaults(&patsConfig)

	if patsConfig.NamePrefix != "CATS" && patsConfig.NamePrefix != "" {
		patsConfig.NamePrefix = patsConfig.NamePrefix + "-ginkgoPATS"
		brokerUrl = "http://pats-broker." + patsConfig.NamePrefix + "." + patsConfig.AppsDomain
		serviceName = patsConfig.NamePrefix + "-" + serviceName
		planName = patsConfig.NamePrefix + "-" + planName
	} else {
		patsConfig.NamePrefix = "ginkgoPATS"
		brokerUrl = "http://pats-broker." + patsConfig.AppsDomain
	}

	brokerName = patsConfig.NamePrefix + "-" + brokerName
	componentName := "PATS Suite"
	rs := []Reporter{}

	SynchronizedBeforeSuite(func() []byte {
		patsSuiteContext = helpers.NewContext(patsConfig)

		cf.AsUser(patsSuiteContext.AdminUserContext(), DEFAULT_TIMEOUT, func() {
			// make sure we don't have a leftover service broker from another test
			cf.Cf("delete-service-broker", "-f", brokerName).Wait(DEFAULT_TIMEOUT)
			createServiceBroker := cf.Cf("create-service-broker", brokerName, patsConfig.AdminUser, patsConfig.AdminPassword, brokerUrl).Wait(DEFAULT_TIMEOUT)
			Expect(createServiceBroker).To(Exit(0))
			Expect(createServiceBroker).To(Say(brokerName))
		})

		return nil
	}, func(_ []byte) {})

	SynchronizedAfterSuite(func() {}, func() {
		cf.AsUser(patsSuiteContext.AdminUserContext(), DEFAULT_TIMEOUT, func() {
			cf.Cf("delete-service-broker", "-f", brokerName).Wait(DEFAULT_TIMEOUT)
		})
	})

	if patsConfig.ArtifactsDirectory != "" {
		helpers.EnableCFTrace(patsConfig, componentName)
		rs = append(rs, helpers.NewJUnitReporter(patsConfig, componentName))
	}

	RunSpecsWithDefaultAndCustomReporters(t, componentName, rs)
}

func defaults(config *helpers.Config) {
	if config.DefaultTimeout > 0 {
		DEFAULT_TIMEOUT = config.DefaultTimeout * time.Second
	}
}
