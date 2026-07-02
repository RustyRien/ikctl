BIN_DIR := bin
BIN_NAME := ikctl

.PHONY: build
build:
	go build -o $(BIN_DIR)/$(BIN_NAME) .

.PHONY: test
test:
	go test ./...

.PHONY: run
run:
	go run .
