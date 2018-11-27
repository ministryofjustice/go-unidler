# See: https://www.cloudreach.com/blog/containerize-this-golang-dockerfiles/
FROM golang:1.11-alpine AS builder

RUN apk update \
    && apk add --no-cache \
      ca-certificates \
      git \
    && rm -rf /var/cache/apk/*

WORKDIR /go/src/github.com/ministryofjustice/go-unidler

COPY app.go k8s.go main.go sse.go go.mod ./

# update vendored dependencies
#RUN CGO_ENABLED=0 GOOS=linux GO111MODULE=on go mod vendor

# NOTE: statically compiled as final image is based on "scratch"
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o go-unidler .

#FROM builder AS test
#RUN CGO_ENABLED=0 GOOS=linux go test

FROM scratch
WORKDIR /bin
COPY templates templates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/src/github.com/ministryofjustice/go-unidler/go-unidler .
CMD ["/bin/go-unidler"]
