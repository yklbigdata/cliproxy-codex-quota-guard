BIN_DIR ?= dist
PLUGIN := $(BIN_DIR)/codex-quota-guard.so

GOOS ?= linux
GOARCH ?= amd64

.PHONY: test build tidy vet clean

tidy:
	go mod tidy

test:
	go test ./...

vet:
	go vet ./...

build:
	mkdir -p $(BIN_DIR)
	CGO_ENABLED=1 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -buildmode=c-shared -trimpath -o $(PLUGIN) .

clean:
	rm -rf $(BIN_DIR)
