FROM registry.access.redhat.com/ubi9/ubi:latest as build
WORKDIR /build

RUN dnf -y --disableplugin=subscription-manager install go

# We need to override the toolchain to the latest version because
# unfortunately the latest "ubi8" image does not contain the go version 1.23,
# which is required for the latest dependency updates.

COPY go.mod .
RUN GOTOOLCHAIN=go1.24.2 go mod download

COPY . .
RUN GOTOOLCHAIN=go1.24.2 go build -o sources-api-go . && strip sources-api-go

FROM registry.access.redhat.com/ubi9/ubi-minimal:latest
WORKDIR /app
RUN chmod 777 /app

COPY --from=build /build/sources-api-go /app/sources-api-go

COPY licenses/LICENSE /licenses/LICENSE

USER 1001

ENTRYPOINT ["/app/sources-api-go"]
