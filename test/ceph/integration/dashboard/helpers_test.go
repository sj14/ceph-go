package integration

import (
	"context"
	"testing"

	rgw "github.com/sj14/rgw-go/dashboard"
	"github.com/sj14/rgw-go/test/ceph/testutil"
)

func integrationClient(t *testing.T) *rgw.Client {
	return testutil.DashboardClient(t)
}

func integrationContext(t *testing.T) context.Context {
	return testutil.Context(t)
}

func uniqueResourceName(t *testing.T, prefix string) string {
	return testutil.UniqueResourceName(t, prefix)
}
