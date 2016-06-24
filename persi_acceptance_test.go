package persi_acceptance_tests_test

import (
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
	"os"
)

var _ = Describe("Cloud Foundry Persistence", func() {
	// given a target, org and space from suite
	It("should have a ginkgoPATS test org", func() {
		orgs := cf.Cf("orgs").Wait(DEFAULT_TIMEOUT)
		Expect(orgs).To(Exit(0))
		Expect(orgs).To(Say("ginkgoPATS"))
	})
	It("should have a ginkgoPATS test space", func() {
		orgs := cf.Cf("spaces").Wait(DEFAULT_TIMEOUT)
		Expect(orgs).To(Exit(0))
		Expect(orgs).To(Say("ginkgoPATS"))
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
				createServiceBroker := cf.Cf("create-service-broker", BROKER_NAME, patsConfig.AdminUser, patsConfig.AdminPassword, BROKER_URL).Wait(DEFAULT_TIMEOUT)
				Expect(createServiceBroker).To(Exit(0))
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
							app := cf.Cf("push", APP_NAME, "-p", appPath, "--no-start").Wait(DEFAULT_TIMEOUT)
							Expect(app).To(Exit(0))
						})
					})
					It("it should be have the app", func() {
						cf.AsUser(patsContext.RegularUserContext(), DEFAULT_TIMEOUT, func() {
							marketplaceItems := cf.Cf("apps").Wait(DEFAULT_TIMEOUT)
							Expect(marketplaceItems).To(Exit(0))
							Expect(marketplaceItems).To(Say(APP_NAME))
						})
					})
					AfterEach(func() {
						cf.AsUser(patsContext.RegularUserContext(), DEFAULT_TIMEOUT, func() {
							app := cf.Cf("delete", APP_NAME, "-r", "-f").Wait(DEFAULT_TIMEOUT)
							Expect(app).To(Exit(0))
						})
					})
					AfterEach(func() {
						// destroy service
						cf.AsUser(patsContext.RegularUserContext(), DEFAULT_TIMEOUT, func() {
							deleteServiceBroker := cf.Cf("delete-service", INSTANCE_NAME, "-f").Wait(DEFAULT_TIMEOUT)
							Expect(deleteServiceBroker).To(Exit(0))
						})
					})
				})
				AfterEach(func() {
					cf.AsUser(patsContext.RegularUserContext(), DEFAULT_TIMEOUT, func() {
						createService := cf.Cf("delete-service", INSTANCE_NAME, "-f").Wait(DEFAULT_TIMEOUT)
						Expect(createService).To(Exit(0))
					})
				})
			})
			AfterEach(func() {/*disable service*/})
		})
		AfterEach(func() {
			//destroy broker
			cf.AsUser(patsContext.AdminUserContext(), DEFAULT_TIMEOUT, func() {
				deleteServiceBroker := cf.Cf("delete-service-broker", "-f", BROKER_NAME).Wait(DEFAULT_TIMEOUT)
				Expect(deleteServiceBroker).To(Exit(0))
				Expect(deleteServiceBroker).To(Say(BROKER_NAME))
			})
		})
	})
})

