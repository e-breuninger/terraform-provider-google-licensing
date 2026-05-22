BINARY        = terraform-provider-google-licensing
VERSION      ?= dev
GOOS         ?= $(shell go env GOOS)
GOARCH       ?= $(shell go env GOARCH)

.PHONY: default build install lint fmt generate test testacc clean

default: fmt lint install generate

build:
	go build -ldflags="-X main.version=$(VERSION)" -o $(BINARY) .

install: build
	@PLUGIN_DIR=~/.terraform.d/plugins/registry.terraform.io/e-breuninger/google-licensing/$(VERSION)/$(GOOS)_$(GOARCH); \
	mkdir -p $$PLUGIN_DIR && \
	cp $(BINARY) $$PLUGIN_DIR/ && \
	echo "Installed to $$PLUGIN_DIR/$(BINARY)"

lint:
	golangci-lint run ./...

fmt:
	gofmt -s -w .

generate:
	cd tools && go generate ./...

test:
	go test ./... -v -timeout 120s

testacc:
	TF_ACC=1 go test ./... -v -timeout 300s -run '^TestAcc'

clean:
	rm -f $(BINARY)
	rm -rf dist/
