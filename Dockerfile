FROM registry.access.redhat.com/ubi8/ubi:latest as build
WORKDIR /build

RUN dnf -y --disableplugin=subscription-manager install go

COPY go.mod .
RUN go mod download

COPY . .
RUN go build -o sources-api-go . && strip sources-api-go

FROM registry.access.redhat.com/ubi8/ubi-minimal:latest
WORKDIR /app
RUN chmod 777 /app

COPY --from=build /build/sources-api-go /app/sources-api-go

COPY licenses/LICENSE /licenses/LICENSE

ENTRYPOINT ["/app/sources-api-go"]
