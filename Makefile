fly-k8s-smb: SHELL:=/bin/bash
fly-k8s-smb:
	PARALLEL_NODES=2 TEST_MOUNT_FAIL_LOGGING=false TEST_MOUNT_OPTIONS=true TEST_MULTI_CELL=false TEST_READ_ONLY=true \
	fly -t persi execute \
	-c /Users/pivotal/workspace/persi-ci/scripts/ci/run-pats.build.yml \
	-i persi-ci=/Users/pivotal/workspace/persi-ci \
	-i pats-config=/tmp/pats-config \
	-i cf-volume-services-acceptance-tests=/Users/pivotal/workspace/cf-volume-services-acceptance-tests

fly-nfs: SHELL:=/bin/bash
fly-nfs:
	mkdir -p /tmp/pats-config/

	FLY_BUILD=`BBL_STATE_DIR=bbl-state-gcp-gorgophone APPS_DOMAIN=gorgophone.cf-app.com CF_API_ENDPOINT=api.gorgophone.cf-app.com CF_USERNAME=admin BIND_BOGUS_CONFIG='{"uid":"1000","gid":"1000"}' BIND_CONFIG='["{\"uid\":\"1000\",\"gid\":\"1000\"}", "{\"uid\":\"1000\",\"gid\":\"1000\",\"mount\": \"/var/vcap/data/foo\"}", "{\"uid\":\"1000\",\"gid\":\"1000\", \"version\": \"3.0\"}", "{\"uid\":\"1000\",\"gid\":\"1000\",\"version\": \"3.0\",\"mount\": \"/var/vcap/data/foo\"}", "{\"uid\":\"1000\",\"gid\":\"1000\", \"version\": \"4.1\"}", "{\"uid\":\"1000\",\"gid\":\"1000\",\"version\": \"4.1\",\"mount\": \"/var/vcap/data/foo\"}", "{\"uid\":\"1000\",\"gid\":\"1000\", \"version\": \"4.2\"}", "{\"uid\":\"1000\",\"gid\":\"1000\",\"version\": \"4.2\",\"mount\": \"/var/vcap/data/foo\"}"]' CREATE_BOGUS_CONFIG='{"share":"nfstestserver.service.cf.internal/export/nonexistensevol"}' CREATE_CONFIG='{"share":"nfstestserver.service.cf.internal/export/users"}' PLAN_NAME=Existing SERVICE_NAME=nfs \
               	fly -t persi execute -c /Users/pivotal/workspace/persi-ci/scripts/ci/generate_pats_config.build.yml -i persi-ci=/Users/pivotal/workspace/persi-ci -i director-state=/Users/pivotal/workspace/gorgophone-env | grep 'executing build' | awk '{print $$3}'`; \
	echo $$FLY_BUILD; \
	fly -t persi hijack -b $$FLY_BUILD cat pats-config/pats.json > /tmp/pats-config/pats.json

	PARALLEL_NODES=2 TEST_MOUNT_FAIL_LOGGING=true TEST_MOUNT_OPTIONS=true TEST_MULTI_CELL=true TEST_READ_ONLY=true \
	fly -t persi execute \
	-c /Users/pivotal/workspace/persi-ci/scripts/ci/run-pats.build.yml \
	-i persi-ci=/Users/pivotal/workspace/persi-ci \
	-i pats-config=/tmp/pats-config \
	-i cf-volume-services-acceptance-tests=/Users/pivotal/workspace/cf-volume-services-acceptance-tests


.PHONY: fly-nfs