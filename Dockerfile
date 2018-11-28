# See: https://www.cloudreach.com/blog/containerize-this-golang-dockerfiles/
FROM golang:1.11-alpine AS builder

RUN apk update \
    && apk add --no-cache \
      ca-certificates \
      git \
	  make \
    && rm -rf /var/cache/apk/*

WORKDIR /go/src/github.com/ministryofjustice/go-unidler

COPY vendor vendor
COPY Makefile app.go k8s.go main.go sse.go go.mod ./

# update vendored dependencies
#RUN make vendored-packages

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
