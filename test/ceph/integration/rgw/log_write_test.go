package integration

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/sj14/ceph-go/rgw"
)

type metadataLogLockFixture struct {
	client  *rgw.Client
	request rgw.LockMetadataLogRequest
	locked  bool
}

func createMetadataLogLockFixture(t *testing.T) *metadataLogLockFixture {
	t.Helper()
	fixture := &metadataLogLockFixture{
		client: rgwIntegrationClient(t),
		request: rgw.LockMetadataLogRequest{
			ShardID: 0, Period: uniqueResourceName(t, "ceph-go-lock-period"),
			LockerID: uniqueResourceName(t, "locker"), ZoneID: uniqueResourceName(t, "zone"),
			LengthSeconds: 60,
		},
	}
	// Locks operate on a log object named by period, without requiring a real
	// realm or period. A unique period keeps this lock away from live sync logs.
	if err := fixture.client.LockMetadataLog(integrationContext(t), fixture.request); err != nil {
		t.Fatal(err)
	}
	fixture.locked = true
	t.Cleanup(func() {
		if !fixture.locked {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := fixture.client.UnlockMetadataLog(ctx, fixture.unlockRequest()); err != nil {
			t.Errorf("cleanup metadata-log lock: %v", err)
		}
	})
	return fixture
}

func (fixture *metadataLogLockFixture) unlockRequest() rgw.UnlockMetadataLogRequest {
	return rgw.UnlockMetadataLogRequest{
		ShardID: fixture.request.ShardID, Period: fixture.request.Period,
		LockerID: fixture.request.LockerID, ZoneID: fixture.request.ZoneID,
	}
}

func TestRGWLockMetadataLog(t *testing.T) {
	t.Parallel()

	fixture := createMetadataLogLockFixture(t)
	ctx := integrationContext(t)
	if err := fixture.client.LockMetadataLog(ctx, fixture.request); err != nil {
		t.Fatalf("renewing metadata-log lock: %v", err)
	}
	other := fixture.request
	other.LockerID = uniqueResourceName(t, "competing-locker")
	err := fixture.client.LockMetadataLog(ctx, other)
	requireRGWAPIStatus(t, err, http.StatusLocked)
}

func TestRGWUnlockMetadataLog(t *testing.T) {
	t.Parallel()

	fixture := createMetadataLogLockFixture(t)
	ctx := integrationContext(t)
	if err := fixture.client.UnlockMetadataLog(ctx, fixture.unlockRequest()); err != nil {
		t.Fatal(err)
	}
	fixture.locked = false
	fixture.request.LockerID = uniqueResourceName(t, "replacement-locker")
	if err := fixture.client.LockMetadataLog(ctx, fixture.request); err != nil {
		t.Fatalf("locking released metadata log: %v", err)
	}
	fixture.locked = true
}

func TestRGWNotifyMetadataLog(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	if err := client.NotifyMetadataLog(integrationContext(t), rgw.NotifyMetadataLogRequest{
		UpdatedShards: []int64{0, 1},
	}); err != nil {
		t.Fatal(err)
	}
	if err := client.NotifyMetadataLog(integrationContext(t), rgw.NotifyMetadataLogRequest{}); err != nil {
		t.Fatalf("empty metadata notification: %v", err)
	}
}

func TestRGWTrimMetadataLog(t *testing.T) {
	t.Parallel()

	fixture := createMetadataLogLockFixture(t)
	// The isolated object has no log entries. Ceph v20.2.4's cls_log_trim returns
	// -ENODATA for an empty range, which RGW maps to 500 UnknownError rather than
	// treating it as success (src/cls/log/cls_log.cc, src/rgw/rgw_rest_log.cc).
	err := fixture.client.TrimMetadataLog(integrationContext(t), rgw.TrimMetadataLogRequest{
		ShardID: fixture.request.ShardID, Period: fixture.request.Period, Marker: "z",
	})
	requireRGWAPIStatus(t, err, http.StatusInternalServerError)
}

func TestRGWTrimBucketIndexLog(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture := createRGWBucketFixture(t, client, ctx)
	// Single-site RGW does not log ordinary S3 writes for replication. Stop and
	// restart this fixture's sync to create real stop/resync log entries.
	for _, enabled := range []bool{false, true} {
		if err := client.SetBucketSync(ctx, rgw.SetBucketSyncRequest{Name: fixture.name, Sync: &enabled}); err != nil {
			t.Fatal(err)
		}
	}
	requireS3Success(t, ctx, http.MethodPut, fixture.name+"/log-entry", []byte("test log entry"))
	bucket, err := client.GetBucket(ctx, rgw.GetBucketRequest{Name: fixture.name})
	if err != nil {
		t.Fatal(err)
	}
	instance := fixture.name + ":" + bucket.ID
	before, err := client.ListBucketIndexLogEntries(ctx, rgw.ListBucketIndexLogEntriesRequest{
		Bucket: fixture.name, BucketInstance: instance, Generation: new(int64(0)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(before.Entries) == 0 {
		t.Fatal("sync toggle did not produce a bucket-index log entry")
	}
	if err := client.TrimBucketIndexLog(ctx, rgw.TrimBucketIndexLogRequest{
		Bucket: fixture.name, BucketInstance: instance, EndMarker: "z", Generation: new(int64(0)),
	}); err != nil {
		t.Fatal(err)
	}
	entries, err := client.ListBucketIndexLogEntries(ctx, rgw.ListBucketIndexLogEntriesRequest{
		Bucket: fixture.name, BucketInstance: instance, Generation: new(int64(0)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries.Entries) != 0 || entries.Truncated {
		t.Fatalf("bucket-index log entries after trim = %#v", entries)
	}
	// Trimming the sync log does not remove the bucket's actual S3 objects.
	requireS3Success(t, ctx, http.MethodGet, fixture.name+"/log-entry", nil)
}

func TestRGWNotifyDataLog(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	if err := client.NotifyDataLog(integrationContext(t), rgw.NotifyDataLogRequest{
		SourceZone: uniqueResourceName(t, "source-zone"),
		Shards:     []rgw.DataLogShardNotification{{ShardID: 0, Keys: []string{"test-bucket:instance:0"}}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := client.NotifyDataLog(integrationContext(t), rgw.NotifyDataLogRequest{}); err != nil {
		t.Fatalf("empty data notification: %v", err)
	}
}

func TestRGWNotifyDataLogV2(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	if err := client.NotifyDataLogV2(integrationContext(t), rgw.NotifyDataLogV2Request{
		SourceZone: uniqueResourceName(t, "source-zone"),
		Shards: []rgw.DataLogShardNotificationV2{{
			ShardID: 0, Entries: []rgw.DataLogNotificationEntry{{Key: "test-bucket:instance:0", Generation: 7}},
		}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := client.NotifyDataLogV2(integrationContext(t), rgw.NotifyDataLogV2Request{}); err != nil {
		t.Fatalf("empty v2 data notification: %v", err)
	}
}

func TestRGWTrimDataLog(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	info, err := client.GetDataLogInfo(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// Data-log shards are global. Exercise the backend's shard boundary check
	// without removing logs belonging to other tests or live replication.
	err = client.TrimDataLog(ctx, rgw.TrimDataLogRequest{ShardID: info.NumShards, Marker: "z"})
	requireRGWAPIStatus(t, err, http.StatusBadRequest)
}
