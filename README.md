# go-unidler
Unidle idled tools

This is performing the reverse operation of the [idler](https://github.com/ministryofjustice/analytics-platform-idler).

This is a rewrite of [unidler](https://github.com/ministryofjustice/analytics-platform-unidler) in Go.

## Usage

A Makefile is provided to enable easily building, testing and running the
unidler.

### `make help`
Show list of available "commands" (targets)

### `make run`
Compiles and runs the unidler on `http://localhost:8080` (or the `$PORT`
specified)

### `make test`
Compiles the test code and runs it

### `make static` (default)
Compiles the unidler to a static-linked binary

### `make docker-image`
Builds a docker image as defined in Dockerfile

### `make docker-run`
Builds and runs the unidler in a docker container


## Configuration
The application doesn't require much configuration to work.
`$HOME` is the only required setting:

| Env variable         | Default |  Details |
| -------------------- | ------- | -------- |
| `HOME`               |         | Home where .kube config would be when not using in-cluster config (e.g. when running locally) |
| `PORT`               | `:8080` | port on which the server listen |

The server will try to load the kubernetes configuration from in-cluster first
(this is the case when running the server within a k8s cluster) and fallback
to load it from `~/.kube/config` when this fails.

If that fails as well the server will not start.


## Endpoints

### `/`
This endpoint will render and send the unidling page.
This page is mostly responsible to show progress to
the user and any error which occurs.

The frontend uses [`EventSource`](https://developer.mozilla.org/en-US/docs/Web/API/EventSource) which will open a persisten connection to the `/events` endpoint.
This is how the user (client) receives the updates on the uniding process
from the unidler (server).

### `/events/` (Server Sent Events)
Requests to `/events/`  will trigger the unidling process.

Roughly, the unidler will perform the following operations:
- set the Deployment's replicas back to whatever number of replicas there
  were before the app was unidled (or `1` if that can't be determined)
- wait for the Deployment to have available replicas
- remove metadata information (label/annotation) that are present when an
  app is idled.
  - this is important at the moment because the idler will assume
  an app is already idled if it finds this metadata
- Update the app service to point to the app's pods
  - when an app is idled the service will direct traffic to the unidler

Requests to this endpoint will be held open, and Server Side Events with
progress updates will be pushed back to the browser as the Deployment
corresponding to the `Host` header is being unidled.

### `/healthz` (healthcheck)
This will responde with a `200 OK` and a brief text body.
It's used by kubernetes (or wathever) to check that the server is still
responding.


## Dependencies

Dependencies are managed using [Go Modules](https://github.com/golang/go/wiki/Modules).

Dependences are vendored in the `/vendor` which is checked in Git.


### Add a new dependency

1. `$ go get foo/bar`
2. Edit your code to import foo/bar

### Upgrade a dependency

As per instructions [here](https://github.com/golang/go/wiki/Modules#how-to-upgrade-and-downgrade-dependencies)

1. `$ go get foo/bar`

This will upgrade to the latest version of `foo/bar` with a semver tag.
Alternatively, `go get foo/bar@v1.2.3` will get a specific version.

## Docker image
The [`Dockerfile`](/) uses 2 stages one for building and the final image.

### builder stage

### final stage
The actual image running `go-unidler` is just scratch with the binary compiled
statically (`-ldflags '-extldflags "-static"'`) to keep the docker image to the minimum.

See this article on containerising Go application: https://www.cloudreach.com/blog/containerize-this-golang-dockerfiles/
