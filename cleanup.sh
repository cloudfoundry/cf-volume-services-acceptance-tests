#!/bin/bash
orgs=$(cf orgs | grep PAT*)
for org in ${orgs}; do

	cf target -o ${org}
	cf purge-service-instance -f $(cf services | grep PATs-pats-volume-instance* | awk '{print $1}')
	cf target -o o
  cf delete-org -f ${org}

done

cf purge-service-offering efs -f
cf delete-service-broker pats-broker -f

