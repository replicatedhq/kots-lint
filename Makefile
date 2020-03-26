PROJECT_NAME ?= kots-lint
BUILDTAGS = containers_image_ostree_stub exclude_graphdriver_devicemapper exclude_graphdriver_btrfs containers_image_openpgp

.PHONY: clean
clean:
	rm -f ./bin/kots-lint
	
.PHONY: run
run:
	./bin/kots-lint

.PHONY: build
build:
	mkdir -p bin
	go build \
		-tags "$(BUILDTAGS)" \
		-o ./bin/kots-lint \
		.

.PHONY: test
test:
	go test -v `go list ./...` -tags "$(BUILDTAGS)"

.PHONY: dev
dev:
	docker build -f Dockerfile.dev -t ${PROJECT_NAME}-dev .
	docker run -p 8082:8082 ${PROJECT_NAME}-dev

.PHONY: prod
prod:
	docker build -t ${PROJECT_NAME} .
	docker run -p 8082:8082 ${PROJECT_NAME}