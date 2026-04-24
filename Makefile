PROJECTNAME ?= $(shell basename "$(PWD)")
DOCKER_USER=e11it
GO_TAGS ?= nomsgpack
RA_DOCKER_VARIANT ?= public
RA_VERSION ?= $(RA_DOCKER_VARIANT)
SECURITY_GOMODCACHE ?= /tmp/ra-gomodcache
SECURITY_GOCACHE ?= /tmp/ra-gocache
SECURITY_GOTOOLCHAIN ?= go1.26.2
PKGS := $(shell go list ./... | grep -v '/\.gomodcache/' | grep -v '/\.gocache/')
PKG_DIRS := $(shell go list -f '{{.Dir}}' ./... | grep -v '/\.gomodcache/' | grep -v '/\.gocache/')

ifeq ($(RA_DOCKER_VARIANT),company)
DOCKER_GO_TAGS ?= nomsgpack,company
else
DOCKER_GO_TAGS ?= nomsgpack
endif

## docker-modules: Build golang:modules docker
docker-modules:
	@echo "  >  Warm build deps stage from RA.Dockerfile"
	@docker build --target=modules --tag=$(PROJECTNAME)-modules --file=docker/RA.Dockerfile .

## docker-build: Build application docker image
## --no-cache && export BUILDKIT_PROGRESS=plain
docker-build:
	@docker build --build-arg GO_TAGS="$(DOCKER_GO_TAGS)" --tag=$(PROJECTNAME):$(RA_VERSION) --file=docker/RA.Dockerfile .

## docker-up: Run docker-compose application from example dir
docker-up: docker-build
	@RA_VERSION=$(RA_VERSION) docker-compose -f example/docker-compose.yml up -d

## docker-down: Down docker-compose application
docker-down:
	@RA_VERSION=$(RA_VERSION) docker-compose -f example/docker-compose.yml down

## docker-logs: Get docker-compose logs
docker-logs:
	@RA_VERSION=$(RA_VERSION) docker-compose -f example/docker-compose.yml logs -f --tail 100

## docker-publish: publish image to hub.docker.com
docker-publish:
	@docker tag ab:latest $(DOCKER_USER)/ra:latest
	@docker push $(DOCKER_USER)/ra:latest

## docker-clean: clean dangling images with label 'autodelete'
docker-clean:
	@docker rmi $(docker images -q -f "dangling=true" -f "label=autodelete=true")

## go-build: build go application
go-build: go-lint
	@go build -tags="$(GO_TAGS)" -o $(PROJECTNAME) .

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

## openapi-gen: Generate OpenAPI models/embedded spec via oapi-codegen
openapi-gen:
	@go generate ./api/openapi

## qa-fast: Fast local quality checks (pre-commit profile)
qa-fast:
	@echo "==> go fmt/vet/test/build"
	@go fmt $(PKGS)
	@go vet $(PKGS)
	@go test -short $(PKGS)
	@go build $(PKGS)

## qa-security: Full local security checks (pre-push/CI profile)
## gosec: exclude .gomodcache/.gocache (repo-local module caches) — same as default vendor/.git, not auto-excluded.
## Pin version to avoid known panics in older analyzers on large dep graphs.
qa-security:
	@echo "==> gosec"
	@GOTOOLCHAIN=$(SECURITY_GOTOOLCHAIN) GOMODCACHE=$(SECURITY_GOMODCACHE) GOCACHE=$(SECURITY_GOCACHE) \
		go run github.com/securego/gosec/v2/cmd/gosec@v2.25.0 \
		-exclude-dir='\.gomodcache' -exclude-dir='\.gocache' \
		$(PKG_DIRS)
	@echo "==> govulncheck"
	@GOTOOLCHAIN=$(SECURITY_GOTOOLCHAIN) GOMODCACHE=$(SECURITY_GOMODCACHE) GOCACHE=$(SECURITY_GOCACHE) \
		go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 $(PKGS)
	@echo "==> gitleaks (if installed)"
	@if command -v gitleaks >/dev/null 2>&1; then \
		gitleaks detect --source . --redact --no-banner --gitleaks-ignore-path .gitleaksignore; \
	else \
		echo "gitleaks not installed locally; run in CI hook/job"; \
	fi

## scan-fs: Trivy filesystem scan (HIGH,CRITICAL)
scan-fs:
	@trivy fs --severity HIGH,CRITICAL --exit-code 1 .

## scan-image: Trivy image scan (HIGH,CRITICAL)
scan-image: docker-build
	@trivy image --severity HIGH,CRITICAL --exit-code 1 $(PROJECTNAME):$(RA_VERSION)

## dast: OWASP ZAP baseline scan against running target
## DAST_TARGET defaults to nginx entrypoint from example stack
dast:
	@DAST_TARGET=$${DAST_TARGET:-http://localhost:8080}; \
	echo "==> ZAP baseline target: $$DAST_TARGET"; \
	docker run --rm --network host -v "$$(pwd):/zap/wrk" ghcr.io/zaproxy/zaproxy:stable \
		zap-baseline.py -t "$$DAST_TARGET" -I -m 3 -r zap-report.html

.PHONY: help
all: help
help: Makefile
	@echo
	@echo " Choose a command run in "$(PROJECTNAME)":"
	@echo
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
	@echo
