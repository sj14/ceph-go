package testutil

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/sj14/rgw-go/admin"
	"github.com/sj14/rgw-go/dashboard"
)

const dashboardMediaType = "application/vnd.ceph.api.v1.0+json"

func AdminClient(t *testing.T) *admin.Client {
	t.Helper()
	skipShort(t)
	client, err := admin.NewClient(
		Environment("CEPH_RGW_URL", "http://127.0.0.1:8000"),
		Environment("CEPH_ADMIN_RGW_ACCESS_KEY", "RGWGOADMINACCESSKEY"),
		Environment("CEPH_ADMIN_RGW_SECRET_KEY", "rgw-go-admin-secret-key-for-integration-tests"),
	)
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func DashboardClient(t *testing.T) *dashboard.Client {
	t.Helper()
	skipShort(t)
	baseURL := Environment("CEPH_DASHBOARD_URL", "http://127.0.0.1:8443")
	token, err := authenticate(
		baseURL,
		Environment("CEPH_DASHBOARD_USER", "admin"),
		Environment("CEPH_DASHBOARD_PASSWORD", "admin"),
	)
	if err != nil {
		t.Fatal(err)
	}
	client, err := dashboard.NewClient(baseURL, dashboard.WithBearerToken(token))
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func Context(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func UniqueResourceName(t *testing.T, prefix string) string {
	t.Helper()
	var suffix [8]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("%s-%x", prefix, suffix)
}

func Environment(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func S3Do(ctx context.Context, method, resource string, body []byte) (int, []byte, error) {
	endpoint := strings.TrimRight(Environment("CEPH_RGW_URL", "http://127.0.0.1:8000"), "/") + "/" + strings.TrimLeft(resource, "/")
	request, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	payloadSum := sha256.Sum256(body)
	payloadHash := hex.EncodeToString(payloadSum[:])
	request.Header.Set("X-Amz-Content-Sha256", payloadHash)
	credentials := aws.Credentials{
		AccessKeyID:     Environment("CEPH_ADMIN_RGW_ACCESS_KEY", "RGWGOADMINACCESSKEY"),
		SecretAccessKey: Environment("CEPH_ADMIN_RGW_SECRET_KEY", "rgw-go-admin-secret-key-for-integration-tests"),
	}
	if err := v4.NewSigner().SignHTTP(ctx, credentials, request, payloadHash, "s3", "default", time.Now()); err != nil {
		return 0, nil, err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return 0, nil, err
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return 0, nil, err
	}
	return response.StatusCode, responseBody, nil
}

func skipShort(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping Ceph integration test in short mode")
	}
}

func authenticate(baseURL, username, password string) (string, error) {
	body, err := json.Marshal(map[string]string{"username": username, "password": password})
	if err != nil {
		return "", err
	}
	request, err := http.NewRequest(http.MethodPost, strings.TrimRight(baseURL, "/")+"/api/auth", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	request.Header.Set("Accept", dashboardMediaType)
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("authentication returned %s", response.Status)
	}
	var result struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.Token == "" {
		return "", errors.New("authentication response contains no token")
	}
	return result.Token, nil
}
