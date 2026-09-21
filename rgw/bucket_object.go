package rgw

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
)

type DeleteObjectRequest struct {
	Bucket string
	Object string
}

// DeleteObject deletes an object through DELETE /admin/bucket?object.
func (client *Client) DeleteObject(ctx context.Context, input DeleteObjectRequest) error {
	if ctx == nil {
		return errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(input.Bucket) == "" {
		return errors.New("rgw: bucket name must not be empty")
	}
	if strings.TrimSpace(input.Object) == "" {
		return errors.New("rgw: object name must not be empty")
	}
	query := url.Values{
		"bucket": {input.Bucket},
		"object": {input.Object},
	}
	return client.bucketAction(ctx, http.MethodDelete, query)
}
