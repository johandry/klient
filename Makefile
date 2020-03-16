SHELL	:= /bin/bash

# Install richgo from https://github.com/kyoh86/richgo
ifeq (,$(shell which richgo))
	GOTEST := go
else
	GOTEST := richgo
endif

.PHONY: install 
install: fmt test
	go install .

.PHONY: test
test:
	$(GOTEST) test -cover -race -coverprofile=coverage.txt -covermode=atomic -v ./...

.PHONY: fmt
fmt:
	go fmt ./...
	go vet ./...

.PHONY: check-fmt
check-fmt:
	@files=$$(GO111MODULE=off go fmt ./...); \
	if [[ -n $${files} ]]; then echo "Go fmt found errors in the following files:\n$${files}\n"; exit 1; fi
	@go vet ./...