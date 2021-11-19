#!/bin/bash

#
# This script runs SonarQube's scanner on the project. As our SonarQube instance
# uses a self-signed certificate, before running the scanner a keystore
# containing RH IT's certificate is created. After that, the scanner is
# downloaded and run.
#

set -euxo pipefail

readonly sonarqube_dir="$PWD/sonarqube"
readonly sonarqube_certs_dir="$sonarqube_dir/certs"
readonly sonarqube_download_dir="$sonarqube_dir/download"
readonly sonarqube_extract_dir="$sonarqube_dir/extract"
readonly sonarqube_store_dir="$sonarqube_dir/store"

readonly rh_it_keystore_file="$sonarqube_store_dir/RH-IT-Root-CA.keystore"
readonly rh_it_keystore_pass="redhat"
readonly rh_it_root_ca_file="$sonarqube_certs_dir/RH-IT-Root-CA.crt"

mkdir "$sonarqube_dir"
mkdir "$sonarqube_certs_dir"
mkdir "$sonarqube_download_dir"
mkdir "$sonarqube_extract_dir"
mkdir "$sonarqube_store_dir"

#
# To make SonarQube's scanner trust the SonarQube instance with the custom
# certificate, a keystore containing Red Hat IT's root certificate must be
# created, to then pass it to the scanner.
#
curl --output "$rh_it_root_ca_file" --insecure "$RH_IT_ROOT_CA_CERT_URL"

# The JAVA_HOME variable is not set by default. Even though it is usually an
# environment variable, we write it in lowercase because in this case we only
# use it in this script.
java_home="/usr/lib/jvm/jre-1.8.0"

"$java_home"/bin/keytool \
  -alias "RH-IT-Root-CA" \
  -file "$rh_it_root_ca_file" \
  -import \
  -keystore "$rh_it_keystore_file" \
  -noprompt \
  -storepass "$rh_it_keystore_pass"

#
# Download and extract sonnarcube-cli.
#
if [[ "$OSTYPE" == "darwin"* ]]; then
  readonly sonar_scanner_os="macosx"
else
  readonly sonar_scanner_os="linux"
fi

readonly sonar_scanner_cli_version="4.6.2.2472"
readonly sonar_scanner_name="sonar-scanner-$sonar_scanner_cli_version-$sonar_scanner_os"
readonly sonar_scanner_zipped_file="$sonarqube_download_dir/$sonar_scanner_name.zip"

curl --output "$sonar_scanner_zipped_file" --insecure "$SONARQUBE_CLI_URL"
unzip -d "$sonarqube_extract_dir" "$sonar_scanner_zipped_file"

#
# Export the path to the binary.
#
export PATH="$sonarqube_extract_dir/$sonar_scanner_name/bin:$PATH"

# Get the commit SHA in very short format for the project version.
commit_short=$(git rev-parse --short=7 HEAD)

#
# Run the sonar-scanner by specifying the previously created keystore.
#
export SONAR_SCANNER_OPTS="-Djavax.net.ssl.trustStore=$rh_it_keystore_file -Djavax.net.ssl.trustStorePassword=$rh_it_keystore_pass"
# The Jenkins pipeline inject the pull request ID with the lowercase variable,
# so the shellcheck rule must be disabled to avoid any issues.
# shellcheck disable=SC2154
sonar-scanner \
  -Dsonar.host.url="$SONARQUBE_REPORT_URL" \
  -Dsonar.login="$SONARQUBE_TOKEN" \
  -Dsonar.projectKey=console.redhat.com:sources-api-go \
  -Dsonar.projectVersion="$commit_short" \
  -Dsonar.pullrequest.base="main" \
  -Dsonar.pullrequest.branch="$GIT_BRANCH" \
  -Dsonar.pullrequest.key="$ghprbPullId" \
  -Dsonar.sources=.
