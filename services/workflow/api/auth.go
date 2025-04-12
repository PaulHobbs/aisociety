// Package api provides the workflow service API and authentication/authorization logic.
package api

import (
	"context"
	"encoding/json"
	"log"
	"os"
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

var tokenRoleMap = loadTokenRoleMapFromEnv()

// loadTokenRoleMapFromEnv loads the token-to-role mapping from the WORKFLOW_API_TOKENS environment variable.
// The variable should be a comma-separated list of role:token pairs, e.g. "admin:supersecrettoken,user:othertoken".
// Returns an empty map if the environment variable is not set or is invalid.
func loadTokenRoleMapFromEnv() map[string]Role {
	tokensRaw := getenv("WORKFLOW_API_TOKENS")
	m := make(map[string]Role)

	if strings.TrimSpace(tokensRaw) == "" {
		log.Printf("[WARN] WORKFLOW_API_TOKENS environment variable is missing or empty")
		return m
	}

	pairs := strings.Split(tokensRaw, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		parts := strings.SplitN(pair, ":", 2)
		if len(parts) != 2 {
			log.Printf("[WARN] Malformed token-role pair in WORKFLOW_API_TOKENS: '%s' (expected format 'role:token'), skipping", pair)
			continue
		}
		role := strings.TrimSpace(parts[0])
		token := strings.TrimSpace(parts[1])
		if role == "" || token == "" {
			log.Printf("[WARN] Empty role or token in WORKFLOW_API_TOKENS pair: '%s', skipping", pair)
			continue
		}
		m[token] = Role(role)
	}
	log.Printf("[INFO] Loaded %d token-role pairs from WORKFLOW_API_TOKENS", len(m))
	return m
}

func getenv(key string) string {
	// Try to load from .secrets.json if present, else fall back to environment variable.
	f, err := os.Open(".secrets.json")
	if err == nil {
		defer f.Close()
		var secrets map[string]string
		dec := json.NewDecoder(f)
		if err := dec.Decode(&secrets); err == nil {
			if v, ok := secrets[key]; ok {
				return v
			}
		} else {
			log.Printf("Warning: could not parse .secrets.json: %v", err)
		}
	}
	return os.Getenv(key)
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
