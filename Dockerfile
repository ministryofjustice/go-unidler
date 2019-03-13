# See: https://www.cloudreach.com/blog/containerize-this-golang-dockerfiles/

# Builder image stage
FROM golang:1.12-alpine AS builder

RUN apk update \
      && apk add --no-cache \
      ca-certificates \
      git \
      make

WORKDIR /go/src/github.com/ministryofjustice/analytics-platform-go-unidler

COPY vendor/ vendor/
COPY jsonpatch/ jsonpatch/
COPY templates/ templates/
COPY Makefile ./
COPY *.go ./
COPY go.mod ./
COPY go.sum ./

RUN go mod verify
RUN make test

# NOTE: statically compiled as final image is based on "scratch"
RUN make static

# Binary image stage
FROM scratch
WORKDIR /bin
COPY templates templates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/src/github.com/ministryofjustice/analytics-platform-go-unidler/go-unidler .

CMD ["/bin/go-unidler"]
