package authz

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Permission string

const (
	PermissionRead   Permission = "documents:read"
	PermissionCreate Permission = "documents:create"
	PermissionUpload Permission = "documents:upload"
)

func Require(ctx context.Context, organizationID string, permission Permission) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "authenticated gateway metadata is required")
	}
	claimedOrganization := strings.TrimSpace(first(md.Get("x-organization-id")))
	if claimedOrganization == "" || claimedOrganization != strings.TrimSpace(organizationID) {
		return status.Error(codes.PermissionDenied, "organization scope does not match authenticated identity")
	}
	roles := md.Get("x-roles")
	if len(roles) == 0 {
		return status.Error(codes.PermissionDenied, "authenticated roles are required")
	}
	for _, role := range roles {
		for _, value := range strings.Split(role, ",") {
			if grants(strings.ToLower(strings.TrimSpace(value)), permission) {
				return nil
			}
		}
	}
	return status.Error(codes.PermissionDenied, "role is not allowed for this operation")
}

func first(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func grants(role string, permission Permission) bool {
	switch permission {
	case PermissionRead:
		return role == "platform_admin" || role == "org_admin" || role == "reviewer" || role == "analyst" || role == "user" || role == "member"
	case PermissionCreate, PermissionUpload:
		return role == "platform_admin" || role == "org_admin" || role == "reviewer" || role == "analyst" || role == "user" || role == "member"
	default:
		return false
	}
}
