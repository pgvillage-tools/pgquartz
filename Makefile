JOB ?= jobs/jobspec1/job.yml

PGVERSION ?= 18

uname_p := $(shell uname -p) # store the output of the command in a variable

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

$(shell mkdir -p bin )

# CONTAINER_TOOL defines the container tool to be used for building images.
# Be aware that the target commands are only tested with Docker which is
# scaffolded by default. However, you might want to replace it to use other
# tools. (i.e. podman)
CONTAINER_TOOL ?= docker

build_dlv:
	go get github.com/go-delve/delve/cmd/dlv@latest
	go build -o /bin/dlv.$(uname_p) github.com/go-delve/delve/cmd/dlv

# Use the following on m1:
# alias make='/usr/bin/arch -arch arm64 /usr/bin/make'
debug:
	go build -gcflags "all=-N -l" -o ./bin/pgquartz.debug.$(uname_p) ./cmd/pgquartz
	~/go/bin/dlv --headless --listen=:2345 --api-version=2 --accept-multiclient exec ./bin/pgquartz.debug.$(uname_p) -- -c '$(JOB)'

debug_test:
	~/go/bin/dlv --headless --listen=:2345 --api-version=2 --accept-multiclient test ./pkg/git/

run:
	./bin/pgquartz.$(uname_p) -c '$(JOB)'

fmt:
	gofmt -w .
	goimports -w .
	gci write .

.PHONY: test
test:
	go test $$(go list ./... | grep -v github.com/pgvillage-tools/pgquartz/tests) -coverprofile cover.out -coverpkg=./...

.PHONY: install-go-test-coverage
install-go-test-coverage:
	go install github.com/vladopajic/go-test-coverage/v2@latest

.PHONY: check-coverage
check-coverage: install-go-test-coverage test
	${GOBIN}/go-test-coverage --config=./.testcoverage.yaml

.PHONY: e2e-test
e2e-test:
	cd ./tests/smoke && PGVERSION=$(PGVERSION) go test -count=1 -v ./...

