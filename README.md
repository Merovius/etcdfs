# etcdfs - Read-only FUSE filesystem to access etcd

etcdfs lets you mount a subdirectory of an etcd cluster via FUSE. It only
provides read-only access - you should use etcdctl to set keys. The reason for
this is on the one hand simplicity and on the other hand the fact that a
filesystem API makes it difficult to map the consistency properties of etcd.

The intended usage is to bootstrap
[confd](https://github.com/kelseyhightower/confd) templates via etcd too. You
can also (if you don't want to use confd) use it to store configuration in etcd
for programs that don't support an etcd API for configuration.

# Installation

Currently, etcdfs is not packaged for any distribution. There is no immediate
plan to do this, though there will be a docker container soon. Until then, you
can install it using a recent go toolchain. Assuming, you have set that up
correctly and set your GOPATH accordingly, it's a simply `go get
merovius.de/etcdfs`.

# Running

Command line usage is

```
Usage:
	./etcdfs [flags…] [<subdir>] <mountpoint>

Flags:
  -allow_other
    	Allow other users to access this filesystem
  -allow_root
    	Allow root to access this filesystem
  -debug
    	Enable debugging
```

so, e.g. `etcdfs /etcd`. If `<subdir>` is given, it names the subdirectory of
the etcd keyspace you want to mount. If your etcd cluster isn't avaliable on
`http://localhost:4001`, you should set `ETCD_ENDPOINTS` to a comma-separated
list of urls that should be used instead.

# Contributing

Please see the [Contribution guideluines](CONTRIBUTING.md) before opening an
issue or creating a pull request.

# License

[Apache 2.0](LICENSE)

```
Copyright 2015 Axel Wagner

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```
