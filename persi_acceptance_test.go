package persi_acceptance_tests_test

import (
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"strconv"

	"bytes"
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
	"fmt"
)

var _ = Describe("Cloud Foundry Persistence", func() {
	var (
		appHost, appURL, appName, instanceName string
	)

	BeforeEach(func() {
		parallelNode := strconv.Itoa(GinkgoParallelNode())

		instanceName = "pats-volume-instance"
		appName = "pats-pora"

		instanceName = cfConfig.NamePrefix + "-" + instanceName + parallelNode
		appName = cfConfig.NamePrefix + "-" + appName + parallelNode

		appHost = appName + "." + cfConfig.AppsDomain
		appURL = "http://" + appHost
	})

	Context("given a service broker", func() {
		It("should have a volume service broker", func() {
			cf.AsUser(patsTestContext.AdminUserContext(), DEFAULT_TIMEOUT, func() {
				serviceBrokers := cf.Cf("service-brokers").Wait(DEFAULT_TIMEOUT)
				Expect(serviceBrokers).To(Exit(0))
				Expect(serviceBrokers).To(Say(brokerName))
			})
		})

		It("should not have enabled access", func() {
			cf.AsUser(patsTestContext.AdminUserContext(), DEFAULT_TIMEOUT, func() {
				serviceAccess := cf.Cf("service-access").Wait(DEFAULT_TIMEOUT)
				Expect(serviceAccess).To(Exit(0))
				Expect(serviceAccess).To(Say(brokerName))
				Expect(serviceAccess).To(Say(pConfig.ServiceName + ".*" + pConfig.PlanName + ".*"))
				Expect(serviceAccess).NotTo(Say(patsTestContext.RegularUserContext().Org))
			})
		})

		Context("given an enabled service", func() {
			BeforeEach(func() {
				cf.AsUser(patsTestContext.AdminUserContext(), DEFAULT_TIMEOUT, func() {
					publishService := cf.Cf("enable-service-access", pConfig.ServiceName, "-o", patsTestContext.RegularUserContext().Org).Wait(DEFAULT_TIMEOUT)
					Expect(publishService).To(Exit(0))
				})
			})

			AfterEach(func() {
				cf.AsUser(patsTestContext.AdminUserContext(), DEFAULT_TIMEOUT, func() {
					publishService := cf.Cf("disable-service-access", pConfig.ServiceName, "-o", patsTestContext.RegularUserContext().Org).Wait(DEFAULT_TIMEOUT)
					Expect(publishService).To(Exit(0))
				})
			})

			It("should have enabled access", func() {
				cf.AsUser(patsTestContext.AdminUserContext(), DEFAULT_TIMEOUT, func() {
					serviceAccess := cf.Cf("service-access").Wait(DEFAULT_TIMEOUT)
					Expect(serviceAccess).To(Exit(0))
					Expect(serviceAccess).To(Say(brokerName))
					Expect(serviceAccess).To(Say(pConfig.ServiceName + ".*" + pConfig.PlanName + ".*limited.*" + patsTestContext.RegularUserContext().Org))
				})
			})

			It("should be able to find a service in the marketplace", func() {
				cf.AsUser(patsTestContext.RegularUserContext(), DEFAULT_TIMEOUT, func() {
					marketplaceItems := cf.Cf("marketplace").Wait(DEFAULT_TIMEOUT)
					Expect(marketplaceItems).To(Exit(0))
					Expect(marketplaceItems).To(Say(pConfig.ServiceName))
					Expect(marketplaceItems).To(Say(pConfig.PlanName))
				})
			})

			Context("given a service instance", func() {
				BeforeEach(func() {
					cf.AsUser(patsTestContext.RegularUserContext(), DEFAULT_TIMEOUT, func() {
						var createService *Session
						if pConfig.ServerAddress == "NotUsed" {
							createService = cf.Cf("create-service", pConfig.ServiceName, pConfig.PlanName, instanceName).Wait(DEFAULT_TIMEOUT)
						} else {
							nfsParams := `{"share": "` + pConfig.ServerAddress + pConfig.Share + `"}`
							createService = cf.Cf("create-service", pConfig.ServiceName, pConfig.PlanName, instanceName, "-c", nfsParams).Wait(DEFAULT_TIMEOUT)
						}
						Expect(createService).To(Exit(0))
					})

					// wait for async service to finish
					Eventually(func() *Session {
						serviceDetails := cf.Cf("service", instanceName).Wait(DEFAULT_TIMEOUT)
						Expect(serviceDetails).To(Exit(0))
						return serviceDetails
					}, LONG_TIMEOUT, POLL_INTERVAL).Should(Say("create succeeded"))
				})

				AfterEach(func() {
					cf.AsUser(patsTestContext.RegularUserContext(), DEFAULT_TIMEOUT, func() {
						cf.Cf("delete-service", instanceName, "-f").Wait(DEFAULT_TIMEOUT)
					})

					// wait for async service to finish
					Eventually(func() *Session {
						serviceDetails := cf.Cf("services").Wait(DEFAULT_TIMEOUT)
						Expect(serviceDetails).To(Exit(0))
						return serviceDetails
					}, LONG_TIMEOUT, POLL_INTERVAL).Should(Not(Say(instanceName)))

					cf.AsUser(patsTestContext.AdminUserContext(), DEFAULT_TIMEOUT, func() {
						cf.Cf("purge-service-instance", instanceName, "-f").Wait(DEFAULT_TIMEOUT)
					})
				})

				It("should have a service", func() {
					services := cf.Cf("services").Wait(DEFAULT_TIMEOUT)
					Expect(services).To(Say(instanceName))
				})

				Context("given an installed cf app", func() {
					var appPath string
					BeforeEach(func() {
						appPath = os.Getenv("TEST_APPLICATION_PATH")
						Expect(appPath).To(BeADirectory(), "TEST_APPLICATION_PATH environment variable should point to a CF application")
						cf.AsUser(patsTestContext.RegularUserContext(), DEFAULT_TIMEOUT, func() {
							Eventually(cf.Cf("push", appName, "-p", appPath, "-f", appPath+"/manifest.yml", "--no-start"), DEFAULT_TIMEOUT).Should(Exit(0))
							Eventually(cf.Cf("curl", "/v2/apps/"+GetAppGuid(appName), "-X", "PUT", "-d", `{"diego": true}`), DEFAULT_TIMEOUT).Should(Exit(0))
						})
					})

					AfterEach(func() {
						cf.AsUser(patsTestContext.RegularUserContext(), DEFAULT_TIMEOUT, func() {
							cf.Cf("delete", appName, "-r", "-f").Wait(DEFAULT_TIMEOUT)
						})
					})

					It("it should be have the app", func() {
						cf.AsUser(patsTestContext.RegularUserContext(), DEFAULT_TIMEOUT, func() {
							marketplaceItems := cf.Cf("apps").Wait(DEFAULT_TIMEOUT)
							Expect(marketplaceItems).To(Exit(0))
							Expect(marketplaceItems).To(Say(appName))
						})
					})

					Context("when the app is bound", func() {
						BeforeEach(func() {
							cf.AsUser(patsTestContext.RegularUserContext(), DEFAULT_TIMEOUT, func() {
								if pConfig.BindConfig == "" {
									bindResponse := cf.Cf("bind-service", appName, instanceName).Wait(DEFAULT_TIMEOUT)
									Expect(bindResponse).To(Exit(0))
								} else {
									bindResponse := cf.Cf("bind-service", appName, instanceName, "-c", pConfig.BindConfig).Wait(DEFAULT_TIMEOUT)
									Expect(bindResponse).To(Exit(0))
								}
							})
						})

						AfterEach(func() {
							cf.AsUser(patsTestContext.RegularUserContext(), DEFAULT_TIMEOUT, func() {
								cf.Cf("logs", appName, "--recent").Wait(DEFAULT_TIMEOUT)
								cf.Cf("stop", appName).Wait(DEFAULT_TIMEOUT)

								cf.Cf("unbind-service", appName, instanceName).Wait(DEFAULT_TIMEOUT)
							})
						})

						It("should show up as a bound app in a listing of services", func() {
							cf.AsUser(patsTestContext.RegularUserContext(), DEFAULT_TIMEOUT, func() {
								services := cf.Cf("services").Wait(DEFAULT_TIMEOUT)
								Expect(services).To(Exit(0))
								Expect(services).To(Say(instanceName + "[^\\n]+" + pConfig.ServiceName + "[^\\n]+" + appName))
							})
						})

						Context("when the app is started", func() {
							BeforeEach(func() {
								cf.AsUser(patsTestContext.RegularUserContext(), DEFAULT_TIMEOUT, func() {
									bindResponse := cf.Cf("start", appName).Wait(LONG_TIMEOUT)
									Expect(bindResponse).To(Exit(0))
								})
							})

							AfterEach(func() {
								cf.AsUser(patsTestContext.RegularUserContext(), DEFAULT_TIMEOUT, func() {
									cf.Cf("logs", appName, "--recent").Wait(DEFAULT_TIMEOUT)
									cf.Cf("stop", appName).Wait(DEFAULT_TIMEOUT)
								})
							})

							It("should respond to http requests", func() {
								body, status, err := get(appURL)
								Expect(err).NotTo(HaveOccurred())
								Expect(body).To(ContainSubstring("instance index:"))
								Expect(status).To(Equal(http.StatusOK))
							})

							It("should include the volume mount path in the application's environment", func() {
								cf.AsUser(patsTestContext.RegularUserContext(), DEFAULT_TIMEOUT, func() {
									env := cf.Cf("env", appName).Wait(DEFAULT_TIMEOUT)
									Expect(env).To(Exit(0))
									Expect(env).To(Say(pConfig.ServiceName))
									Expect(env).To(Say(instanceName))
									Expect(env).To(Or(Say("container_path"), Say("container_dir")))
								})
							})

							It("should be able to write to the volume", func() {
								body, status, err := get(appURL + "/write")
								Expect(err).NotTo(HaveOccurred())
								Expect(body).To(ContainSubstring("Hello Persistent World"))
								Expect(status).To(Equal(http.StatusOK))
							})

							if os.Getenv("TEST_MULTI_CELL") == "true" {
								Context("when the app is scaled across cells", func() {
									const appScale = 5
									BeforeEach(func() {
										cf.AsUser(patsTestContext.RegularUserContext(), DEFAULT_TIMEOUT, func() {
											bindResponse := cf.Cf("scale", appName, "-i", strconv.Itoa(appScale)).Wait(LONG_TIMEOUT)
											Expect(bindResponse).To(Exit(0))

											// wait for app to scale
											Eventually(func() int {
												apps := cf.Cf("app", appName).Wait(DEFAULT_TIMEOUT)
												Expect(apps).To(Exit(0))
												return bytes.Count(apps.Out.Contents(), []byte("running"))
											}, LONG_TIMEOUT, POLL_INTERVAL).Should(Equal(appScale))
										})
									})

									It("should be able to create a test file then read it from any instance", func() {
										fname, status, err := get(appURL + "/create")
										Expect(err).NotTo(HaveOccurred())
										Expect(fname).To(ContainSubstring("pora"))
										Expect(status).To(Equal(http.StatusOK))

										responses := map[string]int{}
										for i := 0; i < appScale*10000; i++ {
											body, status, err := get(appURL + "/read/" + fname)
											Expect(err).NotTo(HaveOccurred())
											Expect(body).To(ContainSubstring("Hello Persistent World"))
											Expect(status).To(Equal(http.StatusOK))
											responses[body] = 1
											if len(responses) >= appScale {
												break
											}
										}
										body, status, err := get(appURL + "/delete/" + fname)
										Expect(err).NotTo(HaveOccurred())
										Expect(body).To(ContainSubstring(fname))
										Expect(status).To(Equal(http.StatusOK))

										Expect(len(responses)).To(Equal(appScale))
									})
								})
							}
							if os.Getenv("TEST_MOUNT_OPTIONS") == "true" {
								Context("when a second app is bound with a different uid and gid", func() {
									var (
										app2Name string
									)
									BeforeEach(func() {
										app2Name = appName + "-2"
										cf.AsUser(patsTestContext.RegularUserContext(), DEFAULT_TIMEOUT, func() {
											Eventually(cf.Cf("push", app2Name, "-p", appPath, "-f", appPath+"/manifest.yml", "--no-start"), DEFAULT_TIMEOUT).Should(Exit(0))
											Eventually(cf.Cf("curl", "/v2/apps/"+GetAppGuid(app2Name), "-X", "PUT", "-d", `{"diego": true}`), DEFAULT_TIMEOUT).Should(Exit(0))

											bindConfig := `{"uid":"5000","gid":"5000"}`
											bindResponse := cf.Cf("bind-service", app2Name, instanceName, "-c", bindConfig).Wait(DEFAULT_TIMEOUT)
											Expect(bindResponse).To(Exit(0))

											startResponse := cf.Cf("start", app2Name).Wait(LONG_TIMEOUT)
											Expect(startResponse).To(Exit(0))
										})
									})
									AfterEach(func() {
										cf.AsUser(patsTestContext.RegularUserContext(), DEFAULT_TIMEOUT, func() {
											cf.Cf("unbind-service", app2Name, instanceName).Wait(DEFAULT_TIMEOUT)

											cf.Cf("delete", app2Name, "-r", "-f").Wait(DEFAULT_TIMEOUT)
										})
									})

									Context("when the first app create a file", func() {
										var (
											fname   string
											app2URL string
										)
										BeforeEach(func() {
											app2URL = "http://" + app2Name + "." + cfConfig.AppsDomain

											fname, status, err := get(appURL + "/create")
											Expect(err).NotTo(HaveOccurred())
											Expect(fname).To(ContainSubstring("pora"))
											Expect(status).To(Equal(http.StatusOK))
										})
										AfterEach(func() {
											body, status, err := get(fmt.Sprintf("%s/delete/%s", appURL, fname))
											Expect(err).NotTo(HaveOccurred())
											Expect(body).To(ContainSubstring(fname))
											Expect(status).To(Equal(http.StatusOK))
										})

										It("should be readable by the second app", func() {
											body, status, err := get(fmt.Sprintf("%s/read/%s", app2URL, fname))
											Expect(err).NotTo(HaveOccurred())
											Expect(body).To(ContainSubstring("Hello Persistent World"))
											Expect(status).To(Equal(http.StatusOK))
										})

										It("should not be deletable by the second app", func() {
											body, status, _ := get(fmt.Sprintf("%s/delete/%s", app2URL, fname))
											Expect(body).NotTo(ContainSubstring("deleted"))
											Expect(status).NotTo(Equal(http.StatusOK))
										})
									})
								})
							}
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
