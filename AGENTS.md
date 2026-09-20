# Repository instructions

This repository provides a Go client for Ceph's RGW management endpoints exposed by the Ceph Dashboard API.

## Source of truth

- Do not rely on the published Ceph API documentation when implementing or changing an endpoint.
- Before implementing an endpoint, identify the latest stable Ceph release from `doc/releases/releases.yml` in Ceph's `main` branch, considering only the `releases` section and excluding `development`.
- Confirm that the selected tag's `src/ceph_release` file declares `stable`. Do not treat the numerically highest Git tag or the mere presence of downloadable packages as proof of a stable release; Ceph also publishes development and release-candidate tags and artifacts.
- Inspect the implementation in that stable tag of <https://github.com/ceph/ceph>.
- Inspect the controller, routing/versioning code, the internal service called by the controller, and the Ceph frontend client or tests when available.
- Record the verified Ceph release and relevant source files in code comments, tests, or the README so that later updates can be compared deliberately.
- Treat Ceph's release source code as authoritative when it differs from generated or published documentation.

## Go implementation

- Keep the public API idiomatic and context-aware.
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
- Do not duplicate Ceph's API behavior or test suite with synthetic response
  fixtures or mocks of Ceph internals.
- Use the Go standard library unless an external dependency has a clear benefit.
- Run `gofmt`, `go test -short ./...`, and `go vet ./...` after changes. When
  the Ceph container is available, also run `go test ./test/ceph/integration/...`;
  integration tests use `testing.Short()` as their only skip mechanism.

## CI and Ceph test image

- Reference GitHub Actions by their major-version tag so compatible minor and
  patch releases are adopted automatically; do not pin actions to commit hashes.
- Keep ordinary CI on the prebuilt `rgw-go-ceph-test:main` image and start it
  with `docker compose --no-build --pull always`. Do not rebuild Ceph for each
  library change.
- Build the Ceph image from `main` only when its build inputs change or when the
  image workflow is started manually. Test the candidate by immutable digest
  before promoting it to the moving `main` tag.
- Keep local Compose usage able to build the image directly when
  `CEPH_TEST_IMAGE` is not set.
