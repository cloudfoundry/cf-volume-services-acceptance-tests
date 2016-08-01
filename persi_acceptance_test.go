package persi_acceptance_tests_test

import (
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Cloud Foundry Persistence", func() {
	// given a target, org and space from suite
	It("should have a ginkgoPATS test org", func() {
		orgs := cf.Cf("orgs").Wait(DEFAULT_TIMEOUT)
		Expect(orgs).To(Exit(0))
		Expect(orgs).To(Say(".*ginkgoPATS"))
	})

	It("should have a ginkgoPATS test space", func() {
		orgs := cf.Cf("spaces").Wait(DEFAULT_TIMEOUT)
		Expect(orgs).To(Exit(0))
		Expect(orgs).To(Say(".*ginkgoPATS"))
	})

	It("should have a target", func() {
		orgs := cf.Cf("target").Wait(DEFAULT_TIMEOUT)
		Expect(orgs).To(Exit(0))
		Expect(orgs).To(Say("User:.*ginkgoPATS-USER"))
		Expect(orgs).To(Say("Org:.*ginkgoPATS-ORG"))
		Expect(orgs).To(Say("Space:.*ginkgoPATS-SPACE"))
	})

	Context("given a service broker", func() {
		BeforeEach(func() {
			cf.AsUser(patsContext.AdminUserContext(), DEFAULT_TIMEOUT, func() {
				createServiceBroker := cf.Cf("create-service-broker", BROKER_NAME, patsConfig.AdminUser, patsConfig.AdminPassword, BrokerURL).Wait(DEFAULT_TIMEOUT)
				Expect(createServiceBroker).To(Exit(0))
			})
		})

		AfterEach(func() {
			cf.AsUser(patsContext.AdminUserContext(), DEFAULT_TIMEOUT, func() {
				cf.Cf("purge-service-instance", "-f", INSTANCE_NAME).Wait(DEFAULT_TIMEOUT)
				cf.Cf("delete-service-broker", "-f", BROKER_NAME).Wait(DEFAULT_TIMEOUT)
			})
		})

		It("should have a volume service broker", func() {
			cf.AsUser(patsContext.AdminUserContext(), DEFAULT_TIMEOUT, func() {
				serviceBrokers := cf.Cf("service-brokers").Wait(DEFAULT_TIMEOUT)
				Expect(serviceBrokers).To(Exit(0))
				Expect(serviceBrokers).To(Say(BROKER_NAME))
			})
		})

		It("should not have enabled access", func() {
			cf.AsUser(patsContext.AdminUserContext(), DEFAULT_TIMEOUT, func() {
				serviceAccess := cf.Cf("service-access").Wait(DEFAULT_TIMEOUT)
				Expect(serviceAccess).To(Exit(0))
				Expect(serviceAccess).To(Say(BROKER_NAME))
				Expect(serviceAccess).To(Say(SERVICE_NAME + ".*" + PLAN_NAME + ".*none"))
			})
		})

		Context("given an enabled service", func() {
			BeforeEach(func() {
				cf.AsUser(patsContext.AdminUserContext(), DEFAULT_TIMEOUT, func() {
					publishService := cf.Cf("enable-service-access", SERVICE_NAME, "-o", patsContext.RegularUserContext().Org).Wait(DEFAULT_TIMEOUT)
					Expect(publishService).To(Exit(0))
				})
			})

			AfterEach(func() { /*disable service*/ })

			It("should have enabled access", func() {
				cf.AsUser(patsContext.AdminUserContext(), DEFAULT_TIMEOUT, func() {
					serviceAccess := cf.Cf("service-access").Wait(DEFAULT_TIMEOUT)
					Expect(serviceAccess).To(Exit(0))
					Expect(serviceAccess).To(Say(BROKER_NAME))
					Expect(serviceAccess).To(Say(SERVICE_NAME + ".*" + PLAN_NAME + ".*limited.*" + patsContext.RegularUserContext().Org))
				})
			})

			It("should be able to find a service in the marketplace", func() {
				cf.AsUser(patsContext.RegularUserContext(), DEFAULT_TIMEOUT, func() {
					marketplaceItems := cf.Cf("marketplace").Wait(DEFAULT_TIMEOUT)
					Expect(marketplaceItems).To(Exit(0))
					Expect(marketplaceItems).To(Say(SERVICE_NAME))
					Expect(marketplaceItems).To(Say(PLAN_NAME))
				})
			})

			Context("given a service instance", func() {
				BeforeEach(func() {
					cf.AsUser(patsContext.RegularUserContext(), DEFAULT_TIMEOUT, func() {
						createService := cf.Cf("create-service", SERVICE_NAME, PLAN_NAME, INSTANCE_NAME).Wait(DEFAULT_TIMEOUT)
						Expect(createService).To(Exit(0))
					})
				})

				AfterEach(func() {
					cf.AsUser(patsContext.RegularUserContext(), DEFAULT_TIMEOUT, func() {
						cf.Cf("delete-service", INSTANCE_NAME, "-f").Wait(DEFAULT_TIMEOUT)
					})
				})

				It("should have a service", func() {
					cf.AsUser(patsContext.RegularUserContext(), DEFAULT_TIMEOUT, func() {
						services := cf.Cf("services").Wait(DEFAULT_TIMEOUT)
						Expect(services).To(Exit(0))
						Expect(services).To(Say(INSTANCE_NAME))
					})
				})

				Context("given an installed cf app", func() {
					BeforeEach(func() {
						appPath := os.Getenv("TEST_APPLICATION_PATH")
						Expect(appPath).To(BeADirectory(), "TEST_APPLICATION_PATH environment variable should point to a CF application")
						cf.AsUser(patsContext.RegularUserContext(), DEFAULT_TIMEOUT, func() {
							Eventually(cf.Cf("push", APP_NAME, "-p", appPath, "-f", appPath+"/manifest.yml", "--no-start"), LONG_TIMEOUT).Should(Exit(0))
							Eventually(cf.Cf("curl", "/v2/apps/"+GetAppGuid(APP_NAME), "-X", "PUT", "-d", `{"diego": true}`), DEFAULT_TIMEOUT).Should(Exit(0))
						})
					})

					AfterEach(func() {
						cf.AsUser(patsContext.RegularUserContext(), DEFAULT_TIMEOUT, func() {
							cf.Cf("delete", APP_NAME, "-r", "-f").Wait(DEFAULT_TIMEOUT)
						})
					})

					It("it should be have the app", func() {
						cf.AsUser(patsContext.RegularUserContext(), DEFAULT_TIMEOUT, func() {
							marketplaceItems := cf.Cf("apps").Wait(DEFAULT_TIMEOUT)
							Expect(marketplaceItems).To(Exit(0))
							Expect(marketplaceItems).To(Say(APP_NAME))
						})
					})

					Context("when the app is bound", func() {
						BeforeEach(func() {
							cf.AsUser(patsContext.RegularUserContext(), DEFAULT_TIMEOUT, func() {
								bindResponse := cf.Cf("bind-service", APP_NAME, INSTANCE_NAME).Wait(DEFAULT_TIMEOUT)
								Expect(bindResponse).To(Exit(0))
							})
						})

						AfterEach(func() {
							cf.AsUser(patsContext.RegularUserContext(), DEFAULT_TIMEOUT, func() {
								cf.Cf("logs", APP_NAME, "--recent").Wait(DEFAULT_TIMEOUT)
								cf.Cf("stop", APP_NAME).Wait(DEFAULT_TIMEOUT)

								cf.Cf("unbind-service", APP_NAME, INSTANCE_NAME).Wait(DEFAULT_TIMEOUT)
							})
						})

						It("should show up as a bound app in a listing of services", func() {
							cf.AsUser(patsContext.RegularUserContext(), DEFAULT_TIMEOUT, func() {
								services := cf.Cf("services").Wait(DEFAULT_TIMEOUT)
								Expect(services).To(Exit(0))
								Expect(services).To(Say(INSTANCE_NAME + "[^\\n]+" + SERVICE_NAME + "[^\\n]+" + APP_NAME))
							})
						})

						Context("when the app is started", func() {
							BeforeEach(func() {
								cf.AsUser(patsContext.RegularUserContext(), DEFAULT_TIMEOUT, func() {
									bindResponse := cf.Cf("start", APP_NAME).Wait(LONG_TIMEOUT)
									Expect(bindResponse).To(Exit(0))
								})
							})

							AfterEach(func() {
								cf.AsUser(patsContext.RegularUserContext(), DEFAULT_TIMEOUT, func() {
									cf.Cf("logs", APP_NAME, "--recent").Wait(DEFAULT_TIMEOUT)
									cf.Cf("stop", APP_NAME).Wait(DEFAULT_TIMEOUT)
								})
							})

							It("should respond to http requests", func() {
								body, status, err := get(AppURL)
								Expect(err).NotTo(HaveOccurred())
								Expect(body).To(ContainSubstring("instance index:"))
								Expect(status).To(Equal(http.StatusOK))
							})

							It("should include the volume mount path in the application's environment", func() {
								cf.AsUser(patsContext.RegularUserContext(), DEFAULT_TIMEOUT, func() {
									env := cf.Cf("env", APP_NAME).Wait(DEFAULT_TIMEOUT)
									Expect(env).To(Exit(0))
									Expect(env).To(Say(SERVICE_NAME))
									Expect(env).To(Say(INSTANCE_NAME))
									Expect(env).To(Say("container_path"))
								})
							})

							It("should be able to write to the volume", func() {
								body, status, err := get(AppURL + "/write")
								Expect(err).NotTo(HaveOccurred())
								Expect(body).To(ContainSubstring("Hello Persistent World"))
								Expect(status).To(Equal(http.StatusOK))
							})
						})
					})
				})
			})
		})
	})
})

func GetAppGuid(appName string) string {
	cfApp := cf.Cf("app", appName, "--guid")
	Eventually(cfApp, DEFAULT_TIMEOUT).Should(Exit(0))

	appGuid := strings.TrimSpace(string(cfApp.Out.Contents()))
	Expect(appGuid).NotTo(Equal(""))
	return appGuid
}

func get(uri string) (body string, status int, err error) {
	req, err := http.NewRequest("GET", uri, nil)

	response, err := (&http.Client{}).Do(req)
	if err != nil {
		return "", status, err
	}
	defer response.Body.Close()

	bodyBytes, err := ioutil.ReadAll(response.Body)
	return string(bodyBytes[:]), response.StatusCode, err
}
