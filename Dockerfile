################################
# STEP 1 build executable binary
################################
FROM registry.access.redhat.com/hi/go:latest-fips-builder@sha256:6c04a2b61c7c827a953f25fb92e24a7f2d9529266849080ee8fc321d5d7600c8 AS builder

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
FROM registry.access.redhat.com/hi/go:latest-fips@sha256:d1c1602fe1b45c3f62ffc92aa4be01d89e793cb46edfcfb85abb82f0d5c2d43a

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
