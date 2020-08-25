CMD =
ARGS =
DEPLOY_ROOT = deployment

include local.env

help: ## [GLOBAL] Print this help dialog
	@awk -F ':|##' '/^[^\t].+?:.*?##/ {\
		printf "\033[36m%-30s\033[0m %s\n", $$1, $$NF \
	}' $(MAKEFILE_LIST)

build: ## Build the go binary
	@go build -o bin/$(CMD) cmd/$(CMD)/*.go

run: build ## runs go binary
	@./bin/$(CMD) $(ARGS)

TESTING_OPTS =
test: ## runs tests
	go test $(TESTING_OPTS) ./...

lint: ## runs linter
	golangci-lint run

docker-build: vendor ## runs container build
	docker build \
		--build-arg cmd=$(CMD) \
		-t $(CMD) \
		-f $(DEPLOY_ROOT)/Dockerfile .

vendor: ## runs vendor
	go mod vendor

compose-build: vendor ## build compose images
	docker-compose build

compose-run: ## starts compose with workes scale 4
	docker-compose up --scale worker=4

.PHONY: build run test lint docker-build vendor compose-build compose-run

