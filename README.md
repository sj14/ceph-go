# ceph-go

`ceph-go` provides explicit Go clients for Ceph APIs. The clients keep their
transports separate because authentication, authorization, errors, and response
semantics differ between APIs.

## Clients

- [`mgr`](mgr/README.md) calls Ceph Dashboard APIs exposed by Ceph Manager and
  authenticates with a Dashboard JWT.
- [`rgw`](rgw/README.md) calls the RGW Admin Ops API directly and signs requests
  with AWS Signature Version 4.

Each client README contains usage examples and the complete checklist of
implemented and missing endpoints.

## Installation

```shell
go get github.com/sj14/ceph-go
```

Import the package for the API you want to use:

```go
import (
	"github.com/sj14/ceph-go/mgr"
	"github.com/sj14/ceph-go/rgw"
)
```

## Integration tests

A purpose-built, ephemeral Ceph image with MON, MGR Dashboard, an in-memory
OSD, and RGW lives in [`test/ceph`](test/ceph/README.md). It is substantially
smaller than the general-purpose official image and runs real integration tests
for both clients.
