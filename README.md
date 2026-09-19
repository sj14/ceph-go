# rgw-go

`rgw-go` is a Go client for RGW management through the Ceph Dashboard REST API.

The implementation was verified on 2026-09-19 against the latest stable Ceph release, `v20.2.4` (tag commit `7f793731f1b3`), rather than the generated API documentation. Stability was confirmed through `doc/releases/releases.yml` and the tag's `src/ceph_release` file:

- [Stable release metadata](https://github.com/ceph/ceph/blob/main/doc/releases/releases.yml)
- [`RgwBucket.create`](https://github.com/ceph/ceph/blob/v20.2.4/src/pybind/mgr/dashboard/controllers/rgw.py)
- [`RgwBucket.list`](https://github.com/ceph/ceph/blob/v20.2.4/src/pybind/mgr/dashboard/controllers/rgw.py)
- [`RgwBucket.get`](https://github.com/ceph/ceph/blob/v20.2.4/src/pybind/mgr/dashboard/controllers/rgw.py)
- [`RgwBucket.set`](https://github.com/ceph/ceph/blob/v20.2.4/src/pybind/mgr/dashboard/controllers/rgw.py)
- [`RgwBucket.delete`](https://github.com/ceph/ceph/blob/v20.2.4/src/pybind/mgr/dashboard/controllers/rgw.py)
- [`RgwUser`](https://github.com/ceph/ceph/blob/v20.2.4/src/pybind/mgr/dashboard/controllers/rgw.py)
- [Ceph Dashboard's RGW bucket frontend client](https://github.com/ceph/ceph/blob/v20.2.4/src/pybind/mgr/dashboard/frontend/src/app/shared/api/rgw-bucket.service.ts)
- [`RgwClient.create_bucket`](https://github.com/ceph/ceph/blob/v20.2.4/src/pybind/mgr/dashboard/services/rgw_client.py)
- [REST routing and status mapping](https://github.com/ceph/ceph/blob/v20.2.4/src/pybind/mgr/dashboard/controllers/_rest_controller.py)

## Implementation status

This checklist covers the RGW-related Dashboard controllers in Ceph `v20.2.4`.
Checked items are available through the public Go client; unchecked items are
not yet implemented. Controller helpers that are not HTTP endpoints are omitted.

### Buckets

- [x] `CreateBucket` — `POST /api/rgw/bucket`
- [x] `GetBucket` — `GET /api/rgw/bucket/{bucket}`
- [x] `ListBuckets` — `GET /api/rgw/bucket?stats=true` (API version 1.1)
- [x] `UpdateBucket` — `PUT /api/rgw/bucket/{bucket}`
- [x] `DeleteBucket` — `DELETE /api/rgw/bucket/{bucket}`
- [ ] Set encryption configuration — `PUT /api/rgw/bucket/setEncryptionConfig`
- [ ] Get bucket encryption — `GET /api/rgw/bucket/getEncryption`
- [ ] Delete bucket encryption — `DELETE /api/rgw/bucket/deleteEncryption`
- [ ] Get encryption configuration — `GET /api/rgw/bucket/getEncryptionConfig`
- [ ] Set lifecycle policy — `PUT /api/rgw/bucket/lifecycle`
- [ ] Get lifecycle policy — `GET /api/rgw/bucket/lifecycle`
- [ ] Get bucket notifications — `GET /api/rgw/bucket/notification`
- [ ] Create or update bucket notifications — `PUT /api/rgw/bucket/notification`
- [ ] Delete bucket notifications — `DELETE /api/rgw/bucket/notification`
- [ ] Get global bucket rate limits — `GET /api/rgw/bucket/ratelimit`
- [ ] Get bucket rate limits — `GET /api/rgw/bucket/{uid}/ratelimit`
- [ ] Update bucket rate limits — `PUT /api/rgw/bucket/{uid}/ratelimit`
- [ ] Get bucket and user counts — `GET /ui-api/rgw/bucket`

### Status and multisite setup

- [ ] Get RGW status — `GET /ui-api/rgw/status`
- [ ] Get multisite status — `GET /ui-api/rgw/multisite/status`
- [ ] Migrate to multisite — `PUT /ui-api/rgw/multisite/migrate`
- [ ] Set up multisite replication — `POST /ui-api/rgw/multisite/multisite-replications`
- [ ] Get available RGW ports — `GET /ui-api/rgw/multisite/available-ports`
- [ ] Check RGW daemon status — `GET /ui-api/rgw/multisite/check-daemons-status`

### Multisite policies

- [ ] Get sync status — `GET /api/rgw/multisite/sync_status`
- [ ] Get sync policy — `GET /api/rgw/multisite/sync-policy`
- [ ] Get sync-policy group — `GET /api/rgw/multisite/sync-policy-group`
- [ ] Create sync-policy group — `POST /api/rgw/multisite/sync-policy-group`
- [ ] Update sync-policy group — `PUT /api/rgw/multisite/sync-policy-group`
- [ ] Delete sync-policy group — `DELETE /api/rgw/multisite/sync-policy-group`
- [ ] Create or update sync flow — `PUT /api/rgw/multisite/sync-flow`
- [ ] Delete sync flow — `DELETE /api/rgw/multisite/sync-flow`
- [ ] Create or update sync pipe — `PUT /api/rgw/multisite/sync-pipe`
- [ ] Delete sync pipe — `DELETE /api/rgw/multisite/sync-pipe`

### Daemons and sites

- [ ] List RGW daemons — `GET /api/rgw/daemon`
- [ ] Get RGW daemon — `GET /api/rgw/daemon/{svc_id}`
- [ ] Set daemon multisite configuration — `PUT /api/rgw/daemon/set_multisite_config`
- [ ] Query site information — `GET /api/rgw/site`

### Users and access credentials

- [x] `ListUsers` — `GET /api/rgw/user?detailed=true`
- [x] `GetUser` — `GET /api/rgw/user/{uid}`
- [ ] Get user email addresses — `RgwUser.get_emails`
- [x] `CreateUser` — `POST /api/rgw/user`
- [x] `UpdateUser` — `PUT /api/rgw/user/{uid}`
- [x] `DeleteUser` — `DELETE /api/rgw/user/{uid}`
- [x] `AddUserCapability` — `POST /api/rgw/user/{uid}/capability`
- [x] `DeleteUserCapability` — `DELETE /api/rgw/user/{uid}/capability`
- [x] `CreateAccessKey` — `POST /api/rgw/user/{uid}/key`
- [x] `DeleteAccessKey` — `DELETE /api/rgw/user/{uid}/key`
- [x] `GetUserQuota` — `GET /api/rgw/user/{uid}/quota`
- [x] `UpdateUserQuota` — `PUT /api/rgw/user/{uid}/quota`
- [x] `CreateSubuser` — `POST /api/rgw/user/{uid}/subuser`
- [x] `DeleteSubuser` — `DELETE /api/rgw/user/{uid}/subuser/{subuser}`
- [ ] Get global user rate limits — `GET /api/rgw/user/ratelimit`
- [ ] Get user rate limits — `GET /api/rgw/user/{uid}/ratelimit`
- [ ] Update user rate limits — `PUT /api/rgw/user/{uid}/ratelimit`

### Account roles

- [ ] List roles — `RGWRoleEndpoints.role_list`
- [ ] Get role — `RGWRoleEndpoints.get`
- [ ] Create role — `RGWRoleEndpoints.role_create`
- [ ] Update role — `RGWRoleEndpoints.role_update`
- [ ] Delete role — `RGWRoleEndpoints.role_delete`

These operations are rooted at `/api/rgw/accounts/{account_id}/roles` and are
generated by Ceph's `CRUDEndpoint` controller.

### Realms

- [ ] List realms — `GET /api/rgw/realm`
- [ ] Get realm — `GET /api/rgw/realm/{realm_name}`
- [ ] Create realm — `POST /api/rgw/realm`
- [ ] Update realm — `PUT /api/rgw/realm/{realm_name}`
- [ ] Delete realm — `DELETE /api/rgw/realm/{realm_name}`
- [ ] Get complete realm information — `RgwRealm.get_all_realms_info`
- [ ] Get realm tokens — `RgwRealm.get_realm_tokens`
- [ ] Import realm token — `RgwRealm.import_realm_token`

### Zonegroups

- [ ] List zonegroups — `GET /api/rgw/zonegroup`
- [ ] Get zonegroup — `GET /api/rgw/zonegroup/{zonegroup_name}`
- [ ] Create zonegroup — `POST /api/rgw/zonegroup`
- [ ] Update zonegroup — `PUT /api/rgw/zonegroup/{zonegroup_name}`
- [ ] Delete zonegroup — `DELETE /api/rgw/zonegroup/{zonegroup_name}`
- [ ] Get complete zonegroup information — `RgwZonegroup.get_all_zonegroups_info`
- [ ] Get placement target — `RgwZonegroup.get_placement_target_by_placement_id`
- [ ] Create storage class — `RgwZonegroup.storage_class`
- [ ] Update storage class — `RgwZonegroup.editStorageClass`
- [ ] Delete storage class — `RgwZonegroup.remove_storage_class`

### Zones

- [ ] List zones — `GET /api/rgw/zone`
- [ ] Get zone — `GET /api/rgw/zone/{zone_name}`
- [ ] Create zone — `POST /api/rgw/zone`
- [ ] Update zone — `PUT /api/rgw/zone/{zone_name}`
- [ ] Delete zone — `DELETE /api/rgw/zone/{zone_name}`
- [ ] Get complete zone information — `RgwZone.get_all_zones_info`
- [ ] Get pool names — `RgwZone.get_pool_names`
- [ ] Create system user — `RgwZone.create_system_user`
- [ ] Get zone user list — `RgwZone.get_user_list`
- [ ] Create storage class — `RgwZone.create_storage_class`
- [ ] Update storage class — `RgwZone.edit_storage_class`

### Topics

- [ ] List topics — `GET /api/rgw/topic`
- [ ] Get topic — `GET /api/rgw/topic/{key}`
- [ ] Create topic — `POST /api/rgw/topic`
- [ ] Delete topic — `DELETE /api/rgw/topic/{key}`

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
smaller than the general-purpose official image and includes real integration
tests for the implemented Dashboard endpoints.
