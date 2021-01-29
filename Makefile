GO_FILES := $(shell \
	find . '(' -path '*/.*' -o -path './vendor' ')' -prune \
	-o -name '*.go' -print | cut -b3-)

.PHONY: build
build:
	go build ./...

.PHONY: install
install:
	go mod download

.PHONY: test
test:
	go test -race ./...

.PHONY: cover
cover:
	go test -coverprofile=cover.out -covermode=atomic -coverpkg=./... ./...
	go tool cover -html=cover.out -o cover.html

get-deps:
	go get -u golang.org/x/lint/golint honnef.co/go/tools/cmd/staticcheck

.PHONY: lint
lint:
	@rm -rf lint.log
	@echo "Checking gofmt"
	@gofmt -d -s $(GO_FILES) 2>&1 | tee lint.log
	@echo "Checking go vet"
	@go vet ./... 2>&1 | tee -a lint.log
	@echo "Checking golint"
	@golint ./... | tee -a lint.log
	@echo "Checking staticcheck"
	@staticcheck ./... 2>&1 |  tee -a lint.log
	@echo "Checking for license headers..."
	@./.build/check_license.sh | tee -a lint.log
	@[ ! -s lint.log ]
