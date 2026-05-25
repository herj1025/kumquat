.PHONY: build run clean

APP_NAME=kumquat
BUILD_DIR=bin

build:
	go build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/github.com/herj1025/kumquat/

run: build
	./$(BUILD_DIR)/$(APP_NAME) server

clean:
	rm -rf $(BUILD_DIR)

test:
	go test ./...

