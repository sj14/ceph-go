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
- [ ] Write metadata — `PUT /admin/metadata`
- [ ] Delete metadata — `DELETE /admin/metadata`

### Logs

- [x] `GetMetadataLogInfo` — `GET /admin/log?type=metadata`
- [x] `GetMetadataLogShardInfo` — `GET /admin/log?type=metadata&id=...&info`
- [x] `ListMetadataLogEntries` — `GET /admin/log?type=metadata&id=...`
- [x] `GetMetadataLogStatus` — `GET /admin/log?type=metadata&status`
- [ ] Lock, unlock, or notify a metadata log — `POST /admin/log?type=metadata`
- [ ] Delete metadata log entries — `DELETE /admin/log?type=metadata`
- [x] `GetBucketIndexLogInfo` — `GET /admin/log?type=bucket-index&info`
- [x] `ListBucketIndexLogEntries` — `GET /admin/log?type=bucket-index`
- [x] `GetBucketIndexLogStatus` — `GET /admin/log?type=bucket-index&status`
- [ ] Delete bucket-index log entries — `DELETE /admin/log?type=bucket-index`
- [x] `GetDataLogInfo` — `GET /admin/log?type=data`
- [x] `GetDataLogShardInfo` — `GET /admin/log?type=data&id=...&info`
- [x] `ListDataLogEntries` — `GET /admin/log?type=data&id=...`
- [x] `GetDataLogStatus` — `GET /admin/log?type=data&status`
- [ ] Notify a data log — `POST /admin/log?type=data`
- [ ] Delete data log entries — `DELETE /admin/log?type=data`

### Zone, realm, and period

- [ ] Get zone configuration — `GET /admin/config?type=zone`
- [ ] Get realm — `GET /admin/realm`
- [ ] List realms — `GET /admin/realm?list`
- [ ] Get period — `GET /admin/realm/period`
- [ ] Push or commit period — `POST /admin/realm/period`

### Rate limits

- [x] `GetRateLimit` — get user, bucket, or global rate limits through
  `GET /admin/ratelimit`
- [x] `SetRateLimit` — set user, bucket, or global rate limits through
  `POST /admin/ratelimit`
