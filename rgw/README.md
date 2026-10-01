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
resources, locks, limits, and common server errors. Unknown codes remain
available through `APIError.Code`.

## RGW Admin Ops status

This checklist covers the direct Admin Ops resources registered by Ceph
`v20.2.4`. These methods call the RGW daemon directly and return its Admin Ops
response without Dashboard-specific transformations or enrichment.
All operations in these registered resources are implemented, including the
legacy and generation-aware data-log notification formats.

### Users

- [x] `ListUsers` — `GET /admin/user?list`
- [x] `GetUser` — `GET /admin/user`
- [x] `GetUserQuotas` — `GET /admin/user?quota`, returns both scopes
- [x] `GetUserQuota` — `GET /admin/user?quota&quota-type=user`, returns `Quota`
- [x] `GetUserBucketQuota` — `GET /admin/user?quota&quota-type=bucket`, returns `Quota`
- [x] `CreateUser` — `PUT /admin/user`
- [x] `CreateSubuser` — `PUT /admin/user?subuser`
- [x] `CreateS3Key` — `PUT /admin/user?key&key-type=s3`, returns `[]AccessKey`
- [x] `CreateSwiftKey` — `PUT /admin/user?key&key-type=swift`, returns `[]SwiftKey`
- [x] `AddUserCapabilities` — `PUT /admin/user?caps`
- [x] `SetUserQuota` — `PUT /admin/user?quota`
- [x] `UpdateUser` — `POST /admin/user`
- [x] `UpdateSubuser` — `POST /admin/user?subuser`
- [x] `DeleteUser` — `DELETE /admin/user`
- [x] `DeleteSubuser` — `DELETE /admin/user?subuser`
- [x] `DeleteKey` — `DELETE /admin/user?key`
- [x] `DeleteUserCapabilities` — `DELETE /admin/user?caps`

`CreateS3Key` and `CreateSwiftKey` use separate requests and fix the key type
internally. Both return the user's complete collection of that key type after
creation, including existing keys. Swift creation requires a subuser and has
no `AccessKey` field because Ceph derives the identity from the user and subuser.

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

Both listing methods use `ListBucketsRequest.UID` to filter by user; an empty
UID lists all buckets. For an account member, Ceph lists that account's buckets.

`ListBuckets` fetches details and statistics; use `ListBucketNames` for the
cheaper name-only listing.

### Accounts

- [x] `GetAccount` — `GET /admin/account`
- [x] `CreateAccount` — `POST /admin/account`
- [x] `UpdateAccount` — `PUT /admin/account`
- [x] `SetAccountQuota` — `PUT /admin/account?quota`
- [x] `DeleteAccount` — `DELETE /admin/account`

`SetAccountQuotaRequest` uses optional `int32` limits; `MaxSize` is in bytes
and there is no KiB parameter. Stored quotas
and their response models still use `int64`. User and named-bucket quota setter
requests use optional `int64` limits and also support `MaxSizeKB`.

### Usage and gateway information

- [x] `GetGatewayInfo` — `GET /admin/info`
- [x] `GetUsage` — `GET /admin/usage`
- [x] `ListUsageEntries` — `GET /admin/usage`, returns `[]UsageEntry`
- [x] `ListUsageSummaries` — `GET /admin/usage`, returns `[]UsageSummary`
- [x] `TrimUsage` — `DELETE /admin/usage`

`GetUsage` returns both entries and summaries. Use `ListUsageEntries` for
detailed records or `ListUsageSummaries` for aggregate counters. All three
share `GetUsageRequest` filters.

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
- [x] `ListDataLogEntries` — `GET /admin/log?type=data&id=...&extra-info=true`
- [x] `ListDataLogChanges` — `GET /admin/log?type=data&id=...&extra-info=false`
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

Data-log listings share `ListDataLogEntriesRequest` for shard and pagination
filters. `ListDataLogEntries` returns `DataLogEntryList` with log IDs, log
timestamps, and nested changes; `ListDataLogChanges` returns `DataLogChangeList`
with bare changes. Both pages retain `Marker`, `LastUpdate`, and `Truncated`.
When `Truncated` is true, pass the page's opaque `Marker` to the next request.

Trims remove log entries, retaining the underlying resources. Metadata and
data trims use a bounding `Marker`; bucket-index trims use `StartMarker`,
`EndMarker`, and an optional `Generation`. Bucket-index trimming requires
`BucketInstance` in v20.2.4, as do its log reads. Ceph v20.2.4 returns HTTP 500
`UnknownError` when a metadata trim range is empty; the client preserves that error.

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
realm after the response.

### Rate limits

- [x] `GetUserRateLimit` — `GET /admin/ratelimit?ratelimit-scope=user`, returns `RateLimit`
- [x] `GetBucketRateLimit` — `GET /admin/ratelimit?ratelimit-scope=bucket`, returns `RateLimit`
- [x] `GetGlobalRateLimits` — `GET /admin/ratelimit?global=true`, returns `GlobalRateLimitConfiguration`
- [x] `SetUserRateLimit` — `POST /admin/ratelimit?ratelimit-scope=user`
- [x] `SetBucketRateLimit` — `POST /admin/ratelimit?ratelimit-scope=bucket`
- [x] `SetGlobalRateLimit` — `POST /admin/ratelimit?global=true` for one scope

Rate-limit reads have fixed response types and separate user/bucket requests;
global reads accept no target. `GlobalRateLimitConfiguration` always contains
the bucket, user, and anonymous defaults as values. The user and bucket methods
return their stored configuration, without merging global defaults; an
unconfigured resource returns zero limits with `Enabled` false.

Setters also use separate requests: user updates take `UID`, bucket updates
take `Bucket` and optional `Tenant`, and global updates take `Scope` (`user`,
`bucket`, or `anon`). Each embeds `RateLimitUpdate`; nil fields preserve stored
values, zero limits mean unlimited, and `Enabled: new(false)` disables the limit.
Provide at least one change.

```go
err := client.SetUserRateLimit(ctx, rgw.SetUserRateLimitRequest{
    UID: "alice",
    RateLimitUpdate: rgw.RateLimitUpdate{
        MaxReadOps: new(int64(100)),
        Enabled:    new(true),
    },
})
```
