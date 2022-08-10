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
# Create a temporary file for the certificate.
#
ca_cert_path=$(mktemp)

#
# Download the Red Hat IT's certificate to be able to access the SonarQube instance. Otherwise the Sonar Scanner fails
# due to the custom certificate.
#
curl --output "${ca_cert_path}" --insecure "$RH_IT_ROOT_CA_CERT_URL"

#
# Generate the key store and leave it in the same mounted directory.
#
keytool \
    -alias "RH-IT-Root-CA" \
    -file "${ca_cert_path}" \
    -import \
    -keystore "/keystore/${KEYSTORE_NAME}" \
    -noprompt \
    -storepass "${KEYSTORE_PASSWORD}"

#
# Remove the certificate file.
#
rm "${ca_cert_path}"
