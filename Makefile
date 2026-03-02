BINARY_NAME=laradev
BUILD_DIR=..

.PHONY: build clean tidy

build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) .

clean:
	rm -f $(BUILD_DIR)/$(BINARY_NAME)

tidy:
	go mod tidy
