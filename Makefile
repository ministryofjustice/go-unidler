BIN := go-unidler
PKG := github.com/ministryofjustice/go-unidler
REGISTRY := quay.io/mojanalytics/go-unidler
DOCKER_IMAGE := go-unidler
DOCKERFILE := Dockerfile
PORT := 8000


default: build

build:
	docker build ${DOCKER_BUILD_ARGS} -t "$(DOCKER_IMAGE)" -f "$(DOCKERFILE)" .

run: build
	docker run -e PORT=":${PORT}" -v ${HOME}/.kube:/.kube -p ${PORT}:${PORT} "${DOCKER_IMAGE}"
