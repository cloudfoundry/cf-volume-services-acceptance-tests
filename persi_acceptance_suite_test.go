package persi_acceptance_tests_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"

	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"

	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
)

var (
	patsContext helpers.SuiteContext
	patsConfig  helpers.Config

	DEFAULT_TIMEOUT = 30 * time.Second
	LONG_TIMEOUT    = 300 * time.Second

	BrokerURL, AppHost, AppURL string

	BROKER_NAME  = "pats-broker"
	SERVICE_NAME = "pats-service"
	PLAN_NAME    = "pats-plan"

	INSTANCE_NAME = "pats-volume-instance"
	APP_NAME      = "pats-pora"
)

func TestPersiAcceptance(t *testing.T) {
	RegisterFailHandler(Fail)

	patsConfig = helpers.LoadConfig()
	defaults(&patsConfig)
	if patsConfig.NamePrefix != "" {
		patsConfig.NamePrefix = patsConfig.NamePrefix + "-ginkgoPATS"
		BROKER_NAME = patsConfig.NamePrefix + BROKER_NAME
		SERVICE_NAME = patsConfig.NamePrefix + SERVICE_NAME
		PLAN_NAME = patsConfig.NamePrefix + PLAN_NAME
		INSTANCE_NAME = patsConfig.NamePrefix + INSTANCE_NAME
		APP_NAME = patsConfig.NamePrefix + APP_NAME
	} else {
		patsConfig.NamePrefix = "ginkgoPATS"
	}

	patsContext = helpers.NewContext(patsConfig)
	environment := helpers.NewEnvironment(patsContext)

	BrokerURL = "http://pats-broker." + patsConfig.AppsDomain
	AppHost = APP_NAME + "." + patsConfig.AppsDomain
	AppURL = "http://" + AppHost

	BeforeSuite(func() {
		cf.AsUser(patsContext.AdminUserContext(), DEFAULT_TIMEOUT, func() {
			cf.Cf("delete-route", AppHost).Wait(DEFAULT_TIMEOUT)
			cf.Cf("delete-service-broker", "-f", BROKER_NAME).Wait(DEFAULT_TIMEOUT)
		})
		environment.Setup()
	})

	AfterSuite(func() {
		environment.Teardown()
	})

	componentName := "PATS Suite"
	rs := []Reporter{}

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
