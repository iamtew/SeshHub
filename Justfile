default:
    @just --list

deps:
    go mod tidy

dev *args:
    go run ./cmd/server {{args}}

test:
    go test ./...

[unix]
build:
    mkdir -p bin
    go build -ldflags="-s -w" -o bin/seshhub ./cmd/server

[windows]
build:
    if not exist dist mkdir dist
    go build -ldflags="-s -w" -o dist/seshhub.exe ./cmd/server

[unix]
clean:
    rm -rf bin dist

[windows]
clean:
    if exist bin rmdir /s /q bin
    if exist dist rmdir /s /q dist
