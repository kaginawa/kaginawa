.DEFAULT_GOAL := help

.PHONY: setup
setup: ## Resolve dependencies using Go Modules
	go mod download

.PHONY: clean
clean: ## Remove build artifact directory
	-rm -rfv build

.PHONY: test
test: ## Tests all code
	go test -cover -race ./...

.PHONY: lint
lint: ## Runs static code analysis
	command -v golint >/dev/null 2>&1 || { go get golang.org/x/lint/golint; }
	${GOPATH}/bin/golint -set_exit_status ./...
	command -v errcheck >/dev/null 2>&1 || { go get github.com/kisielk/errcheck; }
	${GOPATH}/bin/errcheck ./...

.PHONY: run
run: ## Run agent without build artifact generation
	go run . -d

.PHONY: build
build: ## Build executable binaries for local execution
	go build -ldflags "-s -w" -o build/kaginawa .

.PHONY: build-all
build-all: build ## Build executable binaries for all supported OSs and architectures
	$(eval VER := $(shell git describe --tags))
	GOOS=windows GOARCH=amd64 go build -ldflags "-s -w -X main.ver=$(VER)" -o build/kaginawa.exe .
	GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w -X main.ver=$(VER)" -o build/kaginawa.macos-x64 .
	GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w -X main.ver=$(VER)" -o build/kaginawa.macos-arm64 .
	GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -X main.ver=$(VER)" -o build/kaginawa.linux-x64 .
	GOOS=linux GOARCH=arm GOARM=5 go build -ldflags "-s -w -X main.ver=$(VER)" -o build/kaginawa.linux-arm5 .
	GOOS=linux GOARCH=arm GOARM=6 go build -ldflags "-s -w -X main.ver=$(VER)" -o build/kaginawa.linux-arm6 .
	GOOS=linux GOARCH=arm GOARM=7 go build -ldflags "-s -w -X main.ver=$(VER)" -o build/kaginawa.linux-arm7 .
	GOOS=linux GOARCH=arm64 go build -ldflags "-s -w -X main.ver=$(VER)" -o build/kaginawa.linux-arm8 .
	GOOS=linux GOARCH=riscv64 go build -ldflags "-s -w -X main.ver=$(VER)" -o build/kaginawa.linux-riscv64 .
	zip -jmq9 build/kaginawa.exe.zip build/kaginawa.exe
	bzip2 -f build/kaginawa.macos-x64
	bzip2 -f build/kaginawa.macos-arm64
	bzip2 -f build/kaginawa.linux-x64
	bzip2 -f build/kaginawa.linux-arm5
	bzip2 -f build/kaginawa.linux-arm6
	bzip2 -f build/kaginawa.linux-arm7
	bzip2 -f build/kaginawa.linux-arm8
	bzip2 -f build/kaginawa.linux-riscv64
	cd build; shasum -a 256 kaginawa.exe.zip > kaginawa.exe.zip.sha256
	cd build; shasum -a 256 kaginawa.macos-x64.bz2 > kaginawa.macos-x64.bz2.sha256
	cd build; shasum -a 256 kaginawa.macos-arm64.bz2 > kaginawa.macos-arm64.bz2.sha256
	cd build; shasum -a 256 kaginawa.linux-x64.bz2 > kaginawa.linux-x64.bz2.sha256
	cd build; shasum -a 256 kaginawa.linux-arm5.bz2 > kaginawa.linux-arm5.bz2.sha256
	cd build; shasum -a 256 kaginawa.linux-arm6.bz2 > kaginawa.linux-arm6.bz2.sha256
	cd build; shasum -a 256 kaginawa.linux-arm7.bz2 > kaginawa.linux-arm7.bz2.sha256
	cd build; shasum -a 256 kaginawa.linux-arm8.bz2 > kaginawa.linux-arm8.bz2.sha256
	cd build; shasum -a 256 kaginawa.linux-riscv64.bz2 > kaginawa.linux-riscv64.bz2.sha256
	echo $(VER) > build/LATEST

.PHONY: count-go
count-go: ## Count number of lines of all go codes
	find . -name "*.go" -type f | xargs wc -l | tail -n 1

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
