package integration

import (
	"context"
	"testing"

	mgr "github.com/sj14/ceph-go/mgr"
	"github.com/sj14/ceph-go/test/ceph/testutil"
)

func integrationClient(t *testing.T) *mgr.Client {
	return testutil.MGRClient(t)
}

func integrationContext(t *testing.T) context.Context {
	return testutil.Context(t)
}

func uniqueResourceName(t *testing.T, prefix string) string {
	return testutil.UniqueResourceName(t, prefix)
}
