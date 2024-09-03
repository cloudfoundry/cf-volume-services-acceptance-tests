package persi_acceptance_test

import (
	"fmt"
	persi_acceptance "persi_acceptance_test"
	"testing"
	"time"

	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/config"
	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
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
	invalidCreateConfig                 string
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
		invalidCreateConfig:                 `{"meow": "I don't have a share"}`,
		validCreateServiceConfig:            `{"share": "//smbtestserver.service.cf.internal/vol"}`,
		secondAppValidBindConfig:            `{}`,
		bindConfigWithInvalidKeys:           `{"uid": "1000", "gid": "1000"}`,
		bindConfigWithInvalidKeysFailure:    "Service broker error: - Not allowed options: gid, uid",
		createServiceConfigWithInvalidShare: `{"share": "//meow.smbtestserver.this.does.not.exist.cf.internal/vol"}`,
		validBindConfigs: []string{
			fmt.Sprintf(`{"username": "%s", "password": "%s", "domain": "foo"}`, pConfig.Username, pConfig.Password),
			fmt.Sprintf(`{"username": "%s", "password": "%s"}`, pConfig.Username, pConfig.Password),
			fmt.Sprintf(`{"username": "%s", "password": "%s", "mount": "/var/vcap/data/foo", "domain": "foo"}`, pConfig.Username, pConfig.Password),
			fmt.Sprintf(`{"username": "%s", "password": "%s", "domain": "foo", "version": "3"}`, pConfig.Username, pConfig.Password),
			fmt.Sprintf(`{"username": "%s", "password": "%s", "domain": "foo", "version": "3.1.1"}`, pConfig.Username, pConfig.Password),
		},
	}

	nfsTestValues = persiTestValues{
		invalidCreateConfig:                 `{"meow": "I don't have a share"}`,
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
	ldapUser := "user1000"
	ldapPassword := "secret" // these are hardcoded in nfstestldapserver
	nfsLDAPTestValues = persiTestValues{
		invalidCreateConfig:                 `{"meow": "I don't have a share"}`,
		validCreateServiceConfig:            `{"share": "nfstestldapserver.service.cf.internal/export/users"}`,
		secondAppValidBindConfig:            `{"username": "user2000", "password": "secret"}`, // these are hardcoded in nfstestldapserver
		bindConfigWithInvalidKeys:           `{"domain":"foo"}`,
		bindConfigWithInvalidKeysFailure:    "Service broker error: - Not allowed options: domain",
		createServiceConfigWithInvalidShare: `{"share": "nfstestldapserver.service.cf.internal/meow-meow-this-doesnt-exist", "username": "1000", "password": "secret"}`,
		validBindConfigs: []string{
			fmt.Sprintf(`{"username": "%s", "password": "%s"}`, ldapUser, ldapPassword),
			fmt.Sprintf(`{"username": "%s", "password": "%s", "mount": "/var/vcap/data/foo"}`, ldapUser, ldapPassword),
			fmt.Sprintf(`{"username": "%s", "password": "%s", "version": "3"}`, ldapUser, ldapPassword),
			fmt.Sprintf(`{"username": "%s", "password": "%s", "version": "4.0"}`, ldapUser, ldapPassword),
			fmt.Sprintf(`{"username": "%s", "password": "%s", "version": "4.1"}`, ldapUser, ldapPassword),
			fmt.Sprintf(`{"username": "%s", "password": "%s", "version": "4.2"}`, ldapUser, ldapPassword),
		},
	}

	if pConfig.ServiceName == "nfs-ldap" {
		testValues = nfsLDAPTestValues
	} else if pConfig.ServiceName == "nfs" {
		testValues = nfsTestValues
	} else if pConfig.ServiceName == "smb" {
		testValues = smbTestValues
	} else {
		Expect(pConfig.ServiceName).To(BeElementOf([]string{"nfs-ldap", "nfs", "smb"}))
	}

	if pConfig.IncludeIsolationSegment {
		org := cfTestSuiteSetup.RegularUserContext().Org
		isolationSegment := "persistent_isolation_segment"
		workflowhelpers.AsUser(cfTestSuiteSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
			enableIso := cf.Cf("enable-org-isolation", org, isolationSegment).Wait(DEFAULT_TIMEOUT)
			Expect(enableIso).To(Exit(0))

			defaultIso := cf.Cf("set-org-default-isolation-segment", org, isolationSegment).Wait(DEFAULT_TIMEOUT)
			Expect(defaultIso).To(Exit(0))
		})
	}

})

var _ = AfterSuite(func() {
	if cfTestSuiteSetup != nil {
		cfTestSuiteSetup.Teardown()
	}
})
