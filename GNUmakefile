default: testacc

# Run acceptance tests
.PHONY: testacc
testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m
doc:
	go generate ./...
build: 
	@go build
install_linux: build
	@mkdir -p /home/alexd/go/bin/plugins/$(provider_macos_path)
	@mv fluent-health-terraform-typesense /home/alexd/go/bin/plugins/terraform-provider-typesense