# go-unidler
Unidle idled tools

This is a rewrite of [unidler](https://github.com/ministryofjustice/analytics-platform-unidler) in Go.

This is performing the reverse operation of the [idler](https://github.com/ministryofjustice/analytics-platform-idler).


## Configuration
The application doesn't require any configuration to work.
You can set the following environment variables if you need to:

- `PORT` (default `":8080"`), port on which the server listen
- `INGRESS_CLASS_NAME` (default: `"nginx"`), Ingress class name. This
  depends on your kubernetes cluster and it will be used as value of the
  `kubernetes.io/ingress.class` annotation on unidled applications (this
  annotation is set to `disabled` when they're unidled)

In addition to the above optional configuration the server will try to load
the kubernetes configuration from in-cluster (this is the case when running
the server within a k8s cluster) and fallback to load it from `~/.kube/config`
when this fails. If that fails as well the server will not start.


## Endpoints

### `/` (all)
All request go to the main handler, which performs the following operations:

1. This will get the `Host` request header
2. Find the Ingress resource for that host
3. Find the Deployment resource for that ingress (currently based on name)
4. Set this Deployment's replicas to `1`
5. Wait for this Deployment to have an available replica
6. Once the application is undled and can potentially respond to traffic
   its Ingress resource is enabled by setting the `kubernetes.io/ingress.class` annotation to the value in `INGRESS_CLASS_NAME`
7. The host is also removed from the `unidler` Ingress' `spec.rules`. This
   means the unidler is not responsible for the traffic to the application host
   anymore.
8. Finally the following metadata added by the idler to the Deployment resource is removed:
   - `mojanalytics.xyz/idled` label
   - `mojanalytics.xyz/idled-at` annotation

### `/healthz` (healthcheck)
This will responde with a `200 OK` and a brief text body.
It's used by kubernetes (or wathever) to check that the server is still
responding.


## Dependencies

Dependencies are managed using [Godep](https://github.com/tools/godep).

This is used instead of the new official [dep](https://github.com/golang/dep) because [`k8s.io/client-go/` doesn't support dep yet](https://github.com/kubernetes/client-go/blob/master/INSTALL.md).

Dependences are vendored in the `/vendor` which is checked in Git.

Recent versions of Go should already use the dependences vendored in `/vendor`.

### Install Godep
As per instructions [here](https://github.com/tools/godep#install):

```bash
$ go get github.com/tools/godep
```

### Add a new dependency

As per instructions [here](https://github.com/tools/godep#add-a-dependency)

1. `$ go get foo/bar`
2. Edit your code to import foo/bar
3. `$ godep save ./...`

### Upgrade a dependency

As per instructions [here](https://github.com/tools/godep#add-a-dependency)

1. `$ go get -u foo/bar`
2. `$ godep update foo/bar`


## Docker image
The [`Dockerfile`](/) uses 2 stages one for building and the final image.

### builder stage

### final stage
The actual image running `go-unidler` is just scratch with the binary compiled
statically (`-ldflags '-extldflags "-static"'`) to keep the docker image to the minimum.

See this article on containerising Go application: https://www.cloudreach.com/blog/containerize-this-golang-dockerfiles/
