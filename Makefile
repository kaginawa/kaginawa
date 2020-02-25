.DEFAULT_GOAL := help

.PHONY: setup
setup: ## Resolve dependencies using Go Modules
	GO111MODULE=on go mod download

.PHONY: clean
clean: ## Remove build artifact directory
	-rm -rfv build

.PHONY: test
test: ## Tests all code
	GO111MODULE=on go test -cover -race ./...

.PHONY: lint
lint: ## Runs static code analysis
	command -v golint >/dev/null 2>&1 || { GO111MODULE=on go get -u golang.org/x/lint/golint; }
	go list ./... | xargs -L1 golint -set_exit_status

.PHONY: run
run: ## Run agent without build artifact generation
	GO111MODULE=on go run . -d

.PHONY: build
build: ## Build executable binaries for local execution
	GO111MODULE=on go build -ldflags "-s -w" -o build/kaginawa .

.PHONY: build-all
build-all: build ## Build executable binaries for all supported OSs and architectures
	GO111MODULE=on GOOS=windows GOARCH=amd64 go build -ldflags "-s -w -X main.ver=`git describe --tags`" -o build/kaginawa.exe .
	GO111MODULE=on GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w -X main.ver=`git describe --tags`" -o build/kaginawa.macos .
	GO111MODULE=on GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -X main.ver=`git describe --tags`" -o build/kaginawa.linux-x64 .
	GO111MODULE=on GOOS=linux GOARCH=arm GOARM=6 go build -ldflags "-s -w -X main.ver=`git describe --tags`" -o build/kaginawa.linux-arm6 .
	GO111MODULE=on GOOS=linux GOARCH=arm GOARM=7 go build -ldflags "-s -w -X main.ver=`git describe --tags`" -o build/kaginawa.linux-arm7 .
	GO111MODULE=on GOOS=linux GOARCH=arm64 go build -ldflags "-s -w -X main.ver=`git describe --tags`" -o build/kaginawa.linux-arm8 .
	zip -jmq9 build/kaginawa.exe.zip build/kaginawa.exe
	bzip2 -f build/kaginawa.macos
	bzip2 -f build/kaginawa.linux-x64
	bzip2 -f build/kaginawa.linux-arm6
	bzip2 -f build/kaginawa.linux-arm7
	bzip2 -f build/kaginawa.linux-arm8
	git describe --tags

.PHONY: count
count-go: ## Count number of lines of all go codes
	find . -name "*.go" -type f | xargs wc -l | tail -n 1

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
