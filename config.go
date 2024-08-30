package persi_acceptance

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/cloudfoundry/cf-test-helpers/v2/config"
)

type BindServiceConfig struct {
	Uid      string `json:"uid,omitempty"`
	Gid      string `json:"gid,omitempty"`
	Mount    string `json:"mount,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Domain   string `json:"domain,omitempty"`
	Version  string `json:"version,omitempty"`
}
type CreateServiceConfig struct {
	Share string `json:"share"`
}

func (c CreateServiceConfig) Config() string {
	return fmt.Sprintf(`{"share": "%s"}`, c.Share)
}

func (c BindServiceConfig) Config() (string, error) {
	bytes, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

type Config struct {
	ServiceName             string `json:"service_name"`
	PlanName                string `json:"plan_name"`
	IncludeIsolationSegment bool   `json:"include_isolation_segment"`
	IncludeMultiCell        bool   `json:"include_multi_cell"`
	IsLDAP                  bool   `json:"is_ldap"`
	Valid                   struct {
		CreateService CreateServiceConfig `json:"create_service"`
		BindServices  []BindServiceConfig `json:"bind_services"`
	} `json:"valid"`
	Invalid struct {
		CreateService CreateServiceConfig `json:"create_service"`
		BindService   BindServiceConfig   `json:"bind_service"`
	} `json:"invalid"`
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
