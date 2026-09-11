default:
    @just --list

deps:
    go mod tidy

dev:
    go run ./cmd/server

test:
    go test ./...

build:
    go build -ldflags="-s -w" -o dist/seshhub.exe ./cmd/server
