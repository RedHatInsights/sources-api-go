all: build

setup: tidy
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
	go run `ls *.go | grep -v test`

container:
	docker build . -t sources-api-go

remotedebug:
	dlv debug --headless --listen=:2345 --api-version=2 --accept-multiclient

debug:
	dlv debug

test:
	go test ./...

lint:
	go vet ./...
	golangci-lint run -E gofmt,gci,bodyclose,forcetypeassert,misspell

.PHONY: setup tidy build clean run container remotedebug debug test lint
