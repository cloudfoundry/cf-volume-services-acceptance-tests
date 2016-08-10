# Persi Acceptance Tests
These tests are used to certify Diego Persistence end-to-end functionality
# Installation

Prereqs:
- [go](https://golang.org/dl/)
- ginkgo and gomega; i.e.
```
go get github.com/onsi/ginkgo/ginkgo
go get github.com/onsi/gomega
```
To install pats tests in your `GOPATH`:
```
go get github.com/cloudfoundry-incubator/persi_acceptance_test
```


```bash
cat > integration_config.json <<EOF
{
  "api": "api.bosh-lite.com",
  "admin_user": "admin",
  "admin_password": "admin",
  "apps_domain": "bosh-lite.com",
  "skip_ssl_validation": true,
  "use_http": true,
  "service_name": "pats-service",
  "plan_name": "free",
  "broker_url": "http://pats-broker.bosh-lite.com",
  "broker_user": "admin",
  "broker_password": "admin"
}
EOF
export CONFIG=$PWD/integration_config.json
```

# Acceptance

