# Repository instructions

This repository provides Go clients for Ceph APIs. It currently contains
separate clients for Ceph Dashboard APIs and the direct RGW Admin Ops API.

## Source of truth

- Do not rely on the published Ceph API documentation when implementing or changing an endpoint.
- Before implementing an endpoint, identify the latest stable Ceph release from `doc/releases/releases.yml` in Ceph's `main` branch, considering only the `releases` section and excluding `development`.
- Confirm that the selected tag's `src/ceph_release` file declares `stable`. Do not treat the numerically highest Git tag or the mere presence of downloadable packages as proof of a stable release; Ceph also publishes development and release-candidate tags and artifacts.
- Inspect the implementation in that stable tag of <https://github.com/ceph/ceph>.
- For Dashboard endpoints, inspect the controller, routing/versioning code,
  the internal service called by the controller, and the Ceph frontend client
  or tests when available.
- For direct Admin Ops endpoints, inspect RGW's REST handler and method
  dispatch, parameter parsing, underlying admin operation, response formatter,
  authentication requirements, and tests when available.
- Record the verified Ceph release and relevant source files in code comments, tests, or this file so that later updates can be compared deliberately.
- Treat Ceph's release source code as authoritative when it differs from generated or published documentation.
- Keep READMEs focused on current usage and observable behavior. Until the
  library has a stable release, do not document former methods, fields, values,
  or migration history.
  Record source-file inventories, formatter details, and verification rationale
  in code comments, tests, or this file rather than expanding user documentation.

## Go implementation

- Keep the public API idiomatic and context-aware.
- Keep Dashboard and direct Admin Ops transports in the `mgr` and `rgw`
  packages respectively. Do not silently switch an operation between
  them: their authentication, authorization, errors, and response semantics
  differ even when Dashboard delegates to Admin Ops internally.
- Model source-verified finite or well-known wire values as named Go types with
  constants when implementing an endpoint or response model. Use the Go
  underlying type that matches the actual wire representation, such as
  `string` or `int64`, so callers can still explicitly convert values added by
  other Ceph versions. Do not invent constants for undocumented values or
  force genuinely open-ended fields into enums. When Ceph uses different wire
  representations or values for requests and responses, use separate types
  and constants for the two representations.
- Reuse one Go model when multiple fields or endpoints originate from the same
  Ceph source type and wire representation. Name shared models for the Ceph
  domain concept rather than for one endpoint, and do not duplicate identical
  structs merely because they occur in different controller families.
- Make response shape and data completeness predictable from method names and
  return types. When a Ceph parameter switches between distinct response forms
  (for example, bucket names versus detailed buckets), expose separate methods
  with fixed return types and set that parameter internally. Do not expose a
  shape-switching flag or normalize a name-only response into a mostly empty
  resource struct. Reuse shared filters and full domain models where appropriate.
  Model genuinely optional data explicitly, so missing data cannot be mistaken
  for a real zero value. Do not expose arbitrary field-selection parameters
  when decoding into a full resource model; request all ordinary fields instead.
  Prefer clear, hard-to-misuse APIs over preserving a
  confusing interface; breaking changes are acceptable when needed for this.
- Use separate methods and request types when target selectors have mutually
  exclusive fields, especially for resource-specific versus global mutations.
  Do not expose requests whose target flags silently discard supplied identifiers.
  Share the update values while keeping target fields specific to each operation.
- Require an explicit setting for actions whose omitted state triggers a
  mutation. A nullable setting must not silently mean "leave unchanged" when
  Ceph instead applies a default; validate its presence before sending.
- Expose one explicit unit for each request quantity when Ceph accepts multiple
  equivalent parameters with precedence rules. Prefer the stored domain unit
  rather than allowing conflicting values that the server silently overrides.
- Return decoded domain models and errors from endpoint methods. Keep HTTP status, headers, and raw successful response bodies internal unless an endpoint exposes meaningful transport metadata that callers need.
- Return single resources by value as `(Resource, error)`, collections as `([]Resource, error)`, and actions without a meaningful result as `error`. Use pointers within models only for nullable fields or when absence must be distinguishable from a zero value.
- Preserve Ceph's actual HTTP method, route, media type, parameter location, parameter names, and response status.
- Test shared transport behavior such as authentication and media-type headers,
  response-size limits, HTTP error mapping, and client validation once in the
  client unit tests. Do not add endpoint-specific `httptest` coverage by
  default: Ceph itself is the authoritative check for routes, parameters,
  statuses, and response shapes. Add a focused unit test only for substantial
  client-side logic that the integration suite cannot exercise clearly.
- Add each endpoint to the real Ceph container integration suite when it can be
  tested safely and reversed reliably. Prefer a
  create/get-or-list/update/delete roundtrip with cleanup for mutable
  resources. Treat this suite as the authoritative runtime contract check.
- Keep Dashboard integration tests in `test/ceph/integration/mgr` and direct
  Admin Ops integration tests in `test/ceph/integration/rgw`. Keep
  shared test infrastructure outside those two directories.
- Do not duplicate Ceph's API behavior or test suite with synthetic response
  fixtures or mocks of Ceph internals.
- Use the Go standard library unless an external dependency has a clear benefit.
- Run `gofmt`, `go test -short ./...`, and `go vet ./...` after changes. When
  the Ceph container is available, also run
  `go test -p 1 ./test/ceph/integration/...`; integration tests use
  `testing.Short()` as their only skip mechanism. The package-level
  serialization prevents one suite's fixture deletion from racing Ceph's
  non-atomic user listing in the other suite.

## CI and Ceph test image

- Reference GitHub Actions by their major-version tag so compatible minor and
  patch releases are adopted automatically; do not pin actions to commit hashes.
- Keep ordinary CI on the prebuilt `ceph-go-test:main` image and start it
  with `docker compose --no-build --pull always`. Do not rebuild Ceph for each
  library change.
- Build the Ceph image from `main` only when its build inputs change or when the
  image workflow is started manually. Test the candidate by immutable digest
  before promoting it to the moving `main` tag.
- Keep local Compose usage able to build the image directly when
  `CEPH_TEST_IMAGE` is not set.

## RGW source verification notes

These notes record the source checks behind the current RGW API. Recheck the
latest stable release using the procedure above before changing an endpoint.
The verified baseline is Ceph **v20.2.4**, selected from the `releases` section of
[`main/doc/releases/releases.yml`](https://github.com/ceph/ceph/blob/main/doc/releases/releases.yml)
and confirmed `stable` by the tag's
[`src/ceph_release`](https://github.com/ceph/ceph/blob/v20.2.4/src/ceph_release).
All source paths below refer to that tag.

- Error codes: `src/rgw/rgw_common.cc` contains the S3 error mapping used by
  Admin Ops; retain unknown wire codes through `APIError.Code`.
- User lookup: `RGWOp_User_Info` in `src/rgw/driver/rados/rgw_rest_user.cc`
  accepts UID and S3 access key. `RGWUser::init` in that directory's
  `rgw_user.cc` tries UID first and falls back to the key only when not found;
  the selectors do not verify each other. Keep `GetUser` UID-only and
  `GetUserByAccessKey` key-only, with a required selector and shared response
  and statistics handling. Integration coverage must verify both lookup paths,
  optional statistics, and a key that no longer identifies a user after deletion.
  An unknown key leaves an anonymous UID; `RGWAccessKeyPool::init` rejects it
  with EINVAL before `info` reaches its missing-user check. Preserve the
  resulting HTTP 400 `InvalidArgument` error.
- Key creation: `src/rgw/driver/rados/rgw_rest_user.cc` (`RGWOp_Key_Create`)
  parses the request. In the same directory, `rgw_user.cc`
  (`RGWUserAdminOp_Key::create`) returns the complete collection of the selected
  key type, and `RGWAccessKeyPool::generate_key` derives the Swift identity from
  the user and subuser. Keep separate typed S3 and Swift methods and requests.
- Key deletion: `RGWOp_Key_Remove` in that REST source accepts both selectors.
  `RGWAccessKeyPool::check_op` in `rgw_user.cc` infers Swift when a subuser is
  supplied without a key type; `check_existing_key` ignores `access-key` for
  explicit Swift and derives the identity from UID and Subuser. Expose separate
  S3/Swift deletion requests, fixing the type internally and requiring UID plus
  AccessKey or Subuser respectively. `RGWUserAdminOpState::set_subuser` allows
  a qualified subuser to override UID: require an unqualified deletion subuser.
  Verify missing-target rejection without mutation and preservation of other
  S3/Swift keys and the subuser in real Ceph integration tests.
- Subuser updates: `RGWOp_Subuser_Modify` in that REST source always calls
  `set_perm`, even when `access` is omitted. `rgw_str_to_perm` in
  `src/rgw/rgw_common.cc` maps empty access to no permissions, and
  `RGWSubUserPool::execute_modify` in `rgw_user.cc` applies it. Require a nonnil
  `UpdateSubuserRequest.Access`; a pointer to `SubuserAccessNone` explicitly
  clears permissions. Send its empty wire value rather than dropping it.
  Verify rejection without credential or permission changes, explicit access
  during secret rotation, and deliberate permission clearing in integration tests.
- Bucket listing: `src/rgw/driver/rados/rgw_rest_bucket.cc`
  (`RGWOp_Bucket_Info::execute`) does not parse an `account-id` listing filter.
  `rgw_bucket.cc` in the same directory resolves account membership from UID.
  Do not reintroduce an ineffective `AccountID` filter or a public `Stats` flag;
  the name-only and detailed methods must retain fixed return types.
- Bucket ownership: that REST source parses both `uid` and `account-id` for
  links and unlinks. `RGWBucketAdminOp::link/unlink` in `rgw_bucket.cc` selects
  the account first, while `RGWBucket::init` still uses UID for bucket lookup.
  Keep separate user/account methods and requests with only the applicable owner
  identifier. Linking rejects account-member user ownership; unlinking removes
  a list entry without changing bucket ownership. Verify both owner types and
  unlink/relink roundtrips with real temporary resources.
- Bucket synchronization: `RGWOp_Sync_Bucket` in the same REST source defaults
  an omitted or empty `sync` value to true. `RGWBucket::sync` in `rgw_bucket.cc`
  changes `BUCKET_DATASYNC_DISABLED` and writes the bucket metadata. Require
  `SetBucketSyncRequest.Enabled` and reject nil before sending. Integration
  coverage must verify explicit disable/enable and rejection without changing
  disabled state, using the bucket-index log's `SyncStopped` value.
- ACL reads: `RGWOp_Get_Policy` in that REST source and `RGWBucket::get_policy`
  in `rgw_bucket.cc` read a bucket or object's stored ACL. The `policy` route
  does not return an S3 bucket policy document. Use separate bucket/object ACL
  getters, requiring the object's name to prevent fallback to a bucket read.
  Share `AccessControlPolicy` and `AccessControlList`, matching their source
  types in `src/rgw/rgw_acl.cc`. Verify distinct bucket/object grants through
  S3 setup and direct Admin Ops reads in integration tests.
  `RGWHTTPArgs::append` in `src/rgw/rgw_common.cc` registers only the first
  Admin Ops subresource: encode `policy` before `object` so object ACL reads
  reach the policy handler. SigV4 canonical query sorting permits this order.
- Metadata sections: registration and wire names are in
  `src/rgw/driver/rados/rgw_service.cc` and the handlers listed in
  `rgw/metadata.go`.
- Account quotas: the REST parser uses `int32` limits while stored quota
  models use `int64`; preserve the distinct request representation.
- Account updates: `src/rgw/rgw_rest_account.cc` parses the update parameters,
  and `modify` in `src/rgw/rgw_account.cc` resolves the target by ID, then name,
  then email before applying name/email updates. Require `UpdateAccountRequest.ID`
  so update values cannot select a different target. Keep immutable Tenant out
  of the update request; empty name/email and nil limits preserve stored values.
  Integration coverage must verify missing-ID rejection without mutation and
  name/email updates by ID with an unrelated account unchanged.
- Quota size updates: `RGWOp_Quota_Set` in
  `src/rgw/driver/rados/rgw_rest_user.cc` and `RGWOp_Set_Bucket_Quota` in
  `src/rgw/driver/rados/rgw_rest_bucket.cc` parse `max-size` as bytes, then
  overwrite it when `max-size-kb` is present. Expose only the byte setting in
  user and named-bucket requests. Keep the response's derived `MaxSizeKB`,
  emitted by `RGWQuotaInfo::dump` in `src/rgw/rgw_quota.cc`. Integration tests
  must verify exact byte values and preservation during partial updates.
- Data-log listing: `src/rgw/driver/rados/rgw_rest_log.cc`
  (`RGWOp_DATALog_List`) switches between bare changes and detailed entries.
  In the same directory, `rgw_datalog.cc` (`rgw_data_change::dump` and
  `rgw_data_change_log_entry::dump`) defines the two wire shapes. Keep the
  `extra-info` parameter internal to the separate methods. Do not normalize
  bare changes into entries with empty log metadata.
- Rate-limit reads: `src/rgw/rgw_rest_ratelimit.cc`
  (`RGWOp_Ratelimit_Info::execute`) returns one scoped limit or all three global
  defaults. Keep separate getters and target requests. Global models use value
  fields and are shared with Dashboard through its type alias.
- Rate-limit writes: the same source's `RGWOp_Ratelimit_Set::execute` ignores
  individual identifiers for global writes and chooses the resource from scope.
  Keep separate user, bucket, and global setters with target-specific requests;
  only the global setter exposes `RateLimitScope`. Share `RateLimitUpdate`, whose
  nil fields preserve stored settings. Verify each target and partial updates
  in integration tests, restoring global settings before parallel tests resume.
- Usage reads: `src/rgw/rgw_rest_usage.cc` (`RGWOp_Usage_Get`) parses the
  `show-entries` and `show-summary` flags; `src/rgw/rgw_usage.cc`
  (`RGWUsage::show`) omits the corresponding arrays when false. Keep these
  flags internal: `GetUsage` requests both collections, `ListUsageEntries`
  requests entries only, and `ListUsageSummaries` requests summaries only.
  Reuse the filters in `GetUsageRequest` and return typed slices for individual
  collections rather than a partially populated `Usage`.

### Metadata and log mutations

The mutation contracts were verified against v20.2.4's
`src/rgw/rgw_rest_metadata.cc`/`.h`,
`src/rgw/driver/rados/rgw_rest_log.cc`/`.h`, `src/rgw/rgw_metadata.cc`,
`src/rgw/driver/rados/rgw_metadata.cc`, `src/rgw/driver/rados/rgw_user.cc`,
`src/rgw/services/svc_cls.cc`, `src/rgw/services/svc_bilog_rados.cc`,
`src/rgw/driver/rados/rgw_datalog.cc`,
`src/rgw/driver/rados/rgw_datalog_notify.cc`, and `src/cls/log/cls_log.cc`.
An empty metadata trim range returns `ENODATA` from `cls_log_trim`, which
Admin Ops maps to HTTP 500 `UnknownError`; preserve this server error.

### Zone, realm, and period

The authoritative handlers are `src/rgw/rgw_rest_config.cc`/`.h` and
`src/rgw/driver/rados/rgw_rest_realm.cc`; model formatters are in
`src/rgw/rgw_zone.cc`, `src/rgw/rgw_realm.cc`, `src/rgw/rgw_period.cc`, and
`src/rgw/rgw_sync_policy.cc`. Resource registration is in
`src/rgw/driver/rados/rgw_sal_rados.cc`. Ceph's multisite tests in
`src/test/rgw/rgw_multi` exercise realm and period configuration.
Period operations also use `src/rgw/driver/rados/rgw_period.cc`.

## Dashboard bucket update verification

Dashboard bucket responses were also verified against v20.2.4's
`RgwBucket.list` and `RgwBucket.get` in the controller below. Lists proxy Admin
Ops statistics and add `bid`, while detail reads additionally fetch S3
configuration. Return `[]BucketSummary` from `ListBuckets` and a full `Bucket`
embedding the shared summary fields from `GetBucket`. Do not restore one
model with zero-valued configuration fields for lists. The controller's
`map_bucket_owners` replaces known account IDs with account names only in
list responses; preserve and document that distinction.

Verified against Ceph v20.2.4's
`src/pybind/mgr/dashboard/controllers/rgw.py` (`RgwBucket.set`),
`services/rgw_client.py` (encryption, lifecycle, and locking helpers), and
`frontend/src/app/shared/api/rgw-bucket.service.ts` (`update`), relative to
`src/pybind/mgr/dashboard` for the latter two paths.
`qa/tasks/mgr/dashboard/test_rgw.py` exercises bucket updates.

The controller defaults encryption to false, removes lifecycle configuration
when missing or empty, and reapplies retention for object-lock-enabled buckets.
These are not patch semantics. Require explicit encryption and lifecycle
choices and reject omitted values before sending any request. Send explicit
false/empty values for deletion; do not silently fetch and merge settings or
switch to the direct RGW transport. Require an explicit Object Lock configuration
describing the current lock state. Validate enabled configurations for a valid
mode and exactly one positive retention period. The configuration's `Enabled`
field must be a required pointer so omission cannot be mistaken for false;
reject nil before dereferencing or sending. Reject retention settings
for disabled configurations before sending. This is a caller assertion, not an
update toggle: the route has no `lock_enabled` parameter. Do not imply that
validation checks the live bucket state without a read. Ceph's locking helper
rejects omitted settings before writing retention, but ownership or versioning
may already have changed. Document partial failure behavior and validate the
known conflict between enabled Object Lock and suspended versioning locally.
Integration coverage must check rejection without mutation, explicit retention
of settings, and deliberate encryption/lifecycle deletion.

## Dashboard pool read verification

Verified against Ceph v20.2.4's `Pool._serialize_pool`, `_pool_list`, and `get`
in `src/pybind/mgr/dashboard/controllers/pool.py`, its REST routing in
`controllers/_rest_controller.py`, and `services/ceph_service.py`
(`get_pool_list` and `get_pool_list_with_stats`), relative to
`src/pybind/mgr/dashboard` for the latter two paths. The frontend's
`frontend/src/app/shared/api/pool.service.ts` and
`qa/tasks/mgr/dashboard/test_pool.py` exercise these reads.

Ceph's `attrs` parameter omits ordinary fields, leaving only `pool_name`
mandatory. Keep it out of `ListPoolsRequest` and `GetPoolRequest` so callers
cannot mistake unrequested fields for actual zero values in `Pool`.
Retain the statistics choice: `Stats` and `PGStatus` are nullable maps when
not requested. `Configuration` and `ScheduleInfo` are added only by `GetPool`
and explicitly represent their absence in list results. Integration coverage
must check ordinary fields and statistics presence with both statistics choices.
