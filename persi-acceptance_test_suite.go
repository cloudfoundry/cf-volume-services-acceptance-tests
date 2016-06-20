package persi_acceptance_tests

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"time"
)


var (
	patsContext helpers.SuiteContext
	patsConfig  helpers.Config
)

func TestPersiAcceptance(t *testing.T) {
	RegisterFailHandler(Fail)

	patsConfig = helpers.LoadConfig()
	patsContext = helpers.NewContext(patsConfig)
	environment := helpers.NewEnvironment(patsContext)

	BeforeSuite(func() {
		environment.Setup()
	})

	AfterSuite(func() {
		environment.Teardown()
	})

	componentName := "PATS Suite"
	rs := []Reporter{}

	if patsConfig.ArtifactsDirectory != "" {
		helpers.EnableCFTrace(patsConfig, componentName)
		rs = append(rs, helpers.NewJUnitReporter(patsConfig, componentName))
	}

	RunSpecsWithDefaultAndCustomReporters(t, componentName, rs)
}
