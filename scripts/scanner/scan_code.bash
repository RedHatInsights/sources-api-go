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
# Copy the repository contents so we don't have to deal with changing permissions on the mounted volume.
#
source_dir=$(mktemp --directory)
cp --recursive "/repository" "${source_dir}"

#
# Run the sonar scanner.
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
