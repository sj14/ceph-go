package integration

import (
	"net/http"
	"testing"

	"github.com/sj14/rgw-go/admin"
)

func TestAdminDeleteObject(t *testing.T) {
	t.Parallel()

	client := adminIntegrationClient(t)
	ctx := integrationContext(t)
	fixture := createAdminBucketFixture(t, client, ctx)
	object := "admin-object"
	requireS3Success(t, ctx, http.MethodPut, fixture.name+"/"+object, []byte("content"))
	if err := client.DeleteObject(ctx, admin.DeleteObjectRequest{
		Bucket: fixture.name, Object: object,
	}); err != nil {
		t.Fatal(err)
	}
	status, _ := s3Request(t, ctx, http.MethodGet, fixture.name+"/"+object, nil)
	if status != http.StatusNotFound {
		t.Fatalf("S3 GET after Admin Ops object delete returned %d, want 404", status)
	}
}
