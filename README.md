# rgw-go

`rgw-go` provides two explicit Go clients:

- `github.com/sj14/rgw-go/dashboard` for Ceph Dashboard APIs;
- `github.com/sj14/rgw-go/admin` for direct RGW Admin Ops calls signed with
  AWS Signature Version 4.

## Dashboard API status

This checklist covers the currently targeted Dashboard controllers in Ceph
`v20.2.4`. Checked items are available through the public Dashboard client;
unchecked items are not yet implemented. Controller helpers that are not HTTP
endpoints are omitted.

### Cluster health

- [x] `GetFullHealth` — `GET /api/health/full`
- [x] `GetMinimalHealth` — `GET /api/health/minimal`
- [x] `GetClusterCapacity` — `GET /api/health/get_cluster_capacity`
- [x] `GetClusterFSID` — `GET /api/health/get_cluster_fsid`
- [x] `GetTelemetryStatus` — `GET /api/health/get_telemetry_status`
- [x] `GetHealthSnapshot` — `GET /api/health/snapshot`

### Pools

- [x] `ListPools` — `GET /api/pool`
- [x] `GetPool` — `GET /api/pool/{pool_name}`
- [x] `GetPoolConfiguration` — `GET /api/pool/{pool_name}/configuration`
- [x] `GetPoolInfo` — `GET /ui-api/pool/info`
- [ ] Create pool — `POST /api/pool`
- [ ] Update pool — `PUT /api/pool/{pool_name}`
- [ ] Delete pool — `DELETE /api/pool/{pool_name}`

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
- [x] `GetGlobalBucketRateLimit` — `GET /api/rgw/bucket/ratelimit`
- [x] `GetBucketRateLimit` — `GET /api/rgw/bucket/{uid}/ratelimit`
- [x] `UpdateBucketRateLimit` — `PUT /api/rgw/bucket/{uid}/ratelimit`
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
- [x] `CreateSwiftKey` — `POST /api/rgw/user/{uid}/key`
- [x] `DeleteSwiftKey` — `DELETE /api/rgw/user/{uid}/key`
- [x] `GetUserQuota` — `GET /api/rgw/user/{uid}/quota`
- [x] `UpdateUserQuota` — `PUT /api/rgw/user/{uid}/quota`
- [x] `CreateSubuser` — `POST /api/rgw/user/{uid}/subuser`
- [x] `DeleteSubuser` — `DELETE /api/rgw/user/{uid}/subuser/{subuser}`
- [x] `GetGlobalUserRateLimit` — `GET /api/rgw/user/ratelimit`
- [x] `GetUserRateLimit` — `GET /api/rgw/user/{uid}/ratelimit`
- [x] `UpdateUserRateLimit` — `PUT /api/rgw/user/{uid}/ratelimit`

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

## RGW Admin Ops status

This checklist covers the direct Admin Ops resources registered by Ceph
`v20.2.4`. These methods call the RGW daemon directly and return its Admin Ops
response without Dashboard-specific transformations or enrichment.

### Users

- [x] `ListUsers` — `GET /admin/user?list`
- [x] `GetUser` — `GET /admin/user`
- [x] `GetUserQuota` — `GET /admin/user?quota`
- [x] `CreateUser` — `PUT /admin/user`
- [x] `CreateSubuser` — `PUT /admin/user?subuser`
- [x] `CreateKey` — `PUT /admin/user?key`
- [x] `AddUserCapabilities` — `PUT /admin/user?caps`
- [x] `SetUserQuota` — `PUT /admin/user?quota`
- [x] `UpdateUser` — `POST /admin/user`
- [x] `UpdateSubuser` — `POST /admin/user?subuser`
- [x] `DeleteUser` — `DELETE /admin/user`
- [x] `DeleteSubuser` — `DELETE /admin/user?subuser`
- [x] `DeleteKey` — `DELETE /admin/user?key`
- [x] `DeleteUserCapabilities` — `DELETE /admin/user?caps`

### Buckets

- [x] `ListBuckets` — `GET /admin/bucket`
- [x] `GetBucket` — `GET /admin/bucket`
- [x] `GetBucketPolicy` — `GET /admin/bucket?policy`
- [x] `CheckBucketIndex` — `GET /admin/bucket?index`
- [x] `LinkBucket` — `PUT /admin/bucket`
- [x] `SetBucketQuota` — `PUT /admin/bucket?quota`
- [x] `SetBucketSync` — `PUT /admin/bucket?sync`
- [x] `UnlinkBucket` — `POST /admin/bucket`
- [x] `DeleteBucket` — `DELETE /admin/bucket`
- [x] `DeleteObject` — `DELETE /admin/bucket?object`

The Admin Ops API does not create buckets. `LinkBucket` changes the owner link
of an existing bucket; bucket creation remains an S3 operation.

### Accounts

- [ ] Get account — `GET /admin/account`
- [ ] Create account — `POST /admin/account`
- [ ] Update account — `PUT /admin/account`
- [ ] Set account quota — `PUT /admin/account?quota`
- [ ] Delete account — `DELETE /admin/account`

### Usage and gateway information

- [ ] Get gateway information — `GET /admin/info`
- [ ] Get usage — `GET /admin/usage`
- [ ] Trim usage — `DELETE /admin/usage`

### Metadata

- [ ] List metadata keys — `GET /admin/metadata`
- [ ] Get metadata — `GET /admin/metadata?key=...`
- [ ] Get local metadata — `GET /admin/metadata?myself`
- [ ] Write metadata — `PUT /admin/metadata`
- [ ] Delete metadata — `DELETE /admin/metadata`

### Logs

- [ ] Get metadata log information, shards, entries, or status —
  `GET /admin/log?type=metadata`
- [ ] Lock, unlock, or notify a metadata log —
  `POST /admin/log?type=metadata`
- [ ] Delete metadata log entries — `DELETE /admin/log?type=metadata`
- [ ] Get bucket-index log information, entries, or status —
  `GET /admin/log?type=bucket-index`
- [ ] Delete bucket-index log entries —
  `DELETE /admin/log?type=bucket-index`
- [ ] Get data log information, shards, entries, or status —
  `GET /admin/log?type=data`
- [ ] Notify a data log — `POST /admin/log?type=data`
- [ ] Delete data log entries — `DELETE /admin/log?type=data`

### Zone, realm, and period

- [ ] Get zone configuration — `GET /admin/config?type=zone`
- [ ] Get realm — `GET /admin/realm`
- [ ] List realms — `GET /admin/realm?list`
- [ ] Get period — `GET /admin/realm/period`
- [ ] Push or commit period — `POST /admin/realm/period`

### Rate limits

- [ ] Get user, bucket, or global rate limits — `GET /admin/ratelimit`
- [ ] Set user, bucket, or global rate limits — `PUT /admin/ratelimit`

## Dashboard usage

Import `github.com/sj14/rgw-go/dashboard` and construct a client with a
Dashboard JWT:

```go
client, err := dashboard.NewClient(
    "https://ceph-dashboard.example",
    dashboard.WithBearerToken(os.Getenv("CEPH_DASHBOARD_TOKEN")),
)
if err != nil {
    log.Fatal(err)
}

err = client.CreateBucket(ctx, dashboard.CreateBucketRequest{
    Name: "backups",
    UID:  "alice",
})
if err != nil {
    log.Fatal(err)
}

bucket, err := client.GetBucket(ctx, dashboard.GetBucketRequest{
    Name: "backups",
})
if err != nil {
    log.Fatal(err)
}
log.Printf("bucket %s is owned by %s", bucket.Name, bucket.Owner)
```

The token is the JWT returned by Ceph Dashboard's `POST /api/auth` endpoint. A preconfigured `http.Client` can be supplied with `WithHTTPClient`, for example to set timeouts or a private-CA transport.

## Admin Ops usage

Import `github.com/sj14/rgw-go/admin` and provide RGW access and secret keys:

```go
client, err := admin.NewClient(
    "https://rgw.example",
    os.Getenv("RGW_ACCESS_KEY"),
    os.Getenv("RGW_SECRET_KEY"),
)
if err != nil {
    log.Fatal(err)
}

user, err := client.GetUser(ctx, admin.GetUserRequest{UID: "alice"})
if err != nil {
    log.Fatal(err)
}
log.Printf("user %s has display name %s", user.ID, user.DisplayName)
```

The RGW credentials need suitable Admin Ops capabilities, such as
`users=read` for `GetUser`, `users=write` for user mutations, and the analogous
`buckets` grants for bucket operations. The Admin Ops resource defaults to
`/admin` and can be changed with `admin.WithAdminPath`.

## Integration tests

A purpose-built, ephemeral Ceph image with MON, MGR Dashboard, an in-memory
OSD, and RGW lives in [`test/ceph`](test/ceph/README.md). It is substantially
smaller than the general-purpose official image and includes real integration
tests for the implemented Dashboard and direct Admin Ops endpoints.
