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
	go run .

listener:
	go run . -listener

backgroundworker:
	go run . -background-worker

container:
	docker build . -t sources-api-go

remotedebug:
	dlv debug --headless --listen=:2345 --api-version=2 --accept-multiclient

debug:
	dlv debug

test:
	go test ./...

alltest: test
	go test ./... --integration

lint:
	go vet ./...
	golangci-lint run -E gofmt,gci,bodyclose,forcetypeassert,misspell

gci:
	golangci-lint run -E gci --fix

vault:
	# runs a server locally - with `root` as the token. useful for development
	docker run -it --rm --cap-add=IPC_LOCK \
		-e 'VAULT_DEV_ROOT_TOKEN_ID=root' \
		-e 'VAULT_DEV_LISTEN_ADDRESS=0.0.0.0:8200' \
		-p 8200:8200 vault

generate:
	go generate ./...

migration:
	@sh db/migrations/new_migration.sh

docker-up:
	@docker-compose up -d sources-db sources-kafka
	@sleep 7
	@mkdir -p tmp/db
	@docker-compose up sources-api-db-setup
	@docker-compose up init-kafka
	@docker-compose up -d

.PHONY: setup tidy build clean run container remotedebug debug test lint gci vault listener alltest generate
