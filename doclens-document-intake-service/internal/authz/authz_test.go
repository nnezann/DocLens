package authz

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestRequireRoleAndOrganization(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"x-organization-id", "org_1",
		"x-roles", "analyst",
	))
	if err := Require(ctx, "org_1", PermissionUpload); err != nil {
		t.Fatalf("expected analyst upload to be allowed: %v", err)
	}
	if err := Require(ctx, "org_2", PermissionUpload); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected organization mismatch to be denied, got %v", err)
	}
}

func TestRequireRejectsUnknownRole(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"x-organization-id", "org_1",
		"x-roles", "unknown",
	))
	if err := Require(ctx, "org_1", PermissionRead); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected unknown role to be denied, got %v", err)
	}
}
