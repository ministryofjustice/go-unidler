# go-unidler
Unidle idled tools

This is a rewrite of [unidler](https://github.com/ministryofjustice/analytics-platform-unidler) in Go.

This is performing the reverse operation of the [idler](https://github.com/ministryofjustice/analytics-platform-idler).

## Usage

A Makefile is provided to enable easily building, testing and running the
unidler.

### `make static`
Compiles the unidler to a static-linked binary

### `make run`
Compiles and runs the unidler on `http://localhost:8000` (or the `$PORT`
specified)

### `make test`
Compiles the test code and runs it

### `make docker-image`
Builds a docker image as defined in Dockerfile

### `make docker-run`
Builds and runs the unidler in a docker container


## Configuration
The application doesn't require any configuration to work.
You can set the following environment variables if you need to:


| Env variable         | Default   | Details |
| -------------------- | --------- | ------- |
| `PORT`               | `:8080` | port on which the server listen |
| `INGRESS_CLASS_NAME` | `nginx` | Ingress class name. This  depends on your kubernetes cluster and it will be used as value of the `kubernetes.io/ingress.class` annotation on unidled applications (this annotation is set to `disabled` when they're idled) |

In addition to the above optional configuration the server will try to load
the kubernetes configuration from in-cluster (this is the case when running
the server within a k8s cluster) and fallback to load it from `~/.kube/config`
when this fails. If that fails as well the server will not start.


## Endpoints

### `/` (all)
Requests to any path except `/events/` and `/healthz` will trigger the unidling
process, which performs the following operations:

1. This will get the `Host` request header
2. Find the Ingress resource for that host
3. Find the Deployment resource for that ingress (currently based on name)
4. Set this Deployment's replicas to `1`
5. Wait for this Deployment to have an available replica
6. Once the application is unidled and can potentially respond to traffic
   its Ingress resource is enabled by setting the `kubernetes.io/ingress.class` annotation to the value in `INGRESS_CLASS_NAME`
7. The host is also removed from the `unidler` Ingress' `spec.rules`. This
   means the unidler is not responsible for the traffic to the application host
   anymore.
8. Finally the following metadata added by the idler to the Deployment resource is removed:
   - `mojanalytics.xyz/idled` label
   - `mojanalytics.xyz/idled-at` annotation

The unidle process is executed asynchronously, and the HTML page returned by
this endpoint shows the status of the process until it completes. If the
Deployment is successfully unidled, the page will automatically redirect to the
unidled app.

### `/events/` (Server Sent Events)
Requests to this endpoint will be held open, and Server Side Events pushed back
to the browser as the Deployment corresponding to the `Host` header is unidled.
If the unidle process is complete, the last status is send (either "Ready", or
an error message).

### `/healthz` (healthcheck)
This will responde with a `200 OK` and a brief text body.
It's used by kubernetes (or wathever) to check that the server is still
responding.


## Dependencies

Dependencies are managed using [Versioned Go](https://github.com/golang/vgo).

Dependences are vendored in the `/vendor` which is checked in Git.

Recent versions of Go should already use the dependences vendored in `/vendor`.

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
