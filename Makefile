# The name of Terraform custom provider.
CUSTOM_PROVIDER_NAME ?= terraform-provider-st-ucloud
# The url of Terraform provider.
CUSTOM_PROVIDER_URL ?= example.local/myklst/st-ucloud
OS := $(shell uname)

.PHONY: all

all: macarm linux

macarm:
ifneq ($(OS), Darwin)
	$(info 'skip macarm')
else
	export PROVIDER_LOCAL_PATH='$(CUSTOM_PROVIDER_URL)'
	GOOS=darwin GOARCH=arm64 go install .
	GO_INSTALL_PATH="$$(go env GOPATH)/bin"; \
	HOME_DIR="$$(ls -d ~)"; \
	mkdir -p  $$HOME_DIR/.terraform.d/plugins/$(CUSTOM_PROVIDER_URL)/0.1.0/darwin_arm64/; \
	cp $$GO_INSTALL_PATH/$(CUSTOM_PROVIDER_NAME) $$HOME_DIR/.terraform.d/plugins/$(CUSTOM_PROVIDER_URL)/0.1.0/darwin_arm64/$(CUSTOM_PROVIDER_NAME)
endif

linux:
ifneq ($(OS), Linux)
	$(info 'skip linux')
else
	export PROVIDER_LOCAL_PATH='$(CUSTOM_PROVIDER_URL)'
	GOOS=linux GOARCH=amd64 go install .
	GO_INSTALL_PATH="$$(go env GOPATH)/bin"; \
	HOME_DIR="$$(ls -d ~)"; \
	mkdir -p  $$HOME_DIR/.terraform.d/plugins/$(CUSTOM_PROVIDER_URL)/0.1.0/linux_amd64/; \
	cp $$GO_INSTALL_PATH/$(CUSTOM_PROVIDER_NAME) $$HOME_DIR/.terraform.d/plugins/$(CUSTOM_PROVIDER_URL)/0.1.0/linux_amd64/$(CUSTOM_PROVIDER_NAME)
	unset PROVIDER_LOCAL_PATH
endif

