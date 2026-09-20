# Minimal Ceph integration-test image

This image runs the smallest useful real Ceph stack for testing `rgw-go`:

- one monitor;
- one manager with the Dashboard module;
- one in-memory OSD;
- one RADOS Gateway.

It deliberately omits CephFS, NFS-Ganesha, iSCSI, RBD tools, monitoring,
cephadm, device management, and persistent storage. It is for tests only.

## Why a custom image?

No maintained, minimal image was found that includes both RGW and the Ceph
Dashboard API. The official [`quay.io/ceph/ceph`](https://quay.io/repository/ceph/ceph)
image is intentionally a general-purpose image for all Ceph daemons. The old
[`ceph-container`](https://github.com/ceph/ceph-container) project is archived,
and `ceph/demo` is an old all-in-one demo rather than a current test runtime.
MicroCeph is distributed as a
[`snap`](https://github.com/canonical/microceph), not as a small OCI image.
S3-only community images do not contain the MGR Dashboard whose REST API this
library calls.

For reference, the summed compressed `linux/amd64` layers inspected on
2026-09-19 were 636 MB for `quay.io/ceph/ceph:v21`, 543 MB for
`quay.io/ceph/ceph:v20.2.4`, and 468 MB for `quay.io/ceph/demo:latest`. This
AlmaLinux-based image measured 325 MB of local content (1.21 GB unpacked) on
the same platform. It installs only the required RPMs, disables weak
dependencies, and uses an in-memory OSD. The Dashboard and Ceph's shared
libraries impose most of the remaining lower bound.

## Run

```sh
docker compose -f test/ceph/compose.yaml up --build --wait
```

The Dashboard API is available at `http://localhost:8443`; RGW listens at
`http://localhost:8000`. The default Dashboard credentials are `admin` / `admin`
and can be overridden through the corresponding environment variables. The
ephemeral RGW user used by tests is `rgw-go-test`.

Run the real Dashboard contract tests after the container is healthy:

```sh
go test -v ./test/ceph/integration
```

The tests run by default and expect the container to be available. Use
`go test -short ./...` to run only the fast client unit tests.

They cover cluster-health reports, capacity, FSID, telemetry status and
snapshots; User Create/Get/List/Update/Delete, capabilities, quotas, rate
limits, subusers, S3 access-key and Swift-key Create/Delete, explicit zero
values, optional statistics, deletion verification; Bucket
Create/Get/List/Update/Delete, bucket rate limits; and missing-resource errors.
Independent endpoint tests run in parallel and clean up their mutable fixtures.
`ListUsers` stays serial because Ceph resolves list entries non-atomically and
can otherwise race with user deletion. In particular, the tests pin Ceph
Dashboard `v20.2.4`'s observed behavior of wrapping RGW `NoSuchUser` and
`NoSuchBucket` responses in HTTP 500 errors.

The small `httptest` suite checks shared Go client behavior such as headers,
error mapping, response limits, and option validation. Endpoint contracts are
not duplicated with synthetic responses: the opt-in integration suite checks
them against real Ceph. Its tests are split by controller family in
`test/ceph/integration/*_test.go`, with one top-level test per endpoint. Mutable
resources use cryptographically random name suffixes and are deleted during
test cleanup; the whole cluster remains ephemeral.

Stop and remove the ephemeral cluster with:

```sh
docker compose -f test/ceph/compose.yaml down
```

## GitHub Actions

The `Publish Ceph test image` workflow builds this image from `main` only when
its build inputs change or when it is started manually. It publishes a
commit-specific candidate to this repository owner's GitHub Container Registry
package and runs the complete integration suite against its immutable digest.
Only a successful candidate is promoted to the moving `main` tag.

For the initial setup:

1. Run `Publish Ceph test image` once from the Actions page.
2. Verify that its integration tests and promotion step succeed.
3. Re-run an initially failed `CI` workflow now that the `main` image exists.

The repository may remain private. Both workflows authenticate to GHCR with
their short-lived `GITHUB_TOKEN`; the publisher has `packages: write` and the
integration job has only `packages: read`. Ensure that the package inherits
access from this repository if its package settings were changed manually.

The regular `CI` workflow always pulls `rgw-go-ceph-test:main`. It starts
Compose with `--no-build`, so ordinary CI runs can never rebuild Ceph. The
moving tag always identifies the newest candidate that completed the real Ceph
integration suite; no image version or repository variable needs manual
maintenance. Making the repository and package public later permits
unauthenticated pulls, although retaining the login step is harmless.

The runtime packages and endpoint contracts use Ceph's latest stable release,
`v20.2.4`, tag commit `7f793731f1b39eb4f465e960113d2363c311b964`.
The release RPM normally configures the moving `rpm-tentacle` channel, so the
build rewrites that URL to the immutable `rpm-20.2.4` repository. The release
RPM checksum and AlmaLinux base-image digest are pinned as well.

This deliberately is not a production Ceph deployment: all cluster data lives
on tmpfs, replication is disabled, SSL is disabled, credentials are test
defaults, and one process supervises all daemons.

## Updating Ceph

Determine the latest stable release from Ceph's source-controlled release
metadata and validate its `src/ceph_release` marker with:

```sh
test/ceph/latest-stable.sh
```

Update `CEPH_RELEASE` and the release-RPM checksum deliberately; do not make the
container build follow that result automatically, because test images should
remain reproducible between builds.
