FROM registry.access.redhat.com/ubi9/ubi:latest@sha256:beba9460df08acc9f2d5a99f61b8544d5367b05e02c945ca726c6f07ca5963a7 as build
WORKDIR /build

RUN dnf --assumeyes --disableplugin=subscription-manager install go

COPY . .
RUN go mod download \
    && go build -o sources-api-go . \
    && strip sources-api-go

FROM registry.access.redhat.com/ubi9/ubi-minimal:latest@sha256:a21d19f72557aa893a0f6ca1177bb1ffd91df6015e60470a116c50a6a05efcd9

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
COPY --from=build /build/sources-api-go /app/sources-api-go
COPY licenses/LICENSE /app/licenses/LICENSE

USER 1001

ENTRYPOINT ["/app/sources-api-go"]
