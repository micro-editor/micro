.PHONY: runtime build generate build-quick

VERSION = $(shell GOOS=$(shell go env GOHOSTOS) GOARCH=$(shell go env GOHOSTARCH) \
	go run tools/build-version.go)
HASH = $(shell git rev-parse --short HEAD)
DATE = $(shell GOOS=$(shell go env GOHOSTOS) GOARCH=$(shell go env GOHOSTARCH) \
	go run tools/build-date.go)
GOBIN ?= $(shell go env GOPATH)/bin
GOVARS = -X github.com/micro-editor/micro/v2/internal/util.Version=$(VERSION) -X github.com/micro-editor/micro/v2/internal/util.CommitHash=$(HASH) -X 'github.com/micro-editor/micro/v2/internal/util.CompileDate=$(DATE)'
DEBUGVAR = -X github.com/micro-editor/micro/v2/internal/util.Debug=ON
VSCODE_TESTS_BASE_URL = 'https://raw.githubusercontent.com/microsoft/vscode/e6a45f4242ebddb7aa9a229f85555e8a3bd987e2/src/vs/editor/test/common/model/'
CGO_ENABLED := $(if $(CGO_ENABLED),$(CGO_ENABLED),0)

ADDITIONAL_GO_LINKER_FLAGS := ""
GOHOSTOS = $(shell go env GOHOSTOS)
ifeq ($(GOHOSTOS), darwin)
	# Native darwin resp. macOS builds need external and dynamic linking
	ADDITIONAL_GO_LINKER_FLAGS += $(shell GOOS=$(GOHOSTOS) \
		GOARCH=$(shell go env GOHOSTARCH) \
		go run tools/info-plist.go "$(shell go env GOOS)" "$(VERSION)")
	CGO_ENABLED = 1
endif

build: generate build-quick

build-quick:
	CGO_ENABLED=$(CGO_ENABLED) go build -trimpath -ldflags "-s -w $(GOVARS) $(ADDITIONAL_GO_LINKER_FLAGS)" ./cmd/micro

build-dbg:
	CGO_ENABLED=$(CGO_ENABLED) go build -trimpath -ldflags "$(ADDITIONAL_GO_LINKER_FLAGS) $(DEBUGVAR)" ./cmd/micro

build-tags: fetch-tags build

build-all: build

install: generate
	go install -ldflags "-s -w $(GOVARS) $(ADDITIONAL_GO_LINKER_FLAGS)" ./cmd/micro

install-all: install

fetch-tags:
	git fetch --tags --force

generate:
	GOOS=$(shell go env GOHOSTOS) GOARCH=$(shell go env GOHOSTARCH) go generate ./runtime
	GOOS=$(shell go env GOHOSTOS) GOARCH=$(shell go env GOHOSTARCH) go generate ./internal/config

testgen:
	mkdir -p tools/vscode-tests
	cd tools/vscode-tests && \
	curl --remote-name-all $(VSCODE_TESTS_BASE_URL){editableTextModelAuto,editableTextModel,model.line}.test.ts
	tsc tools/vscode-tests/*.ts > /dev/null; true
	go run tools/testgen.go tools/vscode-tests/*.js > buffer_generated_test.go
	mv buffer_generated_test.go internal/buffer
	gofmt -w internal/buffer/buffer_generated_test.go

test:
	go test ./internal/...
	go test ./cmd/...
	@test -z "$$(gofmt -l .)" || { echo "gofmt issues:"; gofmt -d -e .; exit 1; }

bench:
	for i in 1 2 3; do \
		go test -bench=. ./internal/...; \
	done > benchmark_results
	benchstat benchmark_results

bench-baseline:
	for i in 1 2 3; do \
		go test -bench=. ./internal/...; \
	done > benchmark_results_baseline

bench-compare:
	for i in 1 2 3; do \
		go test -bench=. ./internal/...; \
	done > benchmark_results
	benchstat -alpha 0.15 benchmark_results_baseline benchmark_results

clean:
	rm -f micro
