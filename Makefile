PKG := $(shell go list ./... | grep -v vendor)
TEST := $(shell go list ./... |grep -v vendor)

ifndef JIRA_URL
JIRA_URL := http://127.0.0.1:2990/jira
endif

ifndef JIRA_USER
JIRA_USER := admin
endif

ifndef JIRA_PASSWORD
JIRA_PASSWORD := admin
endif


.PHONY: test

.DEFAULT_GOAL := help
help: ## List targets & descriptions
	@cat Makefile* | grep -E '^[a-zA-Z_-]+:.*?## .*$$' | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

deps: ## Download dependencies
	go mod download

build: ## Build
	go build .

install:
	cp ./terraform-provider-jira /Users/jess.verner/.terraform.d/plugins/terraform.example.com/local/jira/1.0.0/darwin_arm64

release: ## Build the go binaries for various platform
	./scripts/release.sh

test: ## Run tests
	TF_ACC=1 JIRA_URL="$(JIRA_URL)" JIRA_PASSWORD="$(JIRA_PASSWORD)" JIRA_USER="$(JIRA_USER)" go test -v $(TEST)
