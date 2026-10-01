package integration

import (
	"bytes"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/sj14/ceph-go/rgw"
)

func TestRGWCheckBucketIndex(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture := createRGWBucketFixture(t, client, ctx)
	result, err := client.CheckBucketIndex(ctx, rgw.CheckBucketIndexRequest{Name: fixture.name})
	if err != nil {
		t.Fatal(err)
	}
	if result.InvalidMultipartEntries == nil || result.Result.Existing.Usage == nil ||
		result.Result.Calculated.Usage == nil || result.Objects != nil {
		t.Fatalf("Admin Ops bucket index check = %#v", result)
	}
}

func TestRGWRepairBucketIndex(t *testing.T) {
	t.Parallel()
	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture := createRGWBucketFixture(t, client, ctx)
	objects := []string{"first-object", "second-object"}
	payload := []byte("index repair integration test")
	for _, object := range objects {
		requireS3Success(t, ctx, http.MethodPut, fixture.name+"/"+object, payload)
	}
	for _, name := range []string{"", " "} {
		_, checkErr := client.CheckBucketIndex(ctx, rgw.CheckBucketIndexRequest{Name: name})
		_, repairErr := client.RepairBucketIndex(ctx, rgw.RepairBucketIndexRequest{Name: name, CheckObjects: true})
		for _, err := range []error{checkErr, repairErr} {
			var apiErr *rgw.APIError
			if err == nil || errors.As(err, &apiErr) {
				t.Fatalf("missing bucket error = %v, want local validation error", err)
			}
		}
	}
	for _, checkObjects := range []bool{false, true} {
		result, err := client.RepairBucketIndex(ctx, rgw.RepairBucketIndexRequest{
			Name: fixture.name, CheckObjects: checkObjects,
		})
		if err != nil {
			t.Fatal(err)
		}
		if result.InvalidMultipartEntries == nil || len(result.InvalidMultipartEntries) != 0 ||
			result.Result.Existing.Usage == nil || result.Result.Calculated.Usage == nil ||
			(result.Objects != nil) != checkObjects {
			t.Fatalf("repair report with CheckObjects=%v: %#v", checkObjects, result)
		}
		for _, object := range objects {
			if checkObjects && !bytes.Contains(result.Objects, []byte(object)) {
				t.Fatalf("object reconciliation report %s omitted %q", result.Objects, object)
			}
			status, body := s3Request(t, ctx, http.MethodGet, fixture.name+"/"+object, nil)
			if status != http.StatusOK || !bytes.Equal(body, payload) {
				t.Fatalf("object %q after repair: status %d, body %q", object, status, body)
			}
		}
		checked, err := client.CheckBucketIndex(ctx, rgw.CheckBucketIndexRequest{Name: fixture.name})
		if err != nil {
			t.Fatal(err)
		}
		if checked.Objects != nil || !reflect.DeepEqual(checked.Result.Existing, checked.Result.Calculated) {
			t.Fatalf("index after repair = %#v", checked)
		}
		var count int64
		for _, usage := range checked.Result.Calculated.Usage {
			count += usage.NumObjects
		}
		if count != int64(len(objects)) {
			t.Fatalf("index object count after repair = %d, want %d", count, len(objects))
		}
	}
}
