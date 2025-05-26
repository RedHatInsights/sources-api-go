FROM registry.access.redhat.com/ubi9/ubi:latest as build
WORKDIR /build

RUN dnf --assumeyes --disableplugin=subscription-manager install go

# We need to override the toolchain to the latest version because
# unfortunately the latest "ubi8" image does not contain the go version 1.23,
# which is required for the latest dependency updates.
ARG GOTOOLCHAIN=go1.24.3

COPY . .
RUN go mod download \
    && go build -o sources-api-go . \
    && strip sources-api-go

FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

# The Sources API leaves the RDS CA in a file when the Clowder configuration
# is loaded. In order to avoid permission errors and having to run the
# container with elevated privileges, we create a "sources" user and give it
# an "/app" directory from where it will have permissions to create that file.
RUN /sbin/useradd --shell /bin/sh sources \
    && mkdir /app \
    && chown sources /app

# Copy the binary and the license.
COPY --from=build /build/sources-api-go /app/sources-api-go
COPY licenses/LICENSE /app/licenses/LICENSE

# Set the working directory to the "sources"-owned directory.
WORKDIR /app
USER sources

ENTRYPOINT ["./sources-api-go"]
