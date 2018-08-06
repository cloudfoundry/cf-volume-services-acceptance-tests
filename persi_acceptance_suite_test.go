package persi_acceptance_tests_test

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/zbiljic/go-filelock"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
)

var (
	cfConfig         helpers.Config
	pConfig          patsConfig
	patsSuiteContext helpers.SuiteContext

	patsTestContext                           helpers.SuiteContext
	patsTestEnvironment, patsAdminEnvironment *helpers.Environment

	DEFAULT_TIMEOUT = 30 * time.Second
	LONG_TIMEOUT    = 600 * time.Second
	POLL_INTERVAL   = 3 * time.Second

	brokerName string
)

func TestPersiAcceptance(t *testing.T) {
	RegisterFailHandler(Fail)

	cfConfig = helpers.LoadConfig()
	defaults(&cfConfig)

	err := getPatsSpecificConfig()
	if err != nil {
		panic(err)
	}

	brokerName = pConfig.ServiceName + "-broker"

	componentName := "PATS Suite"
	rs := []Reporter{}
	maxParallelSetup := 5

	SynchronizedBeforeSuite(func() []byte {
		patsSuiteContext = helpers.NewContext(cfConfig)

		cf.AsUser(patsSuiteContext.AdminUserContext(), DEFAULT_TIMEOUT, func() {
			// make sure we don't have a leftover service broker from another test
			deleteBroker(pConfig.BrokerUrl)

			if os.Getenv("TEST_DOCKER_PORA") == "true" {
				Eventually(cf.Cf("enable-feature-flag", "diego_docker"), DEFAULT_TIMEOUT).Should(Exit(0))
			}
		})

		cf.AsUser(patsSuiteContext.AdminUserContext(), DEFAULT_TIMEOUT, func() {
			createServiceBroker := cf.Cf("create-service-broker", brokerName, pConfig.BrokerUser, pConfig.BrokerPassword, pConfig.BrokerUrl).Wait(DEFAULT_TIMEOUT)
			Expect(createServiceBroker).To(Exit(0))
			Expect(createServiceBroker).To(Say(brokerName))
		})

		lockFilePath, err := ioutil.TempDir("", "pats-setup-lock")
		Expect(err).ToNot(HaveOccurred())
		lockFilePath = filepath.Join(lockFilePath, "lock-")

		for i := 0; i < maxParallelSetup; i++ {
			d1 := []byte("this is a lock file")
			ioutil.WriteFile(lockFilePath+strconv.Itoa(i), d1, 0644)
		}

		return []byte(lockFilePath)
	}, func(path []byte) {
		lockFilePath := string(path)

		// rate limit spec setup to do no more than maxParallelSetup creates in parallel, so that CF doesn't get upset and time out on UAA calls
		fl, err := filelock.New(lockFilePath + strconv.Itoa(config.GinkgoConfig.ParallelNode%maxParallelSetup))
		Expect(err).ToNot(HaveOccurred())
		fl.Must().Lock()
		defer fl.Must().Unlock()

		patsTestContext = helpers.NewContext(cfConfig)
		patsTestEnvironment = helpers.NewEnvironment(patsTestContext)

		patsTestEnvironment.Setup()
		if pConfig.IsolationSegment != "" {
			cf.AsUser(patsTestContext.AdminUserContext(), DEFAULT_TIMEOUT, func() {
				Eventually(cf.Cf("create-isolation-segment", pConfig.IsolationSegment), DEFAULT_TIMEOUT).Should(Exit(0))
				Eventually(cf.Cf("enable-org-isolation", patsTestContext.RegularUserContext().Org, pConfig.IsolationSegment), DEFAULT_TIMEOUT).Should(Exit(0))
				Eventually(cf.Cf("set-org-default-isolation-segment", patsTestContext.RegularUserContext().Org, pConfig.IsolationSegment), DEFAULT_TIMEOUT).Should(Exit(0))
			})
		}
	})

	SynchronizedAfterSuite(func() {
		if patsTestEnvironment != nil {
			patsTestEnvironment.Teardown()
		}
	}, func() {
		cf.AsUser(patsSuiteContext.AdminUserContext(), DEFAULT_TIMEOUT, func() {
			if os.Getenv("TEST_DOCKER_PORA") == "true" {
				Eventually(cf.Cf("disable-feature-flag", "diego_docker"), DEFAULT_TIMEOUT).Should(Exit(0))
			}

			session := cf.Cf("delete-service-broker", "-f", brokerName).Wait(DEFAULT_TIMEOUT)
			if session.ExitCode() != 0 {
				cf.Cf("purge-service-offering", pConfig.ServiceName).Wait(DEFAULT_TIMEOUT)
				Fail("pats service broker could not be cleaned up.")
			}
		})
		if patsAdminEnvironment != nil {
			patsAdminEnvironment.Teardown()
		}
	})

	if cfConfig.ArtifactsDirectory != "" {
		helpers.EnableCFTrace(cfConfig, componentName)
		rs = append(rs, helpers.NewJUnitReporter(cfConfig, componentName))
	}

	RunSpecsWithDefaultAndCustomReporters(t, componentName, rs)
}

func deleteBroker(brokerUrl string) {
	serviceBrokers, err := exec.Command("cf", "curl", "/v2/service_brokers").Output()
	Expect(err).NotTo(HaveOccurred())

	var serviceBrokerResponse struct {
		Resources []struct {
			Entity struct {
				BrokerUrl string `json:"broker_url"`
				Name      string
			}
		}
	}

	Expect(json.Unmarshal(serviceBrokers, &serviceBrokerResponse)).To(Succeed())

	for _, broker := range serviceBrokerResponse.Resources {
		if broker.Entity.BrokerUrl == brokerUrl {
			cf.Cf("delete-service-broker", "-f", broker.Entity.Name).Wait(DEFAULT_TIMEOUT)
		}
	}
}

func defaults(config *helpers.Config) {
	if config.DefaultTimeout > 0 {
		DEFAULT_TIMEOUT = config.DefaultTimeout * time.Second
	}
}

type patsConfig struct {
	ServiceName              string `json:"service_name"`
	PlanName                 string `json:"plan_name"`
	BrokerUrl                string `json:"broker_url"`
	BrokerUser               string `json:"broker_user"`
	BrokerPassword           string `json:"broker_password"`
	CreateConfig             string `json:"create_config"`
	CreateBogusConfig        string `json:"create_bogus_config"`
	BindConfig               string `json:"bind_config"`
	BindBogusConfig          string `json:"bind_bogus_config"`
	IsolationSegment         string `json:"isolation_segment"`
	DisallowedLdapBindConfig string `json:"disallowed_ldap_bind_config"`
	MissingGIDLdapBindConfig string `json:"missing_gid_ldap_bind_config"`
}

func getPatsSpecificConfig() error {
	configFile, err := os.Open(helpers.ConfigPath())
	if err != nil {
		return err
	}
	defer configFile.Close()

	decoder := json.NewDecoder(configFile)

	config := &patsConfig{}
	err = decoder.Decode(config)
	if err != nil {
		return err
	}

	pConfig = *config
	return nil
}
