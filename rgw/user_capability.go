package rgw

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type AddUserCapabilitiesRequest struct {
	UID          string
	Capabilities []Capability
}

type DeleteUserCapabilitiesRequest struct {
	UID          string
	Capabilities []Capability
}

// AddUserCapabilities adds one or more grants through PUT /admin/user?caps
// and returns the user's complete capability set.
func (client *Client) AddUserCapabilities(ctx context.Context, input AddUserCapabilitiesRequest) ([]Capability, error) {
	if ctx == nil {
		return nil, errors.New("rgw: context must not be nil")
	}
	query, err := userCapabilitiesQuery(input.UID, input.Capabilities)
	if err != nil {
		return nil, err
	}
	return client.userCapabilitiesRequest(ctx, http.MethodPut, query)
}

// DeleteUserCapabilities removes one or more grants through
// DELETE /admin/user?caps and returns the remaining capability set.
func (client *Client) DeleteUserCapabilities(ctx context.Context, input DeleteUserCapabilitiesRequest) ([]Capability, error) {
	if ctx == nil {
		return nil, errors.New("rgw: context must not be nil")
	}
	query, err := userCapabilitiesQuery(input.UID, input.Capabilities)
	if err != nil {
		return nil, err
	}
	return client.userCapabilitiesRequest(ctx, http.MethodDelete, query)
}

func userCapabilitiesQuery(uid string, capabilities []Capability) (url.Values, error) {
	if strings.TrimSpace(uid) == "" {
		return nil, errors.New("rgw: user UID must not be empty")
	}
	if len(capabilities) == 0 {
		return nil, errors.New("rgw: capabilities must not be empty")
	}
	values := make([]string, 0, len(capabilities))
	for _, capability := range capabilities {
		if capability.Type == "" || capability.Permission == "" {
			return nil, errors.New("rgw: capability type and permission must not be empty")
		}
		values = append(values, string(capability.Type)+"="+string(capability.Permission))
	}
	return url.Values{
		"caps":      {""},
		"uid":       {uid},
		"user-caps": {strings.Join(values, ";")},
	}, nil
}

func (client *Client) userCapabilitiesRequest(ctx context.Context, method string, query url.Values) ([]Capability, error) {
	body, request, err := client.userRawRequest(ctx, method, query)
	if err != nil {
		return nil, err
	}
	var capabilities []Capability
	if err := json.Unmarshal(body, &capabilities); err != nil {
		return nil, fmt.Errorf("rgw: decode %s %s response: %w", request.Method, request.URL.Path, err)
	}
	return capabilities, nil
}
