# rgw-go

`rgw-go` is a Go client for RGW management through the Ceph Dashboard REST API. It currently implements bucket creation.

The implementation was verified on 2026-09-19 against the latest Ceph tag, `v21.3.0` (commit `cc6b5e2da077`), rather than the generated API documentation:

- [`RgwBucket.create`](https://github.com/ceph/ceph/blob/v21.3.0/src/pybind/mgr/dashboard/controllers/rgw.py)
- [Ceph Dashboard's RGW bucket frontend client](https://github.com/ceph/ceph/blob/v21.3.0/src/pybind/mgr/dashboard/frontend/src/app/shared/api/rgw-bucket.service.ts)
- [`RgwClient.create_bucket`](https://github.com/ceph/ceph/blob/v21.3.0/src/pybind/mgr/dashboard/services/rgw_client.py)
- [REST routing and status mapping](https://github.com/ceph/ceph/blob/v21.3.0/src/pybind/mgr/dashboard/controllers/_rest_controller.py)

## Usage

```go
client, err := rgw.NewClient(
    "https://ceph-dashboard.example",
    rgw.WithBearerToken(os.Getenv("CEPH_DASHBOARD_TOKEN")),
)
if err != nil {
    log.Fatal(err)
}

response, err := client.CreateBucket(ctx, rgw.CreateBucketRequest{
    Name: "backups",
    UID:  "alice",
})
if err != nil {
    log.Fatal(err)
}
log.Printf("created bucket: HTTP %d", response.StatusCode)
```

The token is the JWT returned by Ceph Dashboard's `POST /api/auth` endpoint. A preconfigured `http.Client` can be supplied with `WithHTTPClient`, for example to set timeouts or a private-CA transport.
