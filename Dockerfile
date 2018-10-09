# See: https://www.cloudreach.com/blog/containerize-this-golang-dockerfiles/
FROM golang:1.11-alpine AS builder

WORKDIR /go/src/github.com/ministryofjustice/go-unidler

ADD . .

# NOTE: statically compiled as final image is based on "scratch"
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o go-unidler .

FROM scratch
WORKDIR /bin
COPY --from=builder /go/src/github.com/ministryofjustice/go-unidler/go-unidler .
CMD ["/bin/go-unidler"]
