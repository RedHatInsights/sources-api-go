#!/bin/bash

#
# Before running the IQE tests, run the unit and integration tests first for a
# faster development cycle.
#
# Spin up the database for integration tests.
db_container_name="sources-api-db-$(uuidgen)"
readonly db_container_name

echo "Spinning up database container: ${db_container_name}"

podman run \
    --detach \
    --env POSTGRESQL_DATABASE=sources_api_test_go \
    --env POSTGRESQL_PASSWORD=toor \
    --env POSTGRESQL_USER=root \
    --name "${db_container_name}" \
    --publish 5432 \
    quay.io/cloudservices/postgresql-rds:12

database_port=$(podman inspect "${db_container_name}" | grep HostPort | sort | uniq | grep --only-matching "[0-9]*")

echo "Database listening on port: ${database_port}"

export DATABASE_HOST=localhost
export DATABASE_NAME=sources_api_test_go
export DATABASE_PASSWORD=toor
export DATABASE_PORT=$database_port
export DATABASE_USER=root
export GOROOT="/opt/go/1.21.3"
export PATH="${GOROOT}/bin:${PATH}"

echo "Running tests..."

make alltest

result_code=$?
readonly result_code

echo "Stopping database container..."
podman stop "${db_container_name}"

echo "Removing database container..."
podman rm --force "{$db_container_name}"

if [[ $result_code != 0 ]]; then
    exit 1
fi

# --------------------------------------------
# Options that must be configured by app owner
# --------------------------------------------
APP_NAME="sources"  # name of app-sre "application" folder this component lives in
COMPONENT_NAME="sources-api"  # name of app-sre "resourceTemplate" in deploy.yaml for this component
IMAGE="quay.io/cloudservices/sources-api-go"

IQE_CJI_TIMEOUT="30m"  # This is the time to wait for smoke test to complete or fail
IQE_FILTER_EXPRESSION=""  # This is the value passed to pytest -k
IQE_IMAGE_TAG="sources"
IQE_MARKER_EXPRESSION="smoke and not ui"  # This is the value passed to pytest -m
IQE_PARALLEL_ENABLED="false"
IQE_PLUGINS="sources"  # name of the IQE plugin for this app.

# Install and configur Bonfire to be able to build, deploy and run the tests
# in the ephemeral environment.

curl -s "https://raw.githubusercontent.com/RedHatInsights/bonfire/master/cicd/bootstrap.sh" > .cicd_bootstrap.sh && source .cicd_bootstrap.sh

if ! source "${CICD_ROOT}/build.sh"; then
    exit 1
fi

if ! source "${CICD_ROOT}/deploy_ephemeral_env.sh"; then
    exit 1
fi

if ! source "${CICD_ROOT}/cji_smoke_test.sh"; then
    exit 1
fi

# Need to make a dummy results file to make tests pass
mkdir -p artifacts
cat << EOF > artifacts/junit-dummy.xml
<testsuite tests="1">
    <testcase classname="dummy" name="dummytest"/>
</testsuite>
EOF
