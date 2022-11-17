#
# You can override these e.g. as
#     make test TEST_PKG=./immutable/tests/ TEST_ARG="-v --run TestDeletedKey"
#
TEST_PKG=./...
TEST_ARG=

BUILD_PKGS=./...
BUILD_CMD=go build -o .
INSTALL_CMD=go install

all: build-lint

build:
	$(BUILD_CMD) $(BUILD_PKGS)

build-lint: build lint

test: install
	go test $(TEST_PKG) --timeout 10m --count 1 -failfast $(TEST_ARG)

install:
	$(INSTALL_CMD) $(BUILD_PKGS)

lint:
	golangci-lint run --timeout 5m

.PHONY: all build build-lint test install lint
