include .env
.EXPORT_ALL_VARIABLES:

ENV=local
GENPROTO_OUT=/pkg/genproto
TEST_FILE=.
GO_VERSION=1.22

dev-up:
	docker compose up -d

dev-down:
	docker compose down

migrate-up:
	go run scripts/migrate/main.go up

migrate-down:
	go run scripts/migrate/main.go down

gen-proto:
	go run scripts/gen_proto/main.go

dev-merchant-gateway:
	go run cmd/server/merchantgateway/main.go
