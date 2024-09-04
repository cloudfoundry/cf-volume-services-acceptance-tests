package persi_acceptance

import (
	"encoding/json"
	"os"

	"github.com/cloudfoundry/cf-test-helpers/v2/config"
)

type Config struct {
	ServiceName             string `json:"service_name"`
	BrokerName              string `json:"broker_name"`
	PlanName                string `json:"plan_name"`
	IncludeIsolationSegment bool   `json:"include_isolation_segment"`
	IncludeMultiCell        bool   `json:"include_multi_cell"`
	Username                string `json:"username"`
	Password                string `json:"password"`
}

func LoadConfig() (Config, error) {
	configFile, err := os.Open(config.ConfigPath())
	if err != nil {
		return Config{}, err
	}

	defer configFile.Close()

	decoder := json.NewDecoder(configFile)

	config := &Config{}
	err = decoder.Decode(config)
	if err != nil {
		return Config{}, err
	}

	return *config, nil
}
