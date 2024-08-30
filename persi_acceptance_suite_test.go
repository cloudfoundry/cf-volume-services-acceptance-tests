package persi_acceptance_test

import (
	"fmt"
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

	appPath                                         string
	brokerName                                      string
	testValues                                      persiTestValues
	smbTestValues, nfsTestValues, nfsLDAPTestValues persiTestValues
)

type persiTestValues struct {
	validCreateServiceConfig            string
	createServiceConfigWithInvalidShare string
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

	smbTestValues = persiTestValues{
		validCreateServiceConfig:            `{"share": "//smbtestserver.service.cf.internal/vol"}`,
		secondAppValidBindConfig:            `{}`,
		bindConfigWithInvalidKeys:           `{"uid": "1000", "gid": "1000"}`,
		bindConfigWithInvalidKeysFailure:    "Service broker error: - Not allowed options: gid, uid",
		createServiceConfigWithInvalidShare: `{"share": "//meow.smbtestserver.this.does.not.exist.cf.internal/vol"}`,
		validBindConfigs: []string{
			fmt.Sprintf(`{"username": "%s", "password": "%s", "domain": "foo"}`, pConfig.Username, pConfig.Password),
			fmt.Sprintf(`{"username": "%s", "password": "%s"}`, pConfig.Username, pConfig.Password), // TODO pass user/pass thru config unsure if valid without domain
			fmt.Sprintf(`{"username": "%s", "password": "%s", "mount": "/var/vcap/data/foo", "domain": "foo"}`, pConfig.Username, pConfig.Password),
			fmt.Sprintf(`{"username": "%s", "password": "%s", "domain": "foo", "version": "3"}`, pConfig.Username, pConfig.Password),
			fmt.Sprintf(`{"username": "%s", "password": "%s", "domain": "foo", "version": "3.1.1"}`, pConfig.Username, pConfig.Password),
		},
	}

	nfsTestValues = persiTestValues{
		validCreateServiceConfig:            `{"share": "nfstestserver.service.cf.internal/export/users"}`,
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
	nfsLDAPTestValues = persiTestValues{
		validCreateServiceConfig:            `{"share": "nfstestldapserver.service.cf.internal/export/users"}`,
		secondAppValidBindConfig:            `{"username": "user2000", "password": "secret"}`, // TODO: this was added this user manually, need to add progomatically and pass value thru config
		bindConfigWithInvalidKeys:           `{"domain":"foo"}`,
		bindConfigWithInvalidKeysFailure:    "Service broker error: - Not allowed options: domain",
		createServiceConfigWithInvalidShare: `{"share": "nfstestldapserver.service.cf.internal/meow-meow-this-doesnt-exist", "username": "1000", "password": "secret"}`,
		validBindConfigs: []string{
			fmt.Sprintf(`{"username": "%s", "password": "%s"}`, pConfig.Username, pConfig.Password),
			fmt.Sprintf(`{"username": "%s", "password": "%s", "mount": "/var/vcap/data/foo"}`, pConfig.Username, pConfig.Password),
			fmt.Sprintf(`{"username": "%s", "password": "%s", "version": "3"}`, pConfig.Username, pConfig.Password),
			fmt.Sprintf(`{"username": "%s", "password": "%s", "version": "4.0"}`, pConfig.Username, pConfig.Password),
			fmt.Sprintf(`{"username": "%s", "password": "%s", "version": "4.1"}`, pConfig.Username, pConfig.Password),
			fmt.Sprintf(`{"username": "%s", "password": "%s", "version": "4.2"}`, pConfig.Username, pConfig.Password),
		},
	}

	if pConfig.ServiceName == "nfs" && pConfig.IsLDAP {
		testValues = nfsLDAPTestValues
	} else if pConfig.ServiceName == "nfs" {
		testValues = nfsTestValues
	} else if pConfig.ServiceName == "smb" {
		testValues = smbTestValues
	} else {
		Expect(pConfig.ServiceName).To(BeElementOf([]string{"nfs", "smb"}))
	}
})

var _ = AfterSuite(func() {
	if cfTestSuiteSetup != nil {
		cfTestSuiteSetup.Teardown()
	}
})
