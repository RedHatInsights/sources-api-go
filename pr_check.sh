#!/bin/bash

# --------------------------------------------
# Options that must be configured by app owner
# --------------------------------------------
APP_NAME="sources"  # name of app-sre "application" folder this component lives in
COMPONENT_NAME="sources-api"  # name of app-sre "resourceTemplate" in deploy.yaml for this component
IMAGE="quay.io/cloudservices/sources-api-go"

IQE_PLUGINS="sources"  # name of the IQE plugin for this app.
IQE_MARKER_EXPRESSION="sources_smoke"  # This is the value passed to pytest -m
IQE_FILTER_EXPRESSION=""  # This is the value passed to pytest -k
IQE_CJI_TIMEOUT="30m"  # This is the time to wait for smoke test to complete or fail

# Install bonfire repo/initialize
# https://raw.githubusercontent.com/RedHatInsights/bonfire/master/cicd/bootstrap.sh
# This script automates the install / config of bonfire
CICD_URL=https://raw.githubusercontent.com/RedHatInsights/bonfire/master/cicd
curl -s $CICD_URL/bootstrap.sh > .cicd_bootstrap.sh && source .cicd_bootstrap.sh

source $CICD_ROOT/build.sh
if [[ $? != 0 ]]; then
    exit 1
fi
source $CICD_ROOT/deploy_ephemeral_env.sh
if [[ $? != 0 ]]; then
    exit 1
fi
source $CICD_ROOT/cji_smoke_test.sh
if [[ $? != 0 ]]; then
    exit 1
fi

# spin up the db for integration tests
DB_CONTAINER="sources-api-db-$(uuidgen)"
echo "Spinning up container: ${DB_CONTAINER}"

docker run -d \
    --name $DB_CONTAINER \
    -p 5432 \
    -e POSTGRESQL_USER=root \
    -e POSTGRESQL_PASSWORD=toor \
    -e POSTGRESQL_DATABASE=sources_api_test_go \
    quay.io/cloudservices/postgresql-rds:12-1

PORT=$(docker inspect $DB_CONTAINER | grep HostPort | sort | uniq | grep -o [0-9]*)
echo "DB Listening on Port: ${PORT}"

export DATABASE_HOST=localhost
export DATABASE_PORT=$PORT
export DATABASE_USER=root
export DATABASE_PASSWORD=toor
export DATABASE_NAME=sources_api_test_go

echo "Running tests...here we go"

export GOROOT="/opt/go/1.16.10"
export PATH="${GOROOT}/bin:${PATH}"
make alltest

OUT_CODE=$?

echo "Killing DB Container..."
docker kill $DB_CONTAINER
echo "Removing DB Container..."
docker rm -f $DB_CONTAINER

if [[ $OUT_CODE != 0 ]]; then
    exit 1
fi

# Need to make a dummy results file to make tests pass
mkdir -p artifacts
cat << EOF > artifacts/junit-dummy.xml
<testsuite tests="1">
    <testcase classname="dummy" name="dummytest"/>
</testsuite>
EOF
