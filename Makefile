PROJECTNAME ?= $(shell basename "$(PWD)")
DOCKER_USER=e11it

## docker-modules: Build golang:modules docker
docker-modules:
	@echo "  >  Start building golang:modules"
	@docker build --tag=golang:modules --file=docker/Modules.Dockerfile .

## docker-build: Build application docker image
## --no-cache && export BUILDKIT_PROGRESS=plain
docker-build: docker-modules
	@docker build --tag=$(PROJECTNAME) --file=docker/RA.Dockerfile .

## docker-up: Run docker-compose application from example dir
docker-up: docker-build
	@docker-compose -f example/docker-compose.yml up -d

## docker-down: Down docker-compose application
docker-down:
	@docker-compose -f example/docker-compose.yml down

## docker-logs: Get docker-compose logs
docker-logs:
	@docker-compose -f example/docker-compose.yml logs -f --tail 100

## docker-publish: publish image to hub.docker.com
docker-publish:
	@docker tag ab:latest $(DOCKER_USER)/ra:latest
	@docker push $(DOCKER_USER)/ra:latest

## docker-clean: clean dangling images with label 'autodelete'
docker-clean:
	@docker rmi $(docker images -q -f "dangling=true" -f "label=autodelete=true")

## go-build: build go application
go-build: go-lint
	@go build -o $(PROJECTNAME) .

## download: required go modules
go-download:
	@echo "  >  Download modules"
	@go mod download

go-lint-install:
	#@mkdir bin
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.50.1

go-lint:
	./bin/golangci-lint version
	#./bin/golangci-lint run

.PHONY: help
all: help
help: Makefile
	@echo
	@echo " Choose a command run in "$(PROJECTNAME)":"
	@echo
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
	@echo
