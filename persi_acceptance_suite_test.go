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

	"time"
)

var (
	cfConfig         helpers.Config
	pConfig          patsConfig
	patsSuiteContext helpers.SuiteContext

	patsTestContext     helpers.SuiteContext
	patsTestEnvironment *helpers.Environment

	DEFAULT_TIMEOUT = 30 * time.Second
	LONG_TIMEOUT    = 300 * time.Second

	brokerName = "pats-broker"
)

func TestPersiAcceptance(t *testing.T) {
	RegisterFailHandler(Fail)

	cfConfig = helpers.LoadConfig()
	defaults(&cfConfig)

	err := getPatsSpecificConfig()
	if err != nil {
		panic(err)
	}

	componentName := "PATS Suite"
	rs := []Reporter{}

	SynchronizedBeforeSuite(func() []byte {
		patsSuiteContext = helpers.NewContext(cfConfig)

		cf.AsUser(patsSuiteContext.AdminUserContext(), DEFAULT_TIMEOUT, func() {
			// make sure we don't have a leftover service broker from another test
			cf.Cf("delete-service-broker", "-f", brokerName).Wait(DEFAULT_TIMEOUT)
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
		patsTestEnvironment.Teardown()
	}, func() {
		cf.AsUser(patsSuiteContext.AdminUserContext(), DEFAULT_TIMEOUT, func() {
			cf.Cf("delete-service-broker", "-f", brokerName).Wait(DEFAULT_TIMEOUT)
		})
	})

	if cfConfig.ArtifactsDirectory != "" {
		helpers.EnableCFTrace(cfConfig, componentName)
		rs = append(rs, helpers.NewJUnitReporter(cfConfig, componentName))
	}

	RunSpecsWithDefaultAndCustomReporters(t, componentName, rs)
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
