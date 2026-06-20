.PHONY: run dev build swag sqlc test lint tidy clean

run:
	go run ./cmd/main.go

dev:
	air

build:
	go build -o bin/server ./cmd/main.go

swag:
	swag init -g cmd/main.go --output docs

sqlc:
	sqlc generate

test:
	go test ./... -v -count=1

lint:
	golangci-lint run

tidy:
	go mod tidy

clean:
	rm -rf bin tmp docs

all: swag sqlc build
