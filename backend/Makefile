.PHONY: proto build run test lint migrate-up migrate-down migrate-version docker-up docker-down docker-reset efk-up efk-down efk-reset efk-init efk-logs stack-up stack-down jwt-keys

# Proto generation
proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/user/v1/*.proto proto/location/v1/*.proto proto/store/v1/*.proto proto/promotion/v1/*.proto proto/payment/v1/*.proto proto/order/v1/*.proto proto/review/v1/*.proto

# Build
build:
	go build -o bin/user-service ./services/user/cmd

# Run
run:
	go run ./services/user/cmd

# Test
test:
	go test ./... -v -cover

# Lint
lint:
	golangci-lint run ./...

# Database migrations. Default service=user; override: make migrate-up SVC=location
migrate-up:
	go run ./cmd/migrate $(SVC) up

migrate-down:
	go run ./cmd/migrate $(SVC) down

migrate-version:
	go run ./cmd/migrate $(SVC) version

# Docker
docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-reset:
	docker compose down -v
	docker compose up -d

# EFK Logging Stack
efk-up:
	docker compose -f docker-compose.logging.yml up -d --build

efk-down:
	docker compose -f docker-compose.logging.yml down

efk-reset:
	docker compose -f docker-compose.logging.yml down -v
	docker compose -f docker-compose.logging.yml up -d --build

# Re-run init only (ILM policy + index template + Kibana data view)
efk-init:
	docker compose -f docker-compose.logging.yml run --rm logging-init

# Tail Fluentd logs (useful for debugging shipping pipeline)
efk-logs:
	docker compose -f docker-compose.logging.yml logs -f fluentd

# Full Stack (infra + services + EFK)
stack-up:
	docker compose -f docker-compose.logging.yml up -d --build
	docker compose up -d --build

stack-down:
	docker compose down
	docker compose -f docker-compose.logging.yml down

# JWT RSA keypair for local dev — NEVER commit secrets/jwt_private.pem
jwt-keys:
	mkdir -p secrets
	openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -out secrets/jwt_private.pem
	openssl rsa -pubout -in secrets/jwt_private.pem -out secrets/jwt_public.pem
	@echo "Generated secrets/jwt_private.pem and secrets/jwt_public.pem"
