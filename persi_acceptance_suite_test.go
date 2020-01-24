package persi_acceptance_tests_test

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/zbiljic/go-filelock"

	. "github.com/onsi/ginkgo"
	ginkgoconfig "github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/config"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
)

var (
	cfConfig       = loadConfigAndDefaultValues()
	pConfig        = getPatsSpecificConfig()
	patsSuiteSetup *workflowhelpers.ReproducibleTestSuiteSetup
	patsTestSetup  *workflowhelpers.ReproducibleTestSuiteSetup

	DEFAULT_TIMEOUT = 30 * time.Second
	LONG_TIMEOUT    = 600 * time.Second
	POLL_INTERVAL   = 3 * time.Second

	brokerName string
)

func TestPersiAcceptance(t *testing.T) {
	RegisterFailHandler(Fail)

	serviceBroker := &broker{
		Name:     pConfig.ServiceName + "-broker",
		User:     pConfig.BrokerUser,
		Password: pConfig.BrokerPassword,
		URL:      pConfig.BrokerUrl,
	}

	if serviceBroker.managedByPATs() {
		brokerName = serviceBroker.Name
	}

	componentName := "PATS Suite"
	rs := []Reporter{}
	maxParallelSetup := 5

	SynchronizedBeforeSuite(func() []byte {
		patsSuiteSetup = workflowhelpers.NewTestSuiteSetup(cfConfig)

		workflowhelpers.AsUser(patsSuiteSetup.AdminUserContext(), DEFAULT_TIMEOUT, func() {
			// make sure we don't have a leftover service broker from another test
			serviceBroker.Delete()
		})

		serviceBroker.Create()

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
		fl, err := filelock.New(lockFilePath + strconv.Itoa(ginkgoconfig.GinkgoConfig.ParallelNode%maxParallelSetup))
		Expect(err).ToNot(HaveOccurred())
		fl.Must().Lock()
		defer fl.Must().Unlock()

		patsTestSetup = workflowhelpers.NewTestSuiteSetup(cfConfig)

		patsTestSetup.Setup()
		if pConfig.IsolationSegment != "" {
			workflowhelpers.AsUser(patsTestSetup.AdminUserContext(), DEFAULT_TIMEOUT, func() {
				Eventually(cf.Cf("create-isolation-segment", pConfig.IsolationSegment), DEFAULT_TIMEOUT).Should(Exit(0))
				Eventually(cf.Cf("enable-org-isolation", patsTestSetup.RegularUserContext().Org, pConfig.IsolationSegment), DEFAULT_TIMEOUT).Should(Exit(0))
				Eventually(cf.Cf("set-org-default-isolation-segment", patsTestSetup.RegularUserContext().Org, pConfig.IsolationSegment), DEFAULT_TIMEOUT).Should(Exit(0))
			})
		}
	})

	SynchronizedAfterSuite(func() {
		if patsTestSetup != nil {
			patsTestSetup.Teardown()
		}
	}, func() {
		workflowhelpers.AsUser(patsSuiteSetup.AdminUserContext(), DEFAULT_TIMEOUT, func() {
			err := serviceBroker.Delete()
			if err != nil {
				cf.Cf("purge-service-offering", pConfig.ServiceName).Wait(DEFAULT_TIMEOUT)
				Fail("pats service broker could not be cleaned up.")
			}
		})
	})

	if cfConfig.ArtifactsDirectory != "" {
		helpers.EnableCFTrace(cfConfig, componentName)
		rs = append(rs, helpers.NewJUnitReporter(cfConfig, componentName))
	}

	RunSpecsWithDefaultAndCustomReporters(t, componentName, rs)
}

func defaults(config *config.Config) {
	if config.DefaultTimeout > 0 {
		DEFAULT_TIMEOUT = time.Duration(config.DefaultTimeout) * time.Second
	}
}

type patsConfig struct {
	ServiceName                        string   `json:"service_name"`
	PlanName                           string   `json:"plan_name"`
	BrokerUrl                          string   `json:"broker_url"`
	BrokerUser                         string   `json:"broker_user"`
	BrokerPassword                     string   `json:"broker_password"`
	CreateConfig                       string   `json:"create_config"`
	CreateBogusConfig                  string   `json:"create_bogus_config"`
	CreateLazyUnmountConfig            string   `json:"create_lazy_unmount_config"`
	LazyUnmountVmInstance              string   `json:"lazy_unmount_vm_instance"`
	LazyUnmountRemoteServerJobName     string   `json:"lazy_unmount_remote_server_job_name"`
	LazyUnmountRemoteServerProcessName string   `json:"lazy_unmount_remote_server_process_name"`
	BindConfig                         []string `json:"bind_config"`
	BindBogusConfig                    string   `json:"bind_bogus_config"`
	BindLazyUnmountConfig              string   `json:"bind_lazy_unmount_config"`
	IsolationSegment                   string   `json:"isolation_segment"`
	DisallowedLdapBindConfig           string   `json:"disallowed_ldap_bind_config"`
	DisallowedOverrideBindConfig       string   `json:"disallowed_override_bind_config"`
}

func getPatsSpecificConfig() patsConfig {
	configFile, err := os.Open(config.ConfigPath())
	if err != nil {
		panic(err)
	}

	defer configFile.Close()

	decoder := json.NewDecoder(configFile)

	config := &patsConfig{}
	err = decoder.Decode(config)
	if err != nil {
		panic(err)
	}

	return *config
}

func loadConfigAndDefaultValues() *config.Config {
	config := config.LoadConfig()
	defaults(config)
	return config
}

type broker struct {
	Name     string
	User     string
	Password string
	URL      string
}

func (b *broker) Create() {
	if !b.managedByPATs() {
		// service broker was created outside of pats, e.g. in the errand
		return
	}

	workflowhelpers.AsUser(patsSuiteSetup.AdminUserContext(), DEFAULT_TIMEOUT, func() {
		createServiceBroker := cf.Cf("create-service-broker", brokerName, pConfig.BrokerUser, pConfig.BrokerPassword, pConfig.BrokerUrl).Wait(DEFAULT_TIMEOUT)
		Expect(createServiceBroker).To(Exit(0))
		Expect(createServiceBroker).To(Say(brokerName))
	})
}

func (b *broker) Delete() error {
	if !b.managedByPATs() {
		// service broker was created outside of pats, e.g. in the errand
		return nil
	}

	if b.Name != "" {
		session := cf.Cf("delete-service-broker", "-f", b.Name).Wait(DEFAULT_TIMEOUT)
		if session.ExitCode() != 0 {
			return errors.New("failed-to-delete-service-broker")
		}
		return nil
	}

	if b.URL != "" {
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
			if broker.Entity.BrokerUrl == b.URL {
				session := cf.Cf("delete-service-broker", "-f", broker.Entity.Name).Wait(DEFAULT_TIMEOUT)
				if session.ExitCode() != 0 {
					return errors.New("failed-to-delete-service-broker")
				}
				return nil
			}
		}

		return nil
	}

	return nil
}

func (b *broker) managedByPATs() bool {
	return b.User != "" && b.Password != ""
}
