VERSION    = v0.1.0
BINARY     = pulumi-resource-multipass
PLUGIN_DIR = $(HOME)/.pulumi/plugins/resource-multipass-$(VERSION)

.PHONY: build install test test-integration lint schema gen-sdk

build:
	go build -o bin/$(BINARY) ./cmd/pulumi-resource-multipass

install: build
	mkdir -p $(PLUGIN_DIR)
	cp bin/$(BINARY) $(PLUGIN_DIR)/$(BINARY)

test:
	go test ./...

test-integration:
	go test -tags integration -v -timeout 300s ./...

lint:
	golangci-lint run

schema: build
	# pulumi-go-provider binaries are gRPC servers — schema is extracted via the
	# Pulumi RPC protocol, not a CLI flag. Use pulumi package get-schema for that.
	pulumi package get-schema ./bin/$(BINARY) > schema.json

gen-sdk: build
	# Point gen-sdk directly at the binary; Pulumi starts it, calls GetSchema via
	# gRPC, and generates the SDK. --out sdk causes Pulumi to create sdk/nodejs/
	# and sdk/go/ subdirectories automatically — one call per language.
	pulumi package gen-sdk --language nodejs ./bin/$(BINARY) --out sdk
	pulumi package gen-sdk --language go     ./bin/$(BINARY) --out sdk
