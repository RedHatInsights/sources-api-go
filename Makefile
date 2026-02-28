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
	golangci-lint run

gci:
	golangci-lint run --fix

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


run_dependencies:
	podman-compose -f dev.yml up

run_local_instance:
	DATABASE_HOST=localhost \
	DATABASE_PORT=5442 \
	DATABASE_NAME=sources-db \
	DATABASE_USER=insights \
	DATABASE_PASSWORD=insights \
	REDIS_CACHE_HOST=localhost \
	REDIS_CACHE_PORT=6379 \
	BYPASS_RBAC=true \
	LOG_LEVEL=debug \
	./sources-api-go


IDENTITY := $(shell echo -n '{"identity": {"account_number": "000202", "internal": {"org_id": "000101"}, "type": "User", "org_id": "000101", "auth_type": "jwt-auth", "user":{"username": "wilma", "user_id": "wilma-1"}}}' | base64 -w 0)


GRAPH_QUERY='{"query":"{\n meta {\n  count\n }\n sources(\n  offset: 0\n  limit: 50\n  sort_by: { name: \"created_at\", direction: desc }\n  filter: {\n   name: \"source_type.vendor\"\n   operation: \"not_eq\"\n   value: \"Red Hat\"\n  }\n ) {\n  id\n  created_at\n  app_creation_workflow\n  source_type_id\n  name\n  imported\n  availability_status\n  source_ref\n  last_checked_at\n  updated_at\n  last_available_at\n  app_creation_workflow\n  paused_at\n  authentications {\n   authtype\n   username\n  }\n  applications {\n   application_type_id\n   id\n   availability_status_error\n   availability_status\n   paused_at\n   extra\n   authentications {\n    username\n    authtype\n   }\n  }\n  endpoints {\n   id\n   scheme\n   host\n   port\n   path\n   receptor_node\n   role\n   certificate_authority\n   verify_ssl\n   availability_status_error\n   availability_status\n   authentications {\n    authtype\n    availability_status\n    availability_status_error\n   }\n  }\n }\n}\n"}'


example_graphql:
	curl -v \
	-X POST \
	-H "Content-Type: application/json" \
    -H "x-rh-identity: ${IDENTITY}" \
	-d ${GRAPH_QUERY} \
	-H "x-rh-sources-psk: thisMustBeEphemeralOrMinikube" \
	-H "x-rh-sources-org-id: 000001" \
	-H "x-rh-sources-account-number: 0000002" \
	-H "x-rh-insights-request-id: 1238" \
	-H "x-rh-sources-user-id: user-000101-2" \
	http://localhost:8001/api/sources/v3/graphql | jq

example_create_source:
	curl -v \
	-X POST \
    -H "x-rh-identity: ${IDENTITY}" \
	-H "x-rh-sources-user-id: 1" \
	-d '{"name": "wilma-sources-5", "version": "0.1", "uid": "dead10ba-98b6-11f0-8cb2-0a226ef3a833", "source_type_id": "1" }' \
	http://localhost:8000/api/sources/v3/sources

example_list_source:
	curl -v \
	http://localhost:8001/api/sources/v3/sources | jq

example_list_app_meta_data:
	curl -v \
	http://localhost:8001/api/sources/v3/app_meta_data | jq

example_list_source_with_psk:
	curl -v \
    -H "x-rh-identity: ${IDENTITY}" \
	-H "x-rh-sources-psk: thisMustBeEphemeralOrMinikube" \
	-H "x-rh-sources-org-id: 000001" \
	-H "x-rh-sources-account-number: 0000002" \
	-H "x-rh-insights-request-id: 1238" \
	-H "x-rh-sources-user-id: user-000101-2" \
	http://localhost:8001/api/sources/v3/sources/1

.PHONY: setup tidy build clean run container remotedebug debug test lint gci vault listener alltest generate
