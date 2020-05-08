package persi_acceptance_tests_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

const printErrorsOn = true
const printErrorsOff = false

var _ = Describe("Cloud Foundry Persistence", func() {
	var (
		appHost, readWriteAppURL, appName, instanceName string
		bogusAppName, bogusInstanceName                 string
	)

	BeforeEach(func() {
		parallelNode := strconv.Itoa(GinkgoParallelNode())

		instanceName = "pats-volume-instance"
		appName = "pats-pora"

		instanceName = cfConfig.NamePrefix + "-" + instanceName + parallelNode
		bogusInstanceName = cfConfig.NamePrefix + "-bogus-" + instanceName + parallelNode
		appName = cfConfig.NamePrefix + "-" + appName + parallelNode
		bogusAppName = cfConfig.NamePrefix + "-bogus-" + appName + parallelNode

		appHost = appName + "." + cfConfig.AppsDomain
		readWriteAppURL = "http://" + appHost
	})

	Context("given a service broker", func() {
		It("should have a volume service broker", func() {
			workflowhelpers.AsUser(patsTestSetup.AdminUserContext(), DEFAULT_TIMEOUT, func() {
				serviceBrokers := cf.Cf("service-brokers").Wait(DEFAULT_TIMEOUT)
				Expect(serviceBrokers).To(Exit(0))
				Expect(serviceBrokers).To(Say(brokerName))
			})
		})

		It("should not have enabled access", func() {
			workflowhelpers.AsUser(patsTestSetup.AdminUserContext(), DEFAULT_TIMEOUT, func() {
				serviceAccess := cf.Cf("service-access").Wait(DEFAULT_TIMEOUT)
				Expect(serviceAccess).To(Exit(0))
				Expect(serviceAccess).To(Say(brokerName))
				Expect(serviceAccess).To(Say(pConfig.ServiceName + ".*" + pConfig.PlanName + ".*"))
				Expect(serviceAccess).NotTo(Say(patsTestSetup.RegularUserContext().Org))
			})
		})

		Context("given an enabled service", func() {
			BeforeEach(func() {
				workflowhelpers.AsUser(patsTestSetup.AdminUserContext(), DEFAULT_TIMEOUT, func() {
					publishService := cf.Cf("enable-service-access", pConfig.ServiceName, "-o", patsTestSetup.RegularUserContext().Org).Wait(DEFAULT_TIMEOUT)
					Expect(publishService).To(Exit(0))
				})
			})

			AfterEach(func() {
				workflowhelpers.AsUser(patsTestSetup.AdminUserContext(), DEFAULT_TIMEOUT, func() {
					publishService := cf.Cf("disable-service-access", pConfig.ServiceName, "-o", patsTestSetup.RegularUserContext().Org).Wait(DEFAULT_TIMEOUT)
					Expect(publishService).To(Exit(0))
				})
			})

			It("should have enabled access", func() {
				workflowhelpers.AsUser(patsTestSetup.AdminUserContext(), DEFAULT_TIMEOUT, func() {
					serviceAccess := cf.Cf("service-access").Wait(DEFAULT_TIMEOUT)
					Expect(serviceAccess).To(Exit(0))
					Expect(serviceAccess).To(Say(brokerName))
					Expect(serviceAccess).To(Say(pConfig.ServiceName + ".*" + pConfig.PlanName + ".*limited.*" + patsTestSetup.RegularUserContext().Org))
				})
			})

			It("should be able to find a service in the marketplace", func() {
				workflowhelpers.AsUser(patsTestSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
					marketplaceItems := cf.Cf("marketplace").Wait(DEFAULT_TIMEOUT)
					Expect(marketplaceItems).To(Exit(0))
					Expect(marketplaceItems).To(Say(pConfig.ServiceName))
					Expect(marketplaceItems).To(Say(pConfig.PlanName))
				})
			})

			Context("given a service instance", func() {
				BeforeEach(func() {
					workflowhelpers.AsUser(patsTestSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
						var (
							createService, createBogusService *Session
						)

						if pConfig.CreateConfig == "" {
							createService = cf.Cf("create-service", pConfig.ServiceName, pConfig.PlanName, instanceName).Wait(DEFAULT_TIMEOUT)
						} else {
							createService = cf.Cf("create-service", pConfig.ServiceName, pConfig.PlanName, instanceName, "-c", pConfig.CreateConfig).Wait(DEFAULT_TIMEOUT)
						}
						Expect(createService).To(Exit(0))

						if os.Getenv("TEST_MOUNT_FAIL_LOGGING") == "true" {
							if pConfig.CreateBogusConfig == "" {
								createBogusService = cf.Cf("create-service", pConfig.ServiceName, pConfig.PlanName, bogusInstanceName).Wait(DEFAULT_TIMEOUT)
							} else {
								createBogusService = cf.Cf("create-service", pConfig.ServiceName, pConfig.PlanName, bogusInstanceName, "-c", pConfig.CreateBogusConfig).Wait(DEFAULT_TIMEOUT)
							}
							Expect(createBogusService).To(Exit(0))
						}
					})

					Eventually(func() *Session {
						serviceDetails := cf.Cf("service", instanceName).Wait(DEFAULT_TIMEOUT)
						Expect(serviceDetails).To(Exit(0))
						return serviceDetails
					}, LONG_TIMEOUT, POLL_INTERVAL).Should(Say("create succeeded"))

					if os.Getenv("TEST_MOUNT_FAIL_LOGGING") == "true" {
						Eventually(func() *Session {
							serviceDetails := cf.Cf("service", bogusInstanceName).Wait(DEFAULT_TIMEOUT)
							Expect(serviceDetails).To(Exit(0))
							return serviceDetails
						}, LONG_TIMEOUT, POLL_INTERVAL).Should(Say("create succeeded"))
					}
				})

				AfterEach(func() {
					workflowhelpers.AsUser(patsTestSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
						cf.Cf("delete-service", instanceName, "-f").Wait(DEFAULT_TIMEOUT)

						if os.Getenv("TEST_MOUNT_FAIL_LOGGING") == "true" {
							cf.Cf("delete-service", bogusInstanceName, "-f").Wait(DEFAULT_TIMEOUT)
						}
					})

					Eventually(func() *Session {
						serviceDetails := cf.Cf("services").Wait(DEFAULT_TIMEOUT)
						Expect(serviceDetails).To(Exit(0))
						return serviceDetails
					}, LONG_TIMEOUT, POLL_INTERVAL).Should(Not(Say(instanceName)))

					workflowhelpers.AsUser(patsTestSetup.AdminUserContext(), DEFAULT_TIMEOUT, func() {
						cf.Cf("purge-service-instance", instanceName, "-f").Wait(DEFAULT_TIMEOUT)
					})

					if os.Getenv("TEST_MOUNT_FAIL_LOGGING") == "true" {
						Eventually(func() *Session {
							serviceDetails := cf.Cf("services").Wait(DEFAULT_TIMEOUT)
							Expect(serviceDetails).To(Exit(0))
							return serviceDetails
						}, LONG_TIMEOUT, POLL_INTERVAL).Should(Not(Say(bogusInstanceName)))

						workflowhelpers.AsUser(patsTestSetup.AdminUserContext(), DEFAULT_TIMEOUT, func() {
							cf.Cf("purge-service-instance", bogusInstanceName, "-f").Wait(DEFAULT_TIMEOUT)
						})
					}
				})

				It("should have a service", func() {
					services := cf.Cf("services").Wait(DEFAULT_TIMEOUT)
					Expect(services).To(Say(instanceName))
				})

				if os.Getenv("TEST_MOUNT_FAIL_LOGGING") == "true" {
					Context("given an installed cf app bound to bogus service", func() {
						var appPath string
						BeforeEach(func() {
							workflowhelpers.AsUser(patsTestSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
								appPath = os.Getenv("TEST_APPLICATION_PATH")
								Expect(appPath).To(BeADirectory(), "TEST_APPLICATION_PATH environment variable should point to a CF application")
								Eventually(cf.Cf("push", bogusAppName, "-p", appPath, "-f", appPath+"/manifest.yml", "--no-start"), DEFAULT_TIMEOUT).Should(Exit(0))
								if pConfig.BindBogusConfig == "" {
									bindResponse := cf.Cf("bind-service", bogusAppName, bogusInstanceName).Wait(DEFAULT_TIMEOUT)
									Expect(bindResponse).To(Exit(0))
								} else {
									bindResponse := cf.Cf("bind-service", bogusAppName, bogusInstanceName, "-c", pConfig.BindBogusConfig).Wait(DEFAULT_TIMEOUT)
									Expect(bindResponse).To(Exit(0))
								}
								startResponse := cf.Cf("start", bogusAppName).Wait(LONG_TIMEOUT)
								Expect(startResponse).To(Exit(1))
							})
						})

						AfterEach(func() {
							workflowhelpers.AsUser(patsTestSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
								cf.Cf("unbind-service", bogusAppName, bogusInstanceName).Wait(DEFAULT_TIMEOUT)
								cf.Cf("delete", bogusAppName, "-r", "-f").Wait(DEFAULT_TIMEOUT)
							})
						})

						It("should see errors in cf logs", func() {
							workflowhelpers.AsUser(patsTestSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
								Eventually(cf.Cf("logs", bogusAppName, "--recent").Wait(DEFAULT_TIMEOUT)).Should(Say("failed to mount volume, errors:"))
							})
						})
					})
				}

				Context("given an exemplar bind config", func() {

					var bindConfigToUse string
					BeforeEach(func() {
						bindConfigToUse = pConfig.BindConfig[0]
					})

					Context("given an installed cf app", func() {
						var appPath string

						BeforeEach(func() {
							workflowhelpers.AsUser(patsTestSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
								if os.Getenv("TEST_DOCKER_PORA") == "true" {
									Eventually(cf.Cf("push", appName, "--docker-image", "cfpersi/pora", "--no-start"), DEFAULT_TIMEOUT).Should(Exit(0))
								} else {
									appPath = os.Getenv("TEST_APPLICATION_PATH")
									Expect(appPath).To(BeADirectory(), "TEST_APPLICATION_PATH environment variable should point to a CF application")
									Eventually(cf.Cf("push", appName, "-p", appPath, "-f", appPath+"/manifest.yml", "--no-start"), DEFAULT_TIMEOUT).Should(Exit(0))
								}

								if bindConfigToUse == "" {
									bindResponse := cf.Cf("bind-service", appName, instanceName).Wait(DEFAULT_TIMEOUT)
									Expect(bindResponse).To(Exit(0))
								} else {
									bindResponse := cf.Cf("bind-service", appName, instanceName, "-c", bindConfigToUse).Wait(DEFAULT_TIMEOUT)
									Expect(bindResponse).To(Exit(0))
								}
							})
						})

						AfterEach(func() {
							workflowhelpers.AsUser(patsTestSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
								cf.Cf("logs", appName, "--recent").Wait(DEFAULT_TIMEOUT)
								cf.Cf("stop", appName).Wait(DEFAULT_TIMEOUT)

								cf.Cf("unbind-service", appName, instanceName).Wait(DEFAULT_TIMEOUT)

								cf.Cf("delete", appName, "-r", "-f").Wait(DEFAULT_TIMEOUT)
							})
						})

						It("it should have the app", func() {
							workflowhelpers.AsUser(patsTestSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
								marketplaceItems := cf.Cf("apps").Wait(DEFAULT_TIMEOUT)
								Expect(marketplaceItems).To(Exit(0))
								Expect(marketplaceItems).To(Say(appName))
							})
						})

						It("should show up as a bound app in a listing of services", func() {
							workflowhelpers.AsUser(patsTestSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
								services := cf.Cf("services").Wait(DEFAULT_TIMEOUT)
								Expect(services).To(Exit(0))
								Expect(services).To(Say(instanceName + "[^\\n]+" + pConfig.ServiceName + "[^\\n]+" + appName))
							})
						})

						Context("when the app is started", func() {
							BeforeEach(func() {
								workflowhelpers.AsUser(patsTestSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
									bindResponse := cf.Cf("start", appName).Wait(LONG_TIMEOUT)
									Expect(bindResponse).To(Exit(0))
								})
							})

							AfterEach(func() {
								workflowhelpers.AsUser(patsTestSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
									cf.Cf("logs", appName, "--recent").Wait(DEFAULT_TIMEOUT)
									cf.Cf("stop", appName).Wait(DEFAULT_TIMEOUT)
								})
							})

							It("should verify that the app mounted the volume", func() {
								By("verifying that it responds to http requests")
								eventuallyExpect(readWriteAppURL, "instance index:")

								By("verifying that the volume mount path is included in the application's environment")
								workflowhelpers.AsUser(patsTestSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
									env := cf.Cf("env", appName).Wait(DEFAULT_TIMEOUT)
									Expect(env).To(Exit(0))
									Expect(env).To(Say(pConfig.ServiceName))
									Expect(env).To(Say(instanceName))
									Expect(env).To(Or(Say("container_path"), Say("container_dir")))
								})

								By("verifying that the app is able to write to the volume")
								eventuallyExpect(readWriteAppURL+"/write", "Hello Persistent World")
							})

							if os.Getenv("TEST_MULTI_CELL") == "true" {
								It("should keep the data across multiple stops and starts", func() {
									fname := eventuallyExpect(readWriteAppURL+"/create", "pora")
									// start a bunch of simultaneous requests to do file io
									var wg sync.WaitGroup
									var done bool
									wg.Add(10)
									for i := 0; i < 10; i++ {
										go func() {
											for !done {
												get(readWriteAppURL+"/loadtest", printErrorsOff)
											}
											wg.Done()
										}()
									}

									workflowhelpers.AsUser(patsTestSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
										for i := 0; i < 20; i++ {
											stopResponse := cf.Cf("stop", appName).Wait(DEFAULT_TIMEOUT)
											Expect(stopResponse).To(Exit(0))
											startResponse := cf.Cf("start", appName).Wait(LONG_TIMEOUT)
											Expect(startResponse).To(Exit(0))
										}
									})

									// signal our background load to stop and then wait for it
									done = true
									wg.Wait()

									eventuallyExpect(readWriteAppURL+"/read/"+fname, "Hello Persistent World")
									eventuallyExpect(readWriteAppURL+"/delete/"+fname, fname)

									// clean up any load test files that got left behind on the mount due to apps stopping
									// and starting
									get(readWriteAppURL+"/loadtestcleanup", printErrorsOn)
								})

								Context("when the app is scaled across cells", func() {
									const appScale = 5
									BeforeEach(func() {
										workflowhelpers.AsUser(patsTestSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
											bindResponse := cf.Cf("scale", appName, "-i", strconv.Itoa(appScale)).Wait(LONG_TIMEOUT)
											Expect(bindResponse).To(Exit(0))

											Eventually(func() int {
												apps := cf.Cf("app", appName).Wait(DEFAULT_TIMEOUT)
												Expect(apps).To(Exit(0))
												return bytes.Count(apps.Out.Contents(), []byte("running"))
											}, LONG_TIMEOUT, POLL_INTERVAL).Should(Equal(appScale))
										})
									})

									It("should be able to create a test file then read it from any instance", func() {
										fname := eventuallyExpect(readWriteAppURL+"/create", "pora")

										responses := map[string]int{}
										for i := 0; i < appScale*10000; i++ {
											body := eventuallyExpect(readWriteAppURL+"/read/"+fname, "Hello Persistent World")
											responses[body] = 1
											if len(responses) >= appScale {
												break
											}
										}
										eventuallyExpect(readWriteAppURL+"/delete/"+fname, fname)

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
										workflowhelpers.AsUser(patsTestSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
											if os.Getenv("TEST_DOCKER_PORA") == "true" {
												Eventually(cf.Cf("push", app2Name, "--docker-image", "cfpersi/pora", "--no-start"), DEFAULT_TIMEOUT).Should(Exit(0))
											} else {
												appPath = os.Getenv("TEST_APPLICATION_PATH")
												Expect(appPath).To(BeADirectory(), "TEST_APPLICATION_PATH environment variable should point to a CF application")
												Eventually(cf.Cf("push", app2Name, "-p", appPath, "-f", appPath+"/manifest.yml", "--no-start"), DEFAULT_TIMEOUT).Should(Exit(0))
											}

											bindConfig := `{"uid":"5000","gid":"5000"}`
											bindResponse := cf.Cf("bind-service", app2Name, instanceName, "-c", bindConfig).Wait(DEFAULT_TIMEOUT)
											Expect(bindResponse).To(Exit(0))

											startResponse := cf.Cf("start", app2Name).Wait(LONG_TIMEOUT)
											Expect(startResponse).To(Exit(0))
										})
									})
									AfterEach(func() {
										workflowhelpers.AsUser(patsTestSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
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

											fname = eventuallyExpect(readWriteAppURL+"/create", "pora")
										})
										AfterEach(func() {
											eventuallyExpect(fmt.Sprintf("%s/delete/%s", readWriteAppURL, fname), fname)
										})

										It("should be readable by the second app", func() {
											eventuallyExpect(fmt.Sprintf("%s/read/%s", app2URL, fname), "Hello Persistent World")
										})

										It("should not be deletable by the second app", func() {
											body, status, _ := get(fmt.Sprintf("%s/delete/%s", app2URL, fname), printErrorsOn)
											Expect(body).NotTo(ContainSubstring("deleted"))
											Expect(status).NotTo(Equal(http.StatusOK))
										})
									})
								})
							}
							if os.Getenv("TEST_READ_ONLY") == "true" {
								Context("when a second app is bound with a readonly mount", func() {
									var (
										app2Name string
									)
									BeforeEach(func() {
										app2Name = appName + "-2"
										workflowhelpers.AsUser(patsTestSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
											if os.Getenv("TEST_DOCKER_PORA") == "true" {
												Eventually(cf.Cf("push", app2Name, "--docker-image", "cfpersi/pora", "--no-start"), DEFAULT_TIMEOUT).Should(Exit(0))
											} else {
												appPath = os.Getenv("TEST_APPLICATION_PATH")
												Expect(appPath).To(BeADirectory(), "TEST_APPLICATION_PATH environment variable should point to a CF application")
												Eventually(cf.Cf("push", app2Name, "-p", appPath, "-f", appPath+"/manifest.yml", "--no-start"), DEFAULT_TIMEOUT).Should(Exit(0))
											}

											if bindConfigToUse == "" || strings.Contains(bindConfigToUse, "{}") {
												bindResponse := cf.Cf("bind-service", app2Name, instanceName, "-c", `{"readonly":true}`).Wait(DEFAULT_TIMEOUT)
												Expect(bindResponse).To(Exit(0))

											} else {
												bindResponse := cf.Cf("bind-service", app2Name, instanceName, "-c", strings.Replace(bindConfigToUse, "}", `,"readonly":true}`, 1)).Wait(DEFAULT_TIMEOUT)
												Expect(bindResponse).To(Exit(0))
											}

											startResponse := cf.Cf("start", app2Name).Wait(LONG_TIMEOUT)
											Expect(startResponse).To(Exit(0))
										})
									})
									AfterEach(func() {
										workflowhelpers.AsUser(patsTestSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
											cf.Cf("unbind-service", app2Name, instanceName).Wait(DEFAULT_TIMEOUT)

											cf.Cf("delete", app2Name, "-r", "-f").Wait(DEFAULT_TIMEOUT)
										})
									})

									It("should include the volume mount as read only in the second application's environment", func() {
										workflowhelpers.AsUser(patsTestSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
											env := cf.Cf("env", app2Name).Wait(DEFAULT_TIMEOUT)
											Expect(env).To(Exit(0))
											Expect(env).To(Say(pConfig.ServiceName))
											Expect(env).To(Say(instanceName))
											Expect(env).To(Say(`"r"`))
										})
									})

									Context("when the second app tries to write a file", func() {
										var (
											body    string
											app2URL string
										)
										BeforeEach(func() {
											app2URL = "http://" + app2Name + "." + cfConfig.AppsDomain

											body, _, _ = get(app2URL+"/create", printErrorsOff)
										})

										It("should fail to write the file", func() {
											Expect(body).To(ContainSubstring("read-only file system"))
										})
									})

									Context("when the first app creates a file", func() {
										var (
											fname          string
											readOnlyAppURL string
										)
										BeforeEach(func() {
											readOnlyAppURL = "http://" + app2Name + "." + cfConfig.AppsDomain

											fname = eventuallyExpect(readWriteAppURL+"/create", "pora")
										})
										AfterEach(func() {
											eventuallyExpect(fmt.Sprintf("%s/delete/%s", readWriteAppURL, fname), fname)
										})

										It("should be readable by the second app", func() {
											eventuallyExpect(fmt.Sprintf("%s/read/%s", readOnlyAppURL, fname), "Hello Persistent World")
										})
									})
								})
							}
						})

						Context("failing bind configs", func() {
							AfterEach(func() {
								workflowhelpers.AsUser(patsTestSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
									cf.Cf("logs", appName, "--recent").Wait(DEFAULT_TIMEOUT)
									cf.Cf("stop", appName).Wait(DEFAULT_TIMEOUT)
									cf.Cf("unbind-service", appName, instanceName).Wait(DEFAULT_TIMEOUT)
								})
							})

							It("fails to bind if bind config is not allowed due to LDAP", func() {
								if pConfig.DisallowedLdapBindConfig == "" {
									Skip("not testing LDAP config")
								}

								workflowhelpers.AsUser(patsTestSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
									bindResponse := cf.Cf("bind-service", appName, instanceName, "-c", pConfig.DisallowedLdapBindConfig).Wait(DEFAULT_TIMEOUT)
									Expect(bindResponse).NotTo(Exit(0))
								})
							})

							It("fails to bind if bind config has non allowed overrides", func() {
								if pConfig.DisallowedOverrideBindConfig == "" {
									Skip("not testing disallowed config override")
								}

								workflowhelpers.AsUser(patsTestSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
									bindResponse := cf.Cf("bind-service", appName, instanceName, "-c", pConfig.DisallowedOverrideBindConfig).Wait(DEFAULT_TIMEOUT)
									Expect(bindResponse).NotTo(Exit(0))
								})
							})
						})
					})
				})

				Context("given multiple bind configs", func() {

					BeforeEach(func() {
						workflowhelpers.AsUser(patsTestSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
							Eventually(cf.Cf("push", appName, "--docker-image", "cfpersi/pora", "--no-start"), DEFAULT_TIMEOUT).Should(Exit(0))
						})
					})

					AfterEach(func() {
						workflowhelpers.AsUser(patsTestSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
							cf.Cf("logs", appName, "--recent").Wait(DEFAULT_TIMEOUT)
							cf.Cf("stop", appName).Wait(DEFAULT_TIMEOUT)

							cf.Cf("unbind-service", appName, instanceName).Wait(DEFAULT_TIMEOUT)

							cf.Cf("delete", appName, "-r", "-f").Wait(DEFAULT_TIMEOUT)
						})
					})

					for _, bindConfig := range pConfig.BindConfig {
						var bindConfigToUse = bindConfig

						Context(fmt.Sprintf("when the app is bound with config [%s]", bindConfigToUse), func() {

							BeforeEach(func() {
								if bindConfigToUse == "" {
									bindResponse := cf.Cf("bind-service", appName, instanceName).Wait(DEFAULT_TIMEOUT)
									Expect(bindResponse).To(Exit(0))
								} else {
									bindResponse := cf.Cf("bind-service", appName, instanceName, "-c", bindConfigToUse).Wait(DEFAULT_TIMEOUT)
									Expect(bindResponse).To(Exit(0))
								}
							})

							AfterEach(func() {
								workflowhelpers.AsUser(patsTestSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
									cf.Cf("unbind-service", appName, instanceName).Wait(DEFAULT_TIMEOUT)
								})
							})

							Context("when the app is started", func() {
								BeforeEach(func() {
									workflowhelpers.AsUser(patsTestSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
										bindResponse := cf.Cf("start", appName).Wait(LONG_TIMEOUT)
										Expect(bindResponse).To(Exit(0))
									})
								})

								AfterEach(func() {
									workflowhelpers.AsUser(patsTestSetup.RegularUserContext(), DEFAULT_TIMEOUT, func() {
										cf.Cf("logs", appName, "--recent").Wait(DEFAULT_TIMEOUT)
										cf.Cf("stop", appName).Wait(DEFAULT_TIMEOUT)
									})
								})

								It("should verify that the app mounted the volume", func() {
									By("verifying that the app is able to write to the volume")

									eventuallyExpect(readWriteAppURL+"/write", "Hello Persistent World")
								})
							})
						})
					}
				})
			})
		})
	})
})

func get(uri string, printErrors bool) (body string, status int, err error) {
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return "", status, err
	}

	response, err := (&http.Client{}).Do(req)
	if err != nil {
		return "", status, err
	}

	bodyBytes, err := ioutil.ReadAll(response.Body)
	defer response.Body.Close()

	if printErrors && response.StatusCode >= http.StatusInternalServerError {
		fmt.Println(fmt.Sprintf("Request: [[%v]]\nResponse: [[%v]] [[%s]]", req, response, string(bodyBytes)))
	}

	return string(bodyBytes[:]), response.StatusCode, err
}

func eventuallyExpect(endpoint string, expectedSubstring string) string {
	Eventually(func() int {
		_, status, err := get(endpoint, printErrorsOn)
		Expect(err).NotTo(HaveOccurred())
		return status
	}, 5 * time.Second, 1 * time.Second).Should(Equal(http.StatusOK))

	var body string
	Eventually(func() string {
		var err error
		body, _, err = get(endpoint, printErrorsOn)
		Expect(err).NotTo(HaveOccurred())
		return body
	}, 5 * time.Second, 1 * time.Second).Should(ContainSubstring(expectedSubstring))

	return body
}
