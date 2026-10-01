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
- Keep READMEs focused on usage, observable behavior, and migration guidance.
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
  for a real zero value. Prefer clear, hard-to-misuse APIs over preserving a
  confusing interface; breaking changes are acceptable when needed for this.
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
- Key creation: `src/rgw/driver/rados/rgw_rest_user.cc` (`RGWOp_Key_Create`)
  parses the request. In the same directory, `rgw_user.cc`
  (`RGWUserAdminOp_Key::create`) returns the complete collection of the selected
  key type, and `RGWAccessKeyPool::generate_key` derives the Swift identity from
  the user and subuser. Keep separate typed S3 and Swift methods and requests.
- Bucket listing: `src/rgw/driver/rados/rgw_rest_bucket.cc`
  (`RGWOp_Bucket_Info::execute`) does not parse an `account-id` listing filter.
  `rgw_bucket.cc` in the same directory resolves account membership from UID.
  Do not reintroduce an ineffective `AccountID` filter or a public `Stats` flag;
  the name-only and detailed methods must retain fixed return types.
- Metadata sections: registration and wire names are in
  `src/rgw/driver/rados/rgw_service.cc` and the handlers listed in
  `rgw/metadata.go`.
- Account quotas: the REST parser uses `int32` limits while stored quota
  models use `int64`; preserve the distinct request representation.
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
