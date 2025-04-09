#!/bin/bash

#
# Bash safety options:
#   - e is for exiting immediately when a command exits with a non-zero status.
#   - u is for treating unset variables as an error when substituting.
#   - x is for printing all the commands as they're executed.
#   - o pipefail is for taking into account the exit status of the commands
#     that run on pipelines.
#
set -euxo pipefail

#
# Get the commit SHA to give the scanner a unique "project version" setting.
#
commit_short=$(git rev-parse --short=7 HEAD)
readonly commit_short

#
# Get the current directory since it represents the project's root directory.
#
current_directory=$(pwd)
readonly current_directory

#
# Set the project's key.
#
project_key="insights-sources"
readonly project_key

#
# Run the Sonar Scanner in a container. The "repository" directory is just the
# source code, and we mount it to be able to scan the source code.
#
# Should we be running on main, then skip passing the information regarding the
# pull request.
#
# Finally, the volume is mounted with the "z" option, because it seems like
# the runner has SELinux activated, and that we need to relabel the directory
# to have permission to read it. More information here: https://www.reddit.com/r/podman/comments/fww87v/permission_denied_within_mounted_volume_inside/
#
if [ -n "${GIT_BRANCH:-}" ] && [ "${GIT_BRANCH}" == "origin/main" ]; then
  podman run \
    --env COMMIT_SHORT="${commit_short}" \
    --env GIT_BRANCH="${GIT_BRANCH}" \
    --env SONARQUBE_HOST_URL="${SONARQUBE_HOST_URL}" \
    --env SONARQUBE_PROJECT_KEY="${project_key}" \
    --env SONARQUBE_TOKEN="${SONARQUBE_TOKEN}" \
    --rm \
    --volume "${current_directory}":/repository:z \
    images.paas.redhat.com/alm/sonar-scanner:latest \
    bash /repository/scripts/scanner/scan_code.bash
else
  podman run \
    --env COMMIT_SHORT="${commit_short}" \
    --env GIT_BRANCH="${GIT_BRANCH}" \
    --env GITHUB_PULL_REQUEST_ID="${ghprbPullId}" \
    --env SONARQUBE_HOST_URL="${SONARQUBE_HOST_URL}" \
    --env SONARQUBE_PROJECT_KEY="${project_key}" \
    --env SONARQUBE_TOKEN="${SONARQUBE_TOKEN}" \
    --rm \
    --volume "${current_directory}":/repository:z \
    images.paas.redhat.com/alm/sonar-scanner:latest \
    bash /repository/scripts/scanner/scan_code.bash
fi

# We need to make a dummy results file to make tests pass.
mkdir -p artifacts
cat << EOF > artifacts/junit-dummy.xml
<testsuite tests="1">
  <testcase classname="dummy" name="dummytest"/>
</testsuite>
EOF
