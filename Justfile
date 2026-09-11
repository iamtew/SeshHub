default:
    @just --list

deps:
    go mod tidy

dev *args:
    go run ./cmd/server {{args}}

test:
    go test ./...

build:
    go build -ldflags="-s -w" -o dist/seshhub.exe ./cmd/server
