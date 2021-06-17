all: build

setup:
	go mod download

tidy:
	go mod tidy

build:
	go build -o sources-api-go .

clean:
	go clean

run: build
	./sources-api-go

inlinerun:
	go run *.go

container:
	docker build . -t sources-api-go

debug:
	DEBUG_SQL=true dlv debug --headless --listen=:2345 --api-version=2 --accept-multiclient

.PHONY: setup tidy build clean run container debug