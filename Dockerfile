FROM registry.access.redhat.com/ubi8/ubi:8.5-214 as build
WORKDIR /build

RUN dnf -y --disableplugin=subscription-manager install go

COPY go.mod .
RUN go mod download 

COPY . .
RUN go build -o sources-api-go .

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.5-218
COPY --from=build /build/sources-api-go /sources-api-go
ENTRYPOINT ["/sources-api-go"]
