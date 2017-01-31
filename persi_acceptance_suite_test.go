package persi_acceptance_tests_test

import (
	"encoding/json"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"testing"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"

	"os/exec"
	"time"
	"path/filepath"
)

var (
	cfConfig         helpers.Config
	pConfig          patsConfig
	patsSuiteContext helpers.SuiteContext

	patsTestContext     helpers.SuiteContext
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

	SynchronizedBeforeSuite(func() []byte {
		patsSuiteContext = helpers.NewContext(cfConfig)
		if pConfig.PushedBrokerName != "" {
			patsAdminEnvironment = helpers.NewEnvironment(patsTestContext)
			patsAdminEnvironment.Setup()
		}

		cf.AsUser(patsSuiteContext.AdminUserContext(), DEFAULT_TIMEOUT, func() {
			// make sure we don't have a leftover service broker from another test
			deleteBroker(pConfig.BrokerUrl)

			if pConfig.PushedBrokerName != "" {
				// push the service broker as a cf application
				Expect(pConfig.SqlServiceName).ToNot(BeEmpty())

				appPath := os.Getenv("BROKER_APPLICATION_PATH")
				Expect(appPath).To(BeADirectory(), "BROKER_APPLICATION_PATH environment variable should point to a CF application")

				assetsPath := os.Getenv("ASSETS_PATH")
				Expect(assetsPath).To(BeADirectory(), "ASSETS_PATH environment variable should be a directory")

				Eventually(cf.Cf("update-security-group", "public_networks", filepath.Join(assetsPath, "security.json")), DEFAULT_TIMEOUT).Should(Exit(0))
				Eventually(cf.Cf("push", pConfig.PushedBrokerName, "-p", appPath, "-f", appPath+"/manifest.yml", "--no-start"), DEFAULT_TIMEOUT).Should(Exit(0))
				Eventually(cf.Cf("bind-service", pConfig.PushedBrokerName, pConfig.SqlServiceName), DEFAULT_TIMEOUT).Should(Exit(0))
				Eventually(cf.Cf("start", pConfig.PushedBrokerName), DEFAULT_TIMEOUT).Should(Exit(0))
			}

			createServiceBroker := cf.Cf("create-service-broker", brokerName, pConfig.BrokerUser, pConfig.BrokerPassword, pConfig.BrokerUrl).Wait(DEFAULT_TIMEOUT)
			Expect(createServiceBroker).To(Exit(0))
			Expect(createServiceBroker).To(Say(brokerName))
		})

		return nil
	}, func(_ []byte) {
		patsTestContext = helpers.NewContext(cfConfig)
		patsTestEnvironment = helpers.NewEnvironment(patsTestContext)

		patsTestEnvironment.Setup()
	})

	SynchronizedAfterSuite(func() {
		if patsTestEnvironment != nil {
			patsTestEnvironment.Teardown()
		}
	}, func() {
		cf.AsUser(patsSuiteContext.AdminUserContext(), DEFAULT_TIMEOUT, func() {
			cf.Cf("delete", "-f", pConfig.PushedBrokerName)

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
	ServiceName    string `json:"service_name"`
	PlanName       string `json:"plan_name"`
	BrokerUrl      string `json:"broker_url"`
	BrokerUser     string `json:"broker_user"`
	BrokerPassword string `json:"broker_password"`
	ServerAddress  string `json:"server_addr"`
	Share          string `json:"share"`
	BindConfig     string `json:"bind_config"`
	PushedBrokerName  string `json:"pushed_broker_name"`
	SqlServiceName string `json:"sql_service_name"`
}

func getPatsSpecificConfig() error {
	configFile, err := os.Open(helpers.ConfigPath())
	if err != nil {
		return err
	}
	defer configFile.Close()

	decoder := json.NewDecoder(configFile)

	config := &patsConfig{
		ServerAddress: "NotUsed",
		Share: "NotUsed",
	}
	err = decoder.Decode(config)
	if err != nil {
		return err
	}

	pConfig = *config
	return nil
}
