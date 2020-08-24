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

ffmpeg-crop-multiple:
	mkfifo /tmp/pipe1.yuv && mkfifo /tmp/pipe2.yuv && ffmpeg -i ~/LetinVR_test_1.mp4 -f rawvideo \
		-filter_complex \
		"[0:v]crop=w=1920:h=1080:x=0:y=0[out1];[out1]format=pix_fmts=yuv420p[out1];[0:v]crop=w=1920:h=1080:x=1920:y=1080[out2];[out2]format=pix_fmts=yuv420p[out2]" \
		-map [out1]  /tmp/pipe1.yuv \
		-map [out2]  /tmp/pipe2.yuv

.PHONY: build run test lint docker-build

