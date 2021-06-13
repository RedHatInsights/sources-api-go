FROM registry.access.redhat.com/ubi8/ubi-minimal:8.4 as build
MAINTAINER jlindgre@redhat.com

RUN mkdir /build
WORKDIR /build

RUN microdnf install go

COPY go.mod .
RUN go mod download 

COPY . .
RUN go build

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.4
COPY --from=build /build/sources-api-go /sources-api-go
ENTRYPOINT ["/sources-api-go"]
