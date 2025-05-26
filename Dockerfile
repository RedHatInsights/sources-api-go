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

COPY --from=build /build/sources-api-go /sources-api-go

COPY licenses/LICENSE /licenses/LICENSE

USER 1001

ENTRYPOINT ["/sources-api-go"]
