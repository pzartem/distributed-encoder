CMD =
ARGS =
DEPLOY_ROOT = deployment

include local.env

help: ## [GLOBAL] Print this help dialog
	@awk -F ':|##' '/^[^\t].+?:.*?##/ {\
		printf "\033[36m%-30s\033[0m %s\n", $$1, $$NF \
	}' $(MAKEFILE_LIST)

build: ## Build the file
	@go build -o bin/$(CMD) cmd/$(CMD)/*.go

run: build
	@./bin/$(CMD) $(ARGS)

TESTING_OPTS =
test:
	go test $(TESTING_OPTS) ./...

lint:
	golangci-lint run

docker-build:
	docker build \
		--build-arg cmd=$(CMD) \
		-t $(CMD) \
		-f $(DEPLOY_ROOT)/Dockerfile .

compose-build:
	docker-compose build

compose-run:
	docker-compose up --scale worker=4

.PHONY: build run test lint docker-build

