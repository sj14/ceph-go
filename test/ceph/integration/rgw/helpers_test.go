package integration

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/sj14/ceph-go/rgw"
	"github.com/sj14/ceph-go/test/ceph/testutil"
)

func rgwIntegrationClient(t *testing.T) *rgw.Client {
	return testutil.RGWClient(t)
}

func integrationContext(t *testing.T) context.Context {
	return testutil.Context(t)
}

func uniqueResourceName(t *testing.T, prefix string) string {
	return testutil.UniqueResourceName(t, prefix)
}

func s3Request(t *testing.T, ctx context.Context, method, resource string, body []byte) (int, []byte) {
	t.Helper()
	status, responseBody, err := testutil.S3Do(ctx, method, resource, body)
	if err != nil {
		t.Fatal(err)
	}
	return status, responseBody
}

func requireS3Success(t *testing.T, ctx context.Context, method, resource string, body []byte) {
	t.Helper()
	status, responseBody := s3Request(t, ctx, method, resource, body)
	if status < http.StatusOK || status >= http.StatusMultipleChoices {
		t.Fatal(fmt.Errorf("S3 %s /%s returned %d: %s", method, resource, status, responseBody))
	}
}
