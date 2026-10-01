# RGW client

Package `rgw` calls Ceph's RGW Admin Ops API directly and signs requests with
AWS Signature Version 4.

## Usage

```go
client, err := rgw.NewClient(
	"https://rgw.example",
	os.Getenv("RGW_ACCESS_KEY"),
	os.Getenv("RGW_SECRET_KEY"),
)
if err != nil {
	log.Fatal(err)
}

user, err := client.GetUser(ctx, rgw.GetUserRequest{UID: "alice"})
if err != nil {
	log.Fatal(err)
}
log.Printf("user %s has display name %s", user.ID, user.DisplayName)
```

The credentials need suitable Admin Ops capabilities, such as `users=read` for
`GetUser`, `users=write` for user mutations, and analogous `buckets` grants for
bucket operations. The Admin Ops resource defaults to `/admin` and can be
changed with `rgw.WithAdminPath`. A preconfigured `http.Client` can be supplied
with `WithHTTPClient`; otherwise the client uses a 30-second total request
timeout.

API failures return `*rgw.APIError`. Use `errors.Is(err, rgw.ErrKeyExists)`
(or another `ErrorCode` constant) to match Ceph's exact wire code. Constants
cover user/key/capability conflicts and validation, authentication, missing
resources, locks, limits, and common server errors. They are verified against
Ceph v20.2.4's `src/rgw/rgw_common.cc` S3 error mapping, which Admin Ops uses;
unknown codes remain available through `APIError.Code`.

## RGW Admin Ops status

This checklist covers the direct Admin Ops resources registered by Ceph
`v20.2.4`. These methods call the RGW daemon directly and return its Admin Ops
response without Dashboard-specific transformations or enrichment.
All operations in these registered resources are implemented, including the
legacy and generation-aware data-log notification formats.

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

- [x] `ListBucketNames` — `GET /admin/bucket?stats=false`, returns `[]string`
- [x] `ListBuckets` — `GET /admin/bucket?stats=true`, returns `[]Bucket`
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

Both listing methods use `ListBucketsRequest` for owner filters. `ListBuckets`
always fetches details and statistics; use `ListBucketNames` for the cheaper
name-only listing. The request has no `Stats` flag, and name-only responses are
never converted into partially populated `Bucket` values.

### Accounts

- [x] `GetAccount` — `GET /admin/account`
- [x] `CreateAccount` — `POST /admin/account`
- [x] `UpdateAccount` — `PUT /admin/account`
- [x] `SetAccountQuota` — `PUT /admin/account?quota`
- [x] `DeleteAccount` — `DELETE /admin/account`

### Usage and gateway information

- [x] `GetGatewayInfo` — `GET /admin/info`
- [x] `GetUsage` — `GET /admin/usage`
- [x] `TrimUsage` — `DELETE /admin/usage`

### Metadata

- [x] `ListMetadataKeys` — `GET /admin/metadata[/<section>]`
- [x] `GetMetadata` — `GET /admin/metadata[/<section>]?key=...`
- [x] `GetLocalMetadata` — `GET /admin/metadata/<section>?myself`
- [x] `PutMetadata` — `PUT /admin/metadata[/<section>]?key=...`
- [x] `DeleteMetadata` — `DELETE /admin/metadata[/<section>]?key=...`

`MetadataSection` provides constants for Ceph v20.2.4's registered sections:
- `account`
- `bucket`
- `bucket.instance`
- `group`
- `otp`
- `roles`
- `topic`
- `user`

The empty section lists the metadata root. Explicit conversions such
as `MetadataSection("new-section")` allow names from other Ceph versions.
The registration and wire names are verified in
`src/rgw/driver/rados/rgw_service.cc` and the handlers listed in `metadata.go`.

Metadata writes and deletes require `metadata=write`. `PutMetadata` sends the
same key/ver/mtime/data envelope returned by `GetMetadata` and decodes the
`RGWX_UPDATE_STATUS` and `RGWX_UPDATE_VERSION` headers into `MetadataUpdate`.
The returned version describes the previous on-disk version. `UpdateType`
accepts `always`, `update-by-version`, or `update-by-timestamp`; Ceph passes
this policy to the section's handler. The v20.2.4 user handler ignores it and
applies all three policies. The query selects the destination independently
of the body envelope's key.

### Logs

- [x] `GetMetadataLogInfo` — `GET /admin/log?type=metadata`
- [x] `GetMetadataLogShardInfo` — `GET /admin/log?type=metadata&id=...&info`
- [x] `ListMetadataLogEntries` — `GET /admin/log?type=metadata&id=...`
- [x] `GetMetadataLogStatus` — `GET /admin/log?type=metadata&status`
- [x] `LockMetadataLog` — `POST /admin/log?type=metadata&lock`
- [x] `UnlockMetadataLog` — `POST /admin/log?type=metadata&unlock`
- [x] `NotifyMetadataLog` — `POST /admin/log?type=metadata&notify`
- [x] `TrimMetadataLog` — `DELETE /admin/log?type=metadata`
- [x] `GetBucketIndexLogInfo` — `GET /admin/log?type=bucket-index&info`
- [x] `ListBucketIndexLogEntries` — `GET /admin/log?type=bucket-index`
- [x] `GetBucketIndexLogStatus` — `GET /admin/log?type=bucket-index&status`
- [x] `TrimBucketIndexLog` — `DELETE /admin/log?type=bucket-index`
- [x] `GetDataLogInfo` — `GET /admin/log?type=data`
- [x] `GetDataLogShardInfo` — `GET /admin/log?type=data&id=...&info`
- [x] `ListDataLogEntries` — `GET /admin/log?type=data&id=...`
- [x] `GetDataLogStatus` — `GET /admin/log?type=data&status`
- [x] `NotifyDataLog` — `POST /admin/log?type=data&notify`
- [x] `NotifyDataLogV2` — `POST /admin/log?type=data&notify2`
- [x] `TrimDataLog` — `DELETE /admin/log?type=data`

These log mutations require `mdlog=write`, `bilog=write`, or `datalog=write`
according to the log type. Metadata locks use the period and shard ID, a
locker cookie, a zone tag, and a positive duration in seconds. Repeating a lock
with the same cookie/tag renews it; a competing lock returns HTTP 423.
Metadata notifications send a JSON array of shard IDs. Data notifications
send Ceph's array of key/val shard entries, with key-only strings for the
original format and key/gen objects for `notify2`. Nil top-level notification
slices are sent as empty arrays.

Trims remove log entries, retaining the underlying resources. Metadata and
data trims use a bounding `Marker`; bucket-index trims use `StartMarker`,
`EndMarker`, and an optional `Generation`. Bucket-index trimming requires
`BucketInstance` in v20.2.4, as do its log reads. Ceph v20.2.4 returns HTTP 500
`UnknownError` when a metadata trim range is empty (`cls_log_trim` returns
`ENODATA`); the client preserves that error.

The mutation contracts were verified against v20.2.4's
`src/rgw/rgw_rest_metadata.cc`/`.h`,
`src/rgw/driver/rados/rgw_rest_log.cc`/`.h`, `src/rgw/rgw_metadata.cc`,
`src/rgw/driver/rados/rgw_metadata.cc`, `src/rgw/driver/rados/rgw_user.cc`,
`src/rgw/services/svc_cls.cc`, `src/rgw/services/svc_bilog_rados.cc`,
`src/rgw/driver/rados/rgw_datalog.cc`,
`src/rgw/driver/rados/rgw_datalog_notify.cc`, and `src/cls/log/cls_log.cc`.

### Zone, realm, and period

- [x] `GetZoneConfiguration` — `GET /admin/config?type=zone`
- [x] `GetRealm` — `GET /admin/realm`
- [x] `ListRealms` — `GET /admin/realm?list`
- [x] `GetPeriod` — `GET /admin/realm/period`
- [x] `PushPeriod` — `POST /admin/realm/period` with a non-empty period ID
- [x] `CommitPeriod` — `POST /admin/realm/period` with an empty period ID

These reads require `zone=read`. `GetZoneConfiguration` returns the serving
zone's configuration, including its system key. `GetRealm` accepts `ID` or
`Name`, with an empty request selecting the default realm. `ListRealms` returns
realm names and the default realm ID. `GetPeriod` accepts `RealmID`, `PeriodID`,
and `Epoch`; omitted identifiers select the default realm/current period, and
epoch zero selects the latest epoch. A gateway without a configured realm
returns a not-found error for default realm and period lookups.

Period mutations require `zone=write` and take a `Period` as their JSON body.
Its realm must match the serving gateway's realm. `PushPeriod` sends an
existing period to a non-master zone. `CommitPeriod` sends a proposed period
to its master zone, with an empty ID, the current period as predecessor, and
the next realm epoch. Ceph returns the resulting period and can reload the
realm after the response. These contracts were verified in
`src/rgw/driver/rados/rgw_rest_realm.cc`, `src/rgw/rgw_period.cc`, and
`src/rgw/driver/rados/rgw_period.cc`.

Verified against Ceph **v20.2.4**, selected from the `releases` section of
[`main/doc/releases/releases.yml`](https://github.com/ceph/ceph/blob/main/doc/releases/releases.yml)
and confirmed `stable` by the tag's
[`src/ceph_release`](https://github.com/ceph/ceph/blob/v20.2.4/src/ceph_release).
The authoritative handlers are `src/rgw/rgw_rest_config.cc`/`.h` and
`src/rgw/driver/rados/rgw_rest_realm.cc`; model formatters are in
`src/rgw/rgw_zone.cc`, `src/rgw/rgw_realm.cc`, `src/rgw/rgw_period.cc`, and
`src/rgw/rgw_sync_policy.cc`. Resource registration is in
`src/rgw/driver/rados/rgw_sal_rados.cc`. Ceph's multisite tests in
`src/test/rgw/rgw_multi` exercise realm and period configuration.

### Rate limits

- [x] `GetRateLimit` — get user, bucket, or global rate limits through
  `GET /admin/ratelimit`
- [x] `SetRateLimit` — set user, bucket, or global rate limits through
  `POST /admin/ratelimit`
