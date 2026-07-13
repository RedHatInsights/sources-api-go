FROM registry.access.redhat.com/ubi9/ubi:latest AS build
WORKDIR /build

ENV GOLANG_VERSION=1.26.5
RUN curl -fsSL https://go.dev/dl/go${GOLANG_VERSION}.linux-amd64.tar.gz \
    | tar -C /usr/local -xz
ENV PATH=/usr/local/go/bin:$PATH

COPY . .
RUN go mod download \
    && go build -o sources-api-go . \
    && strip sources-api-go

FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

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
