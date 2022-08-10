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
# Include the Sonar scanner's binaries in the path so that we can issue a simple command without the full path.
#
export PATH="/sonar-scanner/sonar-scanner-${SCANNER_CLI_VERSION}-linux/bin:${PATH}"

#
# Copy the repository contents so we don't have to deal with changing permissions on the mounted volume.
#
source_dir=$(mktemp --directory)
cp --recursive "/repository" "${source_dir}"

#
# Export the location and password of the keystore to make sure that the sonar scanner uses them to trust our internal
# SonarQube instance.
#
export SONAR_SCANNER_OPTS="-Djavax.net.ssl.trustStore=/keystore/${KEYSTORE_NAME} -Djavax.net.ssl.trustStorePassword=${KEYSTORE_PASSWORD}"

#
# Run the sonar scanner. Since all the variables are passed via environment variables, there's nothing else to specify
# here!
#

sonar-scanner \
  -Dsonar.exclusions="**/*.sql" \
  -Dsonar.projectBaseDir="${source_dir}/repository" \
  -Dsonar.projectKey="${SONAR_PROJECT_KEY}" \
  -Dsonar.projectVersion="${SONAR_PROJECT_VERSION}" \
  -Dsonar.pullrequest.base="main" \
  -Dsonar.pullrequest.branch="${SONAR_PULL_REQUEST_BRANCH}" \
  -Dsonar.pullrequest.key="${SONAR_PULL_REQUEST_KEY}" \
  -Dsonar.sources="${source_dir}/repository"

#
# Clean the repository directory.
#
rm --force --recursive "${source_dir}"
