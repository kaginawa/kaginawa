.DEFAULT_GOAL := help
DEPLOY_BUCKET := "xxx"

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
	GO111MODULE=on go run .

.PHONY: build
build: ## Build executable binaries for local execution
	GO111MODULE=on go build -o build/kaginawa .

.PHONY: build-all
build-all: build ## Build executable binaries for all supported OSs and architectures
	GO111MODULE=on GOOS=windows GOARCH=amd64 go build -ldflags "-X main.ver=`git describe --tags`" -o build/kaginawa.exe .
	GO111MODULE=on GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.ver=`git describe --tags`" -o build/kaginawa.macos .
	GO111MODULE=on GOOS=linux GOARCH=amd64 go build -ldflags "-X main.ver=`git describe --tags`" -o build/kaginawa.linux-x64 .
	GO111MODULE=on GOOS=linux GOARCH=arm GOARM=7 go build -ldflags "-X main.ver=`git describe --tags`" -o build/kaginawa.linux-arm .

.PHONY: deploy
deploy: build-all ## Release cross-compiled binary into snapshots directory of the S3.
	tar czvf build/kaginawa.exe.tar.gz -C build kaginawa.exe
	tar czvf build/kaginawa.macos.tar.gz -C build kaginawa.macos
	tar czvf build/kaginawa.linux-x64.tar.gz -C build kaginawa.linux-x64
	tar czvf build/kaginawa.linux-arm.tar.gz -C build kaginawa.linux-arm
	aws s3 cp build/kaginawa.exe.tar.gz s3://$(DEPLOY_BUCKET)/snapshots/kaginawa_`git describe --tags`.exe.tar.gz
	aws s3 cp build/kaginawa.macos.tar.gz s3://$(DEPLOY_BUCKET)/snapshots/kaginawa_`git describe --tags`.macos.tar.gz
	aws s3 cp build/kaginawa.linux-x64.tar.gz s3://$(DEPLOY_BUCKET)/snapshots/kaginawa_`git describe --tags`.linux-x64.tar.gz
	aws s3 cp build/kaginawa.linux-arm.tar.gz s3://$(DEPLOY_BUCKET)/snapshots/kaginawa_`git describe --tags`.linux-arm.tar.gz

.PHONY: count
count-go: ## Count number of lines of all go codes
	find . -name "*.go" -type f | xargs wc -l | tail -n 1

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
