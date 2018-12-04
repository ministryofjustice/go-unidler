# See: https://www.cloudreach.com/blog/containerize-this-golang-dockerfiles/
FROM golang:1.11-alpine AS builder

RUN apk update \
    && apk add --no-cache \
      ca-certificates \
      git \
	  make

WORKDIR /go/src/github.com/ministryofjustice/go-unidler

#COPY vendor vendor
COPY Makefile ./
COPY *.go ./
COPY go.mod ./

ENV GO111MODULE=on

# NOTE: statically compiled as final image is based on "scratch"
RUN make static

#FROM builder AS test
#RUN make test

FROM scratch
WORKDIR /bin
COPY templates templates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/src/github.com/ministryofjustice/go-unidler/go-unidler .
CMD ["/bin/go-unidler"]
