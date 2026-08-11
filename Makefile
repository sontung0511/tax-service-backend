.PHONY: run dev db-up db-down test fmt vet

run:
	go run ./cmd/server

dev: db-up
	go run ./cmd/server

db-up:
	docker compose up -d --wait postgres

db-down:
	docker compose down

test:
	go test ./...

fmt:
	gofmt -w ./cmd ./internal

vet:
	go vet ./...
