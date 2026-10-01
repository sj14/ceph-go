// Package rgw provides a client for Ceph RGW's Admin Ops API.
package rgw

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
)

const (
	defaultAdminPath   = "admin"
	defaultRegion      = "default"
	defaultHTTPTimeout = 30 * time.Second
	serviceName        = "s3"
	unsignedPayload    = "UNSIGNED-PAYLOAD"
	maxResponseBody    = 4 << 20
)

// Client calls the RGW Admin Ops API directly using AWS Signature Version 4.
// A Client is safe for concurrent use.
type Client struct {
	baseURL     *url.URL
	httpClient  *http.Client
	signer      *v4.Signer
	credentials aws.Credentials
	adminPath   string
	region      string
	userAgent   string
	now         func() time.Time
}

// Option configures a Client.
type Option func(*Client) error

// WithHTTPClient configures the HTTP client used for requests. The supplied
// client replaces the default client, including its 30-second timeout.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(client *Client) error {
		if httpClient == nil {
			return errors.New("rgw: HTTP client must not be nil")
		}
		client.httpClient = httpClient
		return nil
	}
}

// WithAdminPath changes RGW's configured Admin Ops resource from its default
// value, "admin". The path is relative to the endpoint URL.
func WithAdminPath(path string) Option {
	return func(client *Client) error {
		path = strings.Trim(path, "/")
		if path == "" {
			return errors.New("rgw: admin path must not be empty")
		}
		if strings.Contains(path, "?") || strings.Contains(path, "#") {
			return errors.New("rgw: admin path must not include a query or fragment")
		}
		client.adminPath = path
		return nil
	}
}

// WithRegion changes the SigV4 region. RGW does not normally validate a
// specific region, and the default matches Ceph's own clients.
func WithRegion(region string) Option {
	return func(client *Client) error {
		if strings.TrimSpace(region) == "" {
			return errors.New("rgw: region must not be empty")
		}
		client.region = region
		return nil
	}
}

// WithUserAgent configures the User-Agent header. Empty values disable it.
func WithUserAgent(userAgent string) Option {
	return func(client *Client) error {
		client.userAgent = userAgent
		return nil
	}
}

// NewClient constructs a direct RGW Admin Ops client. endpoint is the RGW
// service URL, not the Ceph Dashboard URL. The default HTTP client has a
// 30-second total request timeout.
func NewClient(endpoint, accessKey, secretKey string, options ...Option) (*Client, error) {
	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("rgw: parse endpoint: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, errors.New("rgw: endpoint scheme must be http or https")
	}
	if parsedURL.Host == "" {
		return nil, errors.New("rgw: endpoint must include a host")
	}
	if parsedURL.RawQuery != "" || parsedURL.Fragment != "" {
		return nil, errors.New("rgw: endpoint must not include a query or fragment")
	}
	if strings.TrimSpace(accessKey) == "" {
		return nil, errors.New("rgw: access key must not be empty")
	}
	if strings.TrimSpace(secretKey) == "" {
		return nil, errors.New("rgw: secret key must not be empty")
	}

	client := &Client{
		baseURL:    parsedURL,
		httpClient: &http.Client{Timeout: defaultHTTPTimeout},
		signer:     v4.NewSigner(),
		credentials: aws.Credentials{
			AccessKeyID:     accessKey,
			SecretAccessKey: secretKey,
		},
		adminPath: defaultAdminPath,
		region:    defaultRegion,
		userAgent: "ceph-go/rgw",
		now:       time.Now,
	}
	for _, option := range options {
		if option == nil {
			return nil, errors.New("rgw: option must not be nil")
		}
		if err := option(client); err != nil {
			return nil, err
		}
	}
	return client, nil
}

func (client *Client) endpoint(resource string) *url.URL {
	endpoint := *client.baseURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/" +
		strings.Trim(client.adminPath, "/") + "/" + strings.TrimLeft(resource, "/")
	endpoint.RawPath = ""
	return &endpoint
}

func (client *Client) newRequest(ctx context.Context, method, resource string, query url.Values) (*http.Request, error) {
	return client.newRequestWithBody(ctx, method, resource, query, nil)
}

func (client *Client) newJSONRequest(ctx context.Context, method, resource string, query url.Values, input any) (*http.Request, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("rgw: encode %s %s request: %w", method, resource, err)
	}
	return client.newRequestWithBody(ctx, method, resource, query, body)
}

func (client *Client) newRequestWithBody(ctx context.Context, method, resource string, query url.Values, body []byte) (*http.Request, error) {
	if ctx == nil {
		return nil, errors.New("rgw: context must not be nil")
	}
	endpoint := client.endpoint(resource)
	if query == nil {
		query = url.Values{}
	}
	query.Set("format", "json")
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, method, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	payloadHash := unsignedPayload
	if body != nil {
		sum := sha256.Sum256(body)
		payloadHash = hex.EncodeToString(sum[:])
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("X-Amz-Content-Sha256", payloadHash)
	}
	if client.userAgent != "" {
		request.Header.Set("User-Agent", client.userAgent)
	}
	if err := client.signer.SignHTTP(
		ctx,
		client.credentials,
		request,
		payloadHash,
		serviceName,
		client.region,
		client.now(),
	); err != nil {
		return nil, fmt.Errorf("rgw: sign %s %s request: %w", method, endpoint.Path, err)
	}
	return request, nil
}

func (client *Client) do(request *http.Request) ([]byte, error) {
	body, _, err := client.doWithHeaders(request)
	return body, err
}

// doWithHeaders keeps transport metadata internal. Metadata writes decode the
// meaningful RGWX update headers into a domain result.
func (client *Client) doWithHeaders(request *http.Request) ([]byte, http.Header, error) {
	response, err := client.httpClient.Do(request)
	if err != nil {
		return nil, nil, fmt.Errorf("rgw: %s %s: %w", request.Method, request.URL.Path, err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBody+1))
	if err != nil {
		return nil, nil, fmt.Errorf("rgw: read %s %s response: %w", request.Method, request.URL.Path, err)
	}
	if len(body) > maxResponseBody {
		return nil, nil, fmt.Errorf("rgw: %s %s response exceeds %d bytes", request.Method, request.URL.Path, maxResponseBody)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		apiError := &APIError{
			StatusCode: response.StatusCode,
			Method:     request.Method,
			Path:       request.URL.Path,
			Body:       string(body),
		}
		_ = json.Unmarshal(body, apiError)
		return nil, nil, apiError
	}
	return body, response.Header, nil
}

func setString(query url.Values, name, value string) {
	if value != "" {
		query.Set(name, value)
	}
}

func setStringPointer(query url.Values, name string, value *string) {
	if value != nil {
		query.Set(name, *value)
	}
}

func setBool(query url.Values, name string, value *bool) {
	if value != nil {
		query.Set(name, strconv.FormatBool(*value))
	}
}

func setInt64(query url.Values, name string, value *int64) {
	if value != nil {
		query.Set(name, strconv.FormatInt(*value, 10))
	}
}

// ErrorCode is a stable RGW Admin Ops error code. Unknown codes remain
// available through APIError.Code and can be converted to ErrorCode by callers.
type ErrorCode string

// Verified against Ceph v20.2.4: src/rgw/rgw_common.cc
// (rgw_http_s3_errors and set_req_state_err). User/key/capability failures
// originate in src/rgw/driver/rados/rgw_user.cc; Admin Ops uses the S3 mapping.
const (
	ErrNoSuchUser         ErrorCode = "NoSuchUser"
	ErrNoSuchSubUser      ErrorCode = "NoSuchSubUser"
	ErrNoSuchBucket       ErrorCode = "NoSuchBucket"
	ErrNoSuchKey          ErrorCode = "NoSuchKey"
	ErrNoSuchEntity       ErrorCode = "NoSuchEntity"
	ErrNoSuchBucketPolicy ErrorCode = "NoSuchBucketPolicy"
	ErrNotFound           ErrorCode = "NotFound"

	ErrUserExists             ErrorCode = "UserAlreadyExists"
	ErrAccountExists          ErrorCode = "AccountAlreadyExists"
	ErrBucketExists           ErrorCode = "BucketAlreadyExists"
	ErrEmailExists            ErrorCode = "EmailExists"
	ErrKeyExists              ErrorCode = "KeyExists"
	ErrBucketNotEmpty         ErrorCode = "BucketNotEmpty"
	ErrOperationAborted       ErrorCode = "OperationAborted"
	ErrConcurrentModification ErrorCode = "ConcurrentModification"
	ErrPreconditionFailed     ErrorCode = "PreconditionFailed"
	ErrLocked                 ErrorCode = "Locked"

	ErrInvalidArgument         ErrorCode = "InvalidArgument"
	ErrInvalidRequest          ErrorCode = "InvalidRequest"
	ErrInvalidBucketName       ErrorCode = "InvalidBucketName"
	ErrInvalidObjectName       ErrorCode = "InvalidObjectName"
	ErrInvalidTenantName       ErrorCode = "InvalidTenantName"
	ErrInvalidSecretKey        ErrorCode = "InvalidSecretKey"
	ErrInvalidKeyType          ErrorCode = "InvalidKeyType"
	ErrInvalidCapability       ErrorCode = "InvalidCapability"
	ErrMalformedPolicyDocument ErrorCode = "MalformedPolicyDocument"
	ErrMissingContentLength    ErrorCode = "MissingContentLength"
	ErrMethodNotAllowed        ErrorCode = "MethodNotAllowed"

	ErrAccessDenied          ErrorCode = "AccessDenied"
	ErrAuthorizationError    ErrorCode = "AuthorizationError"
	ErrSignatureDoesNotMatch ErrorCode = "SignatureDoesNotMatch"
	ErrInvalidAccessKeyID    ErrorCode = "InvalidAccessKeyId"
	ErrUserSuspended         ErrorCode = "UserSuspended"
	ErrRequestTimeTooSkewed  ErrorCode = "RequestTimeTooSkewed"

	ErrRequestTimeout       ErrorCode = "RequestTimeout"
	ErrLimitExceeded        ErrorCode = "LimitExceeded"
	ErrQuotaExceeded        ErrorCode = "QuotaExceeded"
	ErrTooManyBuckets       ErrorCode = "TooManyBuckets"
	ErrSlowDown             ErrorCode = "SlowDown"
	ErrInsufficientCapacity ErrorCode = "InsufficientCapacity"
	ErrInternalError        ErrorCode = "InternalError"
	ErrUnknownError         ErrorCode = "UnknownError"
	ErrNotImplemented       ErrorCode = "NotImplemented"
	ErrServiceUnavailable   ErrorCode = "ServiceUnavailable"
)

func (code ErrorCode) Error() string {
	return string(code)
}

// APIError is returned for non-2xx RGW Admin Ops responses.
type APIError struct {
	StatusCode int       `json:"-"`
	Method     string    `json:"-"`
	Path       string    `json:"-"`
	Code       ErrorCode `json:"Code"`
	RequestID  string    `json:"RequestId"`
	HostID     string    `json:"HostId"`
	Body       string    `json:"-"`
}

func (err *APIError) Error() string {
	message := string(err.Code)
	if message == "" {
		message = strings.TrimSpace(err.Body)
	}
	if message == "" {
		message = http.StatusText(err.StatusCode)
	}
	return fmt.Sprintf("rgw: %s %s returned %d: %s", err.Method, err.Path, err.StatusCode, message)
}

// Is supports errors.Is with the exported ErrorCode constants.
func (err *APIError) Is(target error) bool {
	code, ok := target.(ErrorCode)
	return ok && err.Code == code
}
