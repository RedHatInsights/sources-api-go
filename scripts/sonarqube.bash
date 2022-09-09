#!/bin/bash

#
# Bash safety options:
#   - e is for exiting immediately when a command exits with a non-zero status.
#   - u is for treating unset variables as an error when substituting.
#   - x is for printing all the commands as they're executed.
#   - o pipefail is for taking into account the exit status of the commands that run on pipelines.
#
set -euxo pipefail

#
# Get the commit SHA to give the scanner a unique "project version" setting.
#
commit_short=$(git rev-parse --short=7 HEAD)

#
# Get the current directory since it represents the project's root directory.
#
current_directory=$(pwd)

#
# Run the Sonar Scanner in a container. The "repository" directory is just the source code, and we mount it to be able
# to scan the source code.
#
docker run  \
  --rm \
  --volume "${current_directory}":/repository \
  --env SONAR_LOGIN="${SONARQUBE_TOKEN}" \
  --env SONAR_HOST_URL="${SONARQUBE_REPORT_URL}" \
  --env SONAR_PROJECT_KEY="console.redhat.com:sources-api-go" \
  --env SONAR_PROJECT_VERSION="${commit_short}" \
  --env SONAR_PULL_REQUEST_BASE="main" \
  --env SONAR_PULL_REQUEST_BRANCH="${GIT_BRANCH}" \
  --env SONAR_PULL_REQUEST_KEY="${ghprbPullId}" \
  images.paas.redhat.com/alm/sonar-scanner:latest \
  bash /repository/scripts/scanner/scan_code.bash

# Need to make a dummy results file to make tests pass.
mkdir -p artifacts
cat << EOF > artifacts/junit-dummy.xml
<testsuite tests="1">
  <testcase classname="dummy" name="dummytest"/>
</testsuite>
EOF
