PROJECTNAME := $(shell basename "$(PWD)")
PORT := 8000
GO := CGO_ENABLED=0 go


default: static

dependencies:
	@echo " > Checking dependencies..."
	@GO111MODULE=on $(GO) mod vendor

## docker-image: Build docker image.
docker-image:
	@echo " > Building Docker image..."
	docker build ${DOCKER_BUILD_ARGS} -t "$(PROJECTNAME)" .

## docker-run: Run in docker.
docker-run: docker-image
	@echo " > Running Docker container..."
	docker run -e PORT=":${PORT}" -v ${HOME}/.kube:/.kube -p ${PORT}:${PORT} "${PROJECTNAME}"

## static: Build static binary.
static:
	@echo " > Building binary..."
	@${GO} build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o ${PROJECTNAME} .

## run: Run the unidler.
run: static
	@echo " > Starting unidler"
	@./$(PROJECTNAME)

## test: Run unit tests.
test: dependencies
	@echo " > Testing..."
	@${GO} test -v

## race-check: Check for race conditions
race-check:
	@echo " > Checking for race conditions..."
	@go run -race app.go jsonpatch.go k8s.go main.go

# clean: Clean build files. Runs `go clean` internally.
clean:
	@echo " > Cleaning build cache"
	@$(GO) clean

.PHONY: help
all: help
help: Makefile
	@echo
	@echo " Commands in "$(PROJECTNAME)":"
	@echo
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
	@echo
