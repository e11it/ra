PROJECTNAME=$(shell basename "$(PWD)")

## docker-modules: Build golang:modules docker
docker-modules:
	@echo "  >  Start building golang:modules"
	@docker build --tag=golang:modules --file=docker/Modules.Dockerfile .

## docker-build: Build application docker image
docker-build: docker-modules
	@docker build --tag=$(PROJECTNAME) --file=docker/RAB.Dockerfile .

## docker-up: Run docker-compose application from example dir
docker-up: docker-build
	@docker-compose -f example/docker-compose.yml up -d

## docker-down: Down docker-compose application
docker-down:
	@docker-compose -f example/docker-compose.yml down

## docker-logs: Get docker-compose logs
docker-logs:
	@docker-compose -f example/docker-compose.yml logs -f --tail 100

## go-build: build go application
go-build: go-lint
	@go build -o $(PROJECTNAME) .

## download: required go modules
go-download:
	@echo "  >  Download modules"
	@go mod download

go-lint-install:
	#@mkdir bin
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.24.0

go-lint:
	./bin/golangci-lint version
	./bin/golangci-lint run

.PHONY: help
all: help
help: Makefile
	@echo
	@echo " Choose a command run in "$(PROJECTNAME)":"
	@echo
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
	@echo