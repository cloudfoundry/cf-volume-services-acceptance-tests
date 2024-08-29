package persi_acceptance_test

import (
	persi_acceptance "persi_acceptance_test"
	"testing"
	"time"

	"github.com/cloudfoundry/cf-test-helpers/v2/config"
	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	cfConfig         *config.Config
	pConfig          persi_acceptance.Config
	cfTestSuiteSetup *workflowhelpers.ReproducibleTestSuiteSetup

	DEFAULT_TIMEOUT = 30 * time.Second
	LONG_TIMEOUT    = 600 * time.Second
	POLL_INTERVAL   = 3 * time.Second

	appPath       string
	brokerName    string
	testValues    persiTestValues
	nfsTestValues = persiTestValues{
		validCreateServiceConfig:            `{"share": "nfstestserver.service.cf.internal/export/users"}`,
		validBindConfig:                     `{"gid": "1000", "uid": "1000"}`,
		secondAppValidBindConfig:            `{"uid":"5000","gid":"5000"}`,
		bindConfigWithInvalidKeys:           `{"domain":"foo"}`,
		bindConfigWithInvalidKeysFailure:    "Service broker error: - Not allowed options: domain",
		createServiceConfigWithInvalidShare: `{"share": "nfstestserver.service.cf.internal/meow-meow-this-doesnt-exist"}`,
		validBindConfigs: []string{
			`{"uid": "1000", "gid": "1000"}`,
			`{"uid": "1000", "gid": "1000", "mount": "/var/vcap/data/foo"}`,
			`{"uid": "1000", "gid": "1000", "version": "3"}`,
			`{"uid": "1000", "gid": "1000", "version": "4.0"}`,
			`{"uid": "1000", "gid": "1000", "version": "4.1"}`,
			`{"uid": "1000", "gid": "1000", "version": "4.2"}`,
		},
	}
)

type persiTestValues struct {
	validCreateServiceConfig            string
	createServiceConfigWithInvalidShare string
	validBindConfig                     string
	validBindConfigs                    []string
	bindConfigWithInvalidKeys           string
	bindConfigWithInvalidKeysFailure    string
	secondAppValidBindConfig            string
}

func TestPersiAcceptance(t *testing.T) {
	RegisterFailHandler(Fail)
	//TODO: is this needed?
	SetDefaultEventuallyTimeout(5 * time.Second)

	RunSpecs(t, "Cf Volume Services Acceptance Tests")
}

var _ = BeforeSuite(func() {
	var err error
	pConfig, err = persi_acceptance.LoadConfig()
	Expect(err).NotTo(HaveOccurred())
	cfConfig = config.LoadConfig()
	cfTestSuiteSetup = workflowhelpers.NewTestSuiteSetup(cfConfig)
	cfTestSuiteSetup.Setup()
	brokerName = pConfig.ServiceName + "broker"
	appPath = "assets/pora"
	if pConfig.ServiceName == "nfs" {
		testValues = nfsTestValues
	}
})

var _ = AfterSuite(func() {
	if cfTestSuiteSetup != nil {
		cfTestSuiteSetup.Teardown()
	}
})
