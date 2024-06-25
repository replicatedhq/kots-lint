export GO111MODULE=on

PROJECT_NAME ?= kots-lint
VERSION_PACKAGE = github.com/replicatedhq/kots-lint/pkg/version
VERSION ?= local
DATE=`date -u +"%Y-%m-%dT%H:%M:%SZ"`
BUILDTAGS = containers_image_ostree_stub exclude_graphdriver_devicemapper exclude_graphdriver_btrfs containers_image_openpgp

GIT_TREE = $(shell git rev-parse --is-inside-work-tree 2>/dev/null)
ifneq "$(GIT_TREE)" ""
define GIT_UPDATE_INDEX_CMD
git update-index --assume-unchanged
endef
define GIT_SHA
`git rev-parse HEAD`
endef
else
define GIT_UPDATE_INDEX_CMD
echo "Not a git repo, skipping git update-index"
endef
define GIT_SHA
""
endef
endif

define LDFLAGS
-ldflags "\
	-X ${VERSION_PACKAGE}.version=${VERSION} \
	-X ${VERSION_PACKAGE}.gitSHA=${VERSION} \
	-X ${VERSION_PACKAGE}.buildTime=${DATE} \
"
endef

.PHONY: clean
clean:
	rm -f ./bin/kots-lint
	
.PHONY: run
run:
	./bin/kots-lint

.PHONY: build
build:
	mkdir -p bin
	CGO_ENABLED=0 go build \
		${LDFLAGS} \
		-tags "$(BUILDTAGS)" \
		-o ./bin/kots-lint \
		.

.PHONY: test
test:
	go test -v ./pkg/... -tags "$(BUILDTAGS)"

.PHONY: example
example:
	go run -tags "$(BUILDTAGS)" ./example/main.go

.PHONY: debug-build
debug-build:
	go build \
		${LDFLAGS} \
		-tags "$(BUILDTAGS)" \
		-o ./bin/kots-lint-debug \
		.

.PHONY: debug
debug: debug-build
	dlv --listen=:2345 --headless=true --api-version=2 exec ./bin/kots-lint-debug
