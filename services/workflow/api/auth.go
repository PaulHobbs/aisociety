// Package api provides the workflow service API and authentication/authorization logic.
package api

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Role represents a user role for authorization.
type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

// Dummy in-memory token-to-role mapping for demo/testing.
var tokenRoleMap = map[string]Role{
	"admin-token": RoleAdmin,
	"user-token":  RoleUser,
}

// methodPermissions maps gRPC method names to required roles.
var methodPermissions = map[string]Role{
	"/protos.WorkflowService/CreateWorkflow": RoleAdmin,
	"/protos.WorkflowService/UpdateWorkflow": RoleAdmin,
	"/protos.WorkflowService/UpdateNode":     RoleAdmin,
	// Read-only endpoints can be accessed by any authenticated user.
	"/protos.WorkflowService/GetWorkflow":   RoleUser,
	"/protos.WorkflowService/ListWorkflows": RoleUser,
	"/protos.WorkflowService/GetNode":       RoleUser,
}

// AuthInterceptor is a gRPC unary interceptor for authentication and authorization.
func AuthInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	role, err := authenticate(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "authentication failed: "+err.Error())
	}
	requiredRole, ok := methodPermissions[info.FullMethod]
	if !ok {
		// Default: require admin for unknown methods
		requiredRole = RoleAdmin
	}
	if !authorize(role, requiredRole) {
		return nil, status.Error(codes.PermissionDenied, "permission denied")
	}
	return handler(ctx, req)
}

// authenticate extracts and validates the token from metadata, returning the user's role.
func authenticate(ctx context.Context) (Role, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing metadata")
	}
	authHeaders := md["authorization"]
	if len(authHeaders) == 0 {
		return "", status.Error(codes.Unauthenticated, "missing authorization header")
	}
	token := parseBearerToken(authHeaders[0])
	role, ok := tokenRoleMap[token]
	if !ok {
		return "", status.Error(codes.Unauthenticated, "invalid token")
	}
	return role, nil
}

// parseBearerToken extracts the token from a "Bearer ..." header.
func parseBearerToken(header string) string {
	if strings.HasPrefix(header, "Bearer ") {
		return strings.TrimPrefix(header, "Bearer ")
	}
	return header
}

// authorize checks if the user's role meets the required role.
func authorize(userRole Role, required Role) bool {
	if userRole == RoleAdmin {
		return true
	}
	return userRole == required
}
