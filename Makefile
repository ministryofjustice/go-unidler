PROJECTNAME := $(shell basename "$(PWD)")

GOBASE := $(shell pwd)
#GOPATH := $(GOBASE)/vendor:$(GOBASE)
GOBIN := $(GOBASE)/bin
GOFILES := $(wildcard *.go)

STDERR := /tmp/.$(PROJECTNAME)-stderr.txt
PID := /tmp/.$(PROJECTNAME).pid

PORT := 8000
GO := CGO_ENABLED=0 go


default: static

dependencies:
	@echo " > Checking dependencies..."
	@$(GO) mod vendor

## docker-image: Build docker image.
docker-image: dependencies
	@echo " > Building Docker image..."
	docker build ${DOCKER_BUILD_ARGS} -t "$(PROJECTNAME)" .

## docker-run: Run in docker.
docker-run: docker-image
	@echo " > Running Docker container..."
	docker run -e PORT=":${PORT}" -v ${HOME}/.kube:/.kube -p ${PORT}:${PORT} "${PROJECTNAME}"

## static: Build static binary.
static: dependencies
	@echo " > Building binary..."
	@${GO} build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o ${PROJECTNAME} .

## run: Run the unidler.
run: static
	@echo " > Starting unidler"
	@$(PROJECTNAME)

## test: Run unit tests.
test: dependencies
	@echo " > Testing..."
	${GO} test

# clean: Clean build files. Runs `go clean` internally.
clean:
	@echo " > Cleaning build cache"
	@$(GO) clean

go-install:
	@$(GO) install $(GOFILES)

.PHONY: help
all: help
help: Makefile
	@echo
	@echo " Commands in "$(PROJECTNAME)":"
	@echo
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
	@echo
