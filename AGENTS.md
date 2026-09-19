# Repository instructions

This repository provides a Go client for Ceph's RGW management endpoints exposed by the Ceph Dashboard API.

## Source of truth

- Do not rely on the published Ceph API documentation when implementing or changing an endpoint.
- Before implementing an endpoint, identify the latest Ceph release tag and inspect the implementation in that tag of <https://github.com/ceph/ceph>.
- Inspect the controller, routing/versioning code, the internal service called by the controller, and the Ceph frontend client or tests when available.
- Record the verified Ceph release and relevant source files in code comments, tests, or the README so that later updates can be compared deliberately.
- Treat Ceph's release source code as authoritative when it differs from generated or published documentation.

## Go implementation

- Keep the public API idiomatic and context-aware.
- Preserve Ceph's actual HTTP method, route, media type, parameter location, parameter names, and response status.
- Add request-level tests with `httptest` for every endpoint.
- Use the Go standard library unless an external dependency has a clear benefit.
- Run `gofmt`, `go test ./...`, and `go vet ./...` after changes.
