#!/bin/bash

DOMAIN=$(aws ssm get-parameter --output json --name pointypoints.domainname | jq .Parameter.Value -r)

WORKSPACE=`(cd ../infrastructure && terraform workspace show)`

DOMAIN_PREFIX=""
if [ $WORKSPACE != 'default' ]; then
  DOMAIN_PREFIX="${WORKSPACE}-"
fi

echo "VUE_APP_POINTING_SOCKET_URL=wss://${DOMAIN_PREFIX}pointing.${DOMAIN}" >> .env.local