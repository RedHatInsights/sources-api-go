################################
# STEP 1 build executable binary
################################
FROM registry.access.redhat.com/hi/go:latest-fips-builder@sha256:12999eb49f363d9e580dde98e911bd5ba3a9d1deb484d77ab8822720e32bbc97 AS builder

USER 0

WORKDIR /workspace

# Cache deps before copying source so that we do not need to re-download for every build
COPY go.mod go.sum .

# Fetch dependencies hermetically by sourcing the local prefetch cache
RUN if [ -f /cachi2/cachi2.env ]; then . /cachi2/cachi2.env; fi && \
    go mod download

# Now copy the rest of the files for build
COPY . .

# Build the binary using the local cached modules
RUN if [ -f /cachi2/cachi2.env ]; then . /cachi2/cachi2.env; fi && \
    go build -ldflags "-w -s" -o sources-api-go .

############################
# STEP 2 build a small image
############################
FROM registry.access.redhat.com/hi/go:latest-fips@sha256:ebba7318a970b8a5388ea4248e9e0b3d6bc78a0c68c9f0f937633948ff8cc8bc

# The Sources API leaves the RDS CA in a file when the Clowder configuration
# is loaded. In order to avoid permission errors when writing the file to
# the directory, we create one and give it all the permissions.
#
# We have attempted creating a particular user, creating a directory for that
# user and giving it permissions with "chown", but for some reason even
# though things worked locally they did not in stage.
WORKDIR /app
RUN chmod 777 /app

# Copy the binary and the license.
COPY --from=builder /workspace/sources-api-go /app/sources-api-go
COPY licenses/LICENSE /app/licenses/LICENSE

USER 1001

ENTRYPOINT ["/app/sources-api-go"]
