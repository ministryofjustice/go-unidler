# go-unidler
Unidle idled tools

This is a rewrite of [unidler](https://github.com/ministryofjustice/analytics-platform-unidler) in Go.


## Dependencies

Dependencies are managed using [Godep](https://github.com/tools/godep).

This is used instead of the new official [dep](https://github.com/golang/dep) because [`k8s.io/client-go/` doesn't support dep yet](https://github.com/kubernetes/client-go/blob/master/INSTALL.md).

New version of Go should already use the dependences vendored in `/vendor`.

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
