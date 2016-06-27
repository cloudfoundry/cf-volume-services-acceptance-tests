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
	LONG_TIMEOUT = 300 * time.Second

	BROKER_URL = "http://cephbroker.persi.cf-app.com:8999"
	BROKER_NAME = "patscephbroker"

	SERVICE_NAME = "cephfs"
	PLAN_NAME = "free"
	INSTANCE_NAME = "mycephfs"
	APP_NAME = "pora"

	APP_HOST = "pora.persi.cf-app.com"

	APP_URL = "http://" + APP_HOST
)

func TestPersiAcceptance(t *testing.T) {
	RegisterFailHandler(Fail)

	patsConfig = helpers.LoadConfig()
	defaults(&patsConfig)
	patsContext = helpers.NewContext(patsConfig)
	environment := helpers.NewEnvironment(patsContext)

	BeforeSuite(func() {
		cf.AsUser(patsContext.AdminUserContext(), DEFAULT_TIMEOUT, func() {
			cf.Cf("delete-route", APP_HOST).Wait(DEFAULT_TIMEOUT)
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

	config.NamePrefix = "ginkgoPATS"
}
