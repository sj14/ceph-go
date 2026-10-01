package integration

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	mgr "github.com/sj14/ceph-go/mgr"
)

type bucketFixture struct {
	client  *mgr.Client
	name    string
	deleted bool
}

func createBucketFixture(t *testing.T, client *mgr.Client, ctx context.Context) *bucketFixture {
	t.Helper()
	return createBucketFixtureWithSettings(t, client, ctx, mgr.CreateBucketRequest{})
}

func createBucketFixtureWithSettings(t *testing.T, client *mgr.Client, ctx context.Context, input mgr.CreateBucketRequest) *bucketFixture {
	t.Helper()
	fixture := &bucketFixture{
		client: client,
		name:   uniqueResourceName(t, "ceph-go-integration-bucket"),
	}
	input.Name = fixture.name
	input.UID = "ceph-go-test"
	if err := client.CreateBucket(ctx, input); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if fixture.deleted {
			return
		}
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if err := fixture.client.DeleteBucket(cleanupCtx, mgr.DeleteBucketRequest{Name: fixture.name}); err != nil {
			t.Errorf("cleanup bucket %q: %v", fixture.name, err)
		}
	})
	return fixture
}

func TestUpdateBucketObjectLock(t *testing.T) {
	t.Parallel()
	client := integrationClient(t)
	ctx := integrationContext(t)
	fixture := createBucketFixtureWithSettings(t, client, ctx, mgr.CreateBucketRequest{
		LockEnabled: true, LockMode: mgr.LockModeGovernance, LockRetentionDays: new(int64(2)),
	})
	before, err := client.GetBucket(ctx, mgr.GetBucketRequest{Name: fixture.name})
	if err != nil {
		t.Fatal(err)
	}
	if !before.LockEnabled || before.LockMode != mgr.LockModeGovernance ||
		before.LockRetentionDays == nil || *before.LockRetentionDays != 2 {
		t.Fatalf("initial Object Lock configuration = %#v", before)
	}
	for _, test := range []struct {
		name       string
		lock       *mgr.BucketObjectLockConfiguration
		versioning mgr.BucketVersioningState
	}{
		{name: "omitted configuration"},
		{name: "omitted enabled state", lock: &mgr.BucketObjectLockConfiguration{}},
		{name: "missing mode", lock: &mgr.BucketObjectLockConfiguration{Enabled: new(true), RetentionDays: new(int64(2))}},
		{name: "invalid mode", lock: &mgr.BucketObjectLockConfiguration{Enabled: new(true), Mode: "invalid", RetentionDays: new(int64(2))}},
		{name: "missing retention", lock: &mgr.BucketObjectLockConfiguration{Enabled: new(true), Mode: mgr.LockModeGovernance}},
		{name: "both periods", lock: &mgr.BucketObjectLockConfiguration{Enabled: new(true), Mode: mgr.LockModeGovernance, RetentionDays: new(int64(2)), RetentionYears: new(int64(1))}},
		{name: "zero period", lock: &mgr.BucketObjectLockConfiguration{Enabled: new(true), Mode: mgr.LockModeGovernance, RetentionDays: new(int64(0))}},
		{name: "negative period", lock: &mgr.BucketObjectLockConfiguration{Enabled: new(true), Mode: mgr.LockModeGovernance, RetentionYears: new(int64(-1))}},
		{name: "disabled with retention", lock: &mgr.BucketObjectLockConfiguration{Enabled: new(false), Mode: mgr.LockModeGovernance, RetentionDays: new(int64(2))}},
		{name: "suspended versioning", lock: &mgr.BucketObjectLockConfiguration{Enabled: new(true), Mode: mgr.LockModeGovernance, RetentionDays: new(int64(2))}, versioning: mgr.BucketVersioningSuspended},
	} {
		t.Run(test.name, func(t *testing.T) {
			// A request reaching Ceph could change the owner before failing.
			err := client.UpdateBucket(ctx, mgr.UpdateBucketRequest{
				Name: fixture.name, BucketID: before.ID, UID: "ceph-go-admin",
				EncryptionEnabled: new(false), Lifecycle: new(""), ObjectLock: test.lock,
				VersioningState: test.versioning,
			})
			var apiError *mgr.APIError
			if err == nil || errors.As(err, &apiError) || !strings.Contains(err.Error(), "Object Lock") {
				t.Fatalf("update error = %v, want local Object Lock validation error", err)
			}
		})
	}
	unchanged, err := client.GetBucket(ctx, mgr.GetBucketRequest{Name: fixture.name})
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.Owner != before.Owner || unchanged.Versioning != before.Versioning ||
		!unchanged.LockEnabled || unchanged.LockMode != before.LockMode ||
		unchanged.LockRetentionDays == nil || *unchanged.LockRetentionDays != 2 {
		t.Fatalf("rejected updates changed bucket = %#v", unchanged)
	}
	for _, lock := range []mgr.BucketObjectLockConfiguration{
		{Enabled: new(true), Mode: mgr.LockModeGovernance, RetentionDays: new(int64(2))},
		{Enabled: new(true), Mode: mgr.LockModeCompliance, RetentionYears: new(int64(1))},
	} {
		if err := client.UpdateBucket(ctx, mgr.UpdateBucketRequest{
			Name: fixture.name, BucketID: before.ID, UID: "ceph-go-test",
			EncryptionEnabled: new(false), Lifecycle: new(""), ObjectLock: &lock,
		}); err != nil {
			t.Fatal(err)
		}
		bucket, err := client.GetBucket(ctx, mgr.GetBucketRequest{Name: fixture.name})
		if err != nil {
			t.Fatal(err)
		}
		if !bucket.LockEnabled || bucket.LockMode != lock.Mode ||
			(lock.RetentionDays != nil && (bucket.LockRetentionDays == nil || *bucket.LockRetentionDays != *lock.RetentionDays)) ||
			(lock.RetentionYears != nil && (bucket.LockRetentionYears == nil || *bucket.LockRetentionYears != *lock.RetentionYears)) {
			t.Fatalf("updated Object Lock configuration = %#v, want %#v", bucket, lock)
		}
	}
}

func TestCreateBucket(t *testing.T) {
	t.Parallel()

	client := integrationClient(t)
	ctx := integrationContext(t)
	fixture := createBucketFixture(t, client, ctx)

	bucket, err := client.GetBucket(ctx, mgr.GetBucketRequest{Name: fixture.name})
	if err != nil {
		t.Fatal(err)
	}
	if bucket.Name != fixture.name || bucket.Owner != "ceph-go-test" {
		t.Fatalf("bucket = %#v", bucket)
	}
}

func TestListBuckets(t *testing.T) {
	t.Parallel()

	client := integrationClient(t)
	ctx := integrationContext(t)
	fixture := createBucketFixture(t, client, ctx)

	buckets, err := client.ListBuckets(ctx, mgr.ListBucketsRequest{UID: "ceph-go-test"})
	if err != nil {
		t.Fatal(err)
	}
	for _, bucket := range buckets {
		if bucket.Name == fixture.name {
			if bucket.Owner != "ceph-go-test" || bucket.ID == "" {
				t.Fatalf("listed bucket = %#v", bucket)
			}
			return
		}
	}
	t.Fatalf("created bucket %q is missing from ListBuckets", fixture.name)
}

func TestGetBucket(t *testing.T) {
	t.Parallel()

	client := integrationClient(t)
	ctx := integrationContext(t)
	fixture := createBucketFixture(t, client, ctx)

	bucket, err := client.GetBucket(ctx, mgr.GetBucketRequest{Name: fixture.name})
	if err != nil {
		t.Fatal(err)
	}
	if bucket.Name != fixture.name || bucket.Owner != "ceph-go-test" || bucket.ID == "" || bucket.CreationTime == "" {
		t.Fatalf("bucket = %#v", bucket)
	}

	missing := uniqueResourceName(t, "ceph-go-missing-bucket")
	_, err = client.GetBucket(ctx, mgr.GetBucketRequest{Name: missing})
	var apiError *mgr.APIError
	if !errors.As(err, &apiError) || apiError.StatusCode != http.StatusInternalServerError ||
		!strings.Contains(apiError.Body, "NoSuchBucket") {
		t.Fatalf("GetBucket for missing bucket error = %v, want Dashboard HTTP 500 containing NoSuchBucket", err)
	}
}

func TestUpdateBucket(t *testing.T) {
	t.Parallel()

	client := integrationClient(t)
	ctx := integrationContext(t)
	bucketFixture := createBucketFixture(t, client, ctx)

	bucket, err := client.GetBucket(ctx, mgr.GetBucketRequest{Name: bucketFixture.name})
	if err != nil {
		t.Fatal(err)
	}
	// A disabled rule records a real lifecycle policy without expiring objects.
	lifecycle := `<LifecycleConfiguration><Rule><ID>ceph-go-retained-policy</ID><Prefix>ceph-go/</Prefix><Status>Disabled</Status><Expiration><Days>365</Days></Expiration></Rule></LifecycleConfiguration>`
	update := mgr.UpdateBucketRequest{
		Name:              bucketFixture.name,
		BucketID:          bucket.ID,
		UID:               "ceph-go-test",
		EncryptionEnabled: new(true),
		EncryptionType:    mgr.BucketEncryptionAES256,
		Lifecycle:         &lifecycle,
		ObjectLock:        &mgr.BucketObjectLockConfiguration{Enabled: new(false)},
	}
	if err := client.UpdateBucket(ctx, update); err != nil {
		t.Fatal(err)
	}
	bucket, err = client.GetBucket(ctx, mgr.GetBucketRequest{Name: bucketFixture.name})
	if err != nil {
		t.Fatal(err)
	}
	if bucket.Encryption != mgr.BucketEncryptionEnabled || !strings.Contains(string(bucket.Lifecycle), "ceph-go-retained-policy") {
		t.Fatalf("configured bucket encryption/lifecycle = %s/%s", bucket.Encryption, bucket.Lifecycle)
	}
	for _, test := range []struct {
		name    string
		request mgr.UpdateBucketRequest
		wantErr string
	}{
		{
			name: "omitted encryption",
			request: mgr.UpdateBucketRequest{
				Name: bucketFixture.name, BucketID: bucket.ID, UID: "ceph-go-test", Lifecycle: new("{}"),
			},
			wantErr: "encryption state must be explicitly specified",
		},
		{
			name: "omitted lifecycle",
			request: mgr.UpdateBucketRequest{
				Name: bucketFixture.name, BucketID: bucket.ID, UID: "ceph-go-test", EncryptionEnabled: new(false),
			},
			wantErr: "lifecycle policy must be explicitly specified",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := client.UpdateBucket(ctx, test.request); err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("update error = %v, want %q", err, test.wantErr)
			}
			unchanged, err := client.GetBucket(ctx, mgr.GetBucketRequest{Name: bucketFixture.name})
			if err != nil {
				t.Fatal(err)
			}
			if unchanged.Encryption != bucket.Encryption || string(unchanged.Lifecycle) != string(bucket.Lifecycle) {
				t.Fatalf("rejected update changed encryption/lifecycle: %s/%s", unchanged.Encryption, unchanged.Lifecycle)
			}
		})
	}
	// Explicitly retain both settings while updating versioning.
	update.VersioningState = mgr.BucketVersioningEnabled
	if err := client.UpdateBucket(ctx, update); err != nil {
		t.Fatal(err)
	}

	bucket, err = client.GetBucket(ctx, mgr.GetBucketRequest{Name: bucketFixture.name})
	if err != nil {
		t.Fatal(err)
	}
	if bucket.Owner != "ceph-go-test" || bucket.Versioning != mgr.BucketVersioningEnabled ||
		bucket.Encryption != mgr.BucketEncryptionEnabled || !strings.Contains(string(bucket.Lifecycle), "ceph-go-retained-policy") {
		t.Fatalf("updated bucket = %#v", bucket)
	}
	// Deliberately disable encryption and remove lifecycle using each accepted
	// deletion spelling, rather than relying on an omitted field.
	for _, emptyPolicy := range []string{"", "{}"} {
		update.EncryptionEnabled = new(false)
		update.Lifecycle = &emptyPolicy
		if err := client.UpdateBucket(ctx, update); err != nil {
			t.Fatal(err)
		}
		bucket, err = client.GetBucket(ctx, mgr.GetBucketRequest{Name: bucketFixture.name})
		if err != nil {
			t.Fatal(err)
		}
		if bucket.Encryption != mgr.BucketEncryptionDisabled || string(bucket.Lifecycle) != "null" {
			t.Fatalf("bucket encryption/lifecycle after explicit removal = %s/%s", bucket.Encryption, bucket.Lifecycle)
		}
	}
}

func TestDeleteBucket(t *testing.T) {
	t.Parallel()

	client := integrationClient(t)
	ctx := integrationContext(t)
	fixture := createBucketFixture(t, client, ctx)

	if err := client.DeleteBucket(ctx, mgr.DeleteBucketRequest{Name: fixture.name}); err != nil {
		t.Fatal(err)
	}
	fixture.deleted = true

	_, err := client.GetBucket(ctx, mgr.GetBucketRequest{Name: fixture.name})
	var apiError *mgr.APIError
	if !errors.As(err, &apiError) || apiError.StatusCode != http.StatusInternalServerError ||
		!strings.Contains(apiError.Body, "NoSuchBucket") {
		t.Fatalf("GetBucket after delete error = %v, want Dashboard HTTP 500 containing NoSuchBucket", err)
	}
}
