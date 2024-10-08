package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/cloudfoundry/cf-test-helpers/v2/config"
)

type Config struct {
	ServiceName             string `json:"service_name"`
	BrokerName              string `json:"broker_name"`
	PlanName                string `json:"plan_name"`
	AppsDomain              string `json:"apps_domain"`
	IncludeIsolationSegment bool   `json:"include_isolation_segment"`
	IsolationSegmentName    string `json:"isolation_segment_name"`
	IsolationSegmentDomain  string `json:"isolation_segment_domain"`
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

func main() {
	fmt.Println("run the test instead")
}
