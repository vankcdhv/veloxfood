// Package remotechecker implements the auth middleware PermissionChecker by
// delegating to the user-service over gRPC (single source of truth for RBAC).
// Non-user services use this so they don't replicate RBAC tables locally.
package remotechecker

import (
	"context"
	"fmt"

	userv1 "project/proto/user/v1"
)

// RemoteChecker satisfies middleware.PermissionChecker via user-service gRPC.
type RemoteChecker struct {
	client userv1.UserServiceClient
}

func New(client userv1.UserServiceClient) *RemoteChecker {
	return &RemoteChecker{client: client}
}

// HasPermission checks a global permission against the user-service RBAC.
func (c *RemoteChecker) HasPermission(ctx context.Context, userID, code string) (bool, error) {
	resp, err := c.client.CheckPermission(ctx, &userv1.CheckPermissionRequest{
		UserId:         userID,
		PermissionCode: code,
	})
	if err != nil {
		return false, fmt.Errorf("remote permission check: %w", err)
	}
	return resp.GetHasPermission(), nil
}

// HasVendorPermission checks a vendor-scoped permission against the user-service RBAC.
func (c *RemoteChecker) HasVendorPermission(ctx context.Context, userID, vendorID, code string) (bool, error) {
	resp, err := c.client.CheckVendorPermission(ctx, &userv1.CheckVendorPermissionRequest{
		UserId:         userID,
		VendorId:       vendorID,
		PermissionCode: code,
	})
	if err != nil {
		return false, fmt.Errorf("remote vendor permission check: %w", err)
	}
	return resp.GetHasPermission(), nil
}
