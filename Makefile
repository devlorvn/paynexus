.EXPORT_ALL_VARIABLES:

ENV=local
COMPOSE_PROJECT_NAME=paynexus
LD_LIBRARY_PATH:=$(LD_LIBRARY_PATH):/usr/local/lib
DOCKER_FILE=./developments/release.Dockerfile
COMPOSE_FILE=./developments/proto.docker-compose.yml
GENPROTO_OUT=/pkg/genproto
TEST_FILE=.
GO_VERSION ?= $(shell cat ./deployments/versions/go)


gen-proto:
	go run scripts/gen_proto.go
dev-up:
	docker compose up -d
dev-down:
	docker compose down