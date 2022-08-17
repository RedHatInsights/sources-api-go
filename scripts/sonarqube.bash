#!/bin/bash

#
# Bash script to run the SonarQube scanner on the code. Basically what we do is:
#
# 1. Generate a Java Keystore containing Red Hat's custom IT certificate, because the SonarQube instance has one, and
# the scanner needs to trust that certificate.
# 2. Download the Sonar scanner and extract it to a directory.
# 3. Mount the keystore, the scanner and the source code to a container.
# 4. Run the scanner on the code and push the results.
#

#
# Bash safety options:
#   - e is for exiting immediately when a command exits with a non-zero status.
#   - u is for treating unset variables as an error when substituting.
#   - x is for printing all the commands as they're executed.
#   - o pipefail is for taking into account the exit status of the commands that run on pipelines.
#
set -euxo pipefail

#
# Get the current directory since it represents the project's root directory.
#
current_directory=$(pwd)

#
# Prepare the keystore variables for generating the key store with the custom certificate.
#
keystore_dir="${current_directory}/scripts/keystore"
keystore_name="RH-IT-Root-CA.keystore"
keystore_password="redhat"

#
# Set the directory as writable by anyone so that the container can place the generated key store there.
#
chmod 777 "${keystore_dir}"

#
# Generate the key store.
#
docker run \
  --rm \
  --volume "${keystore_dir}":/keystore \
  --env KEYSTORE_NAME="${keystore_name}" \
  --env KEYSTORE_PASSWORD="${keystore_password}" \
  --env RH_IT_ROOT_CA_CERT_URL="$RH_IT_ROOT_CA_CERT_URL" \
  registry.access.redhat.com/ubi8/openjdk-11:1.14-3 \
  bash /keystore/generate_keystore.bash

#
# Revert the directory back to safe permissions.
#
chmod 755 "${keystore_dir}"

#
# Check that the key store was properly generated.
#
if [ ! -f "${keystore_dir}/${keystore_name}" ]; then
  echo "The keystore was not properly generated. File not found."
  exit 1
fi

#
# Create a temporary file for the scanner's zip file we are about to download.
#
sonar_scanner_zip=$(mktemp)

#
# Download the scanner.
#
scanner_cli_version="4.7.0.2747"
curl --output "${sonar_scanner_zip}" "https://binaries.sonarsource.com/Distribution/sonar-scanner-cli/sonar-scanner-cli-${scanner_cli_version}-linux.zip"

#
# Create a temporary directory for the unzipped contents. Give it "755" permissions so that it can be directly mounted
# and used.
#
sonar_scanner_dir=$(mktemp --directory)
chmod 755 "${sonar_scanner_dir}"

#
# Unzip the contents of the scanner in the temporary directory.
#
unzip -d "${sonar_scanner_dir}" "${sonar_scanner_zip}"

#
# Remove the zipped file.
#
rm "${sonar_scanner_zip}"

#
# Get the commit SHA to give the scanner a unique "project version" setting.
#
commit_short=$(git rev-parse --short=7 HEAD)

#
# Run the Sonar Scanner in a container. The "keystore" is mounted to make the scanner trust our SonarQube's custom
# certificate. The "repository" directory is just the source code. Finally, the "sonar-scanner" directory has the
# extracted scanner files ready to be used.
#
docker run \
  --volume "${keystore_dir}":/keystore \
  --volume "${current_directory}":/repository \
  --volume "${sonar_scanner_dir}":/sonar-scanner \
  --env KEYSTORE_NAME="${keystore_name}" \
  --env KEYSTORE_PASSWORD="${keystore_password}" \
  --env SCANNER_CLI_VERSION="${scanner_cli_version}" \
  --env SONAR_LOGIN="${SONARQUBE_TOKEN}" \
  --env SONAR_HOST_URL="${SONARQUBE_REPORT_URL}" \
  --env SONAR_PROJECT_KEY="console.redhat.com:sources-api-go" \
  --env SONAR_PROJECT_VERSION="${commit_short}" \
  --env SONAR_PULL_REQUEST_BASE="main" \
  --env SONAR_PULL_REQUEST_BRANCH="${GIT_BRANCH}" \
  --env SONAR_PULL_REQUEST_KEY="${ghprbPullId}" \
  registry.access.redhat.com/ubi8/openjdk-11:1.14-3 \
  bash /repository/scripts/scanner/scan_code.bash

#
# Clean up the files and the directories.
#
rm "${keystore_dir}/${keystore_name}"
rm --force --recursive "${sonar_scanner_dir}"

# Need to make a dummy results file to make tests pass.
mkdir -p artifacts
cat << EOF > artifacts/junit-dummy.xml
<testsuite tests="1">
  <testcase classname="dummy" name="dummytest"/>
</testsuite>
EOF
