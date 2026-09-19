# rgw-go

`rgw-go` is a Go client for RGW management through the Ceph Dashboard REST API. It currently implements bucket creation and retrieval.

The implementation was verified on 2026-09-19 against the latest stable Ceph release, `v20.2.4` (tag commit `7f793731f1b3`), rather than the generated API documentation. Stability was confirmed through `doc/releases/releases.yml` and the tag's `src/ceph_release` file:

- [Stable release metadata](https://github.com/ceph/ceph/blob/main/doc/releases/releases.yml)
- [`RgwBucket.create`](https://github.com/ceph/ceph/blob/v20.2.4/src/pybind/mgr/dashboard/controllers/rgw.py)
- [`RgwBucket.get`](https://github.com/ceph/ceph/blob/v20.2.4/src/pybind/mgr/dashboard/controllers/rgw.py)
- [Ceph Dashboard's RGW bucket frontend client](https://github.com/ceph/ceph/blob/v20.2.4/src/pybind/mgr/dashboard/frontend/src/app/shared/api/rgw-bucket.service.ts)
- [`RgwClient.create_bucket`](https://github.com/ceph/ceph/blob/v20.2.4/src/pybind/mgr/dashboard/services/rgw_client.py)
- [REST routing and status mapping](https://github.com/ceph/ceph/blob/v20.2.4/src/pybind/mgr/dashboard/controllers/_rest_controller.py)

## Usage

```go
client, err := rgw.NewClient(
    "https://ceph-dashboard.example",
    rgw.WithBearerToken(os.Getenv("CEPH_DASHBOARD_TOKEN")),
)
if err != nil {
    log.Fatal(err)
}

err = client.CreateBucket(ctx, rgw.CreateBucketRequest{
    Name: "backups",
    UID:  "alice",
})
if err != nil {
    log.Fatal(err)
}

bucket, err := client.GetBucket(ctx, rgw.GetBucketRequest{
    Name: "backups",
})
if err != nil {
    log.Fatal(err)
}
log.Printf("bucket %s is owned by %s", bucket.Name, bucket.Owner)
```

The token is the JWT returned by Ceph Dashboard's `POST /api/auth` endpoint. A preconfigured `http.Client` can be supplied with `WithHTTPClient`, for example to set timeouts or a private-CA transport.

## Integration tests

A purpose-built, ephemeral Ceph image with MON, MGR Dashboard, an in-memory
OSD, and RGW lives in [`test/ceph`](test/ceph/README.md). It is substantially
smaller than the general-purpose official image and includes a smoke test for
the Create/Get Bucket flow.
