.PHONY: run dev db-up db-down test fmt vet

COMPOSE := docker-compose

run:
	go run ./cmd/server

dev: db-up
	go run ./cmd/server

db-up:
	$(COMPOSE) up -d --wait postgres

db-down:
	$(COMPOSE) down

test:
	go test ./...

fmt:
	gofmt -w ./cmd ./internal

vet:
	go vet ./...
