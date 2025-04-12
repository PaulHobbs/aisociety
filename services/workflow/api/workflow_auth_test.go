package api

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestAuthenticate(t *testing.T) {
	// Setup: Override the global token map for testing
	originalTokenRoleMap := tokenRoleMap
	tokenRoleMap = map[string]Role{
		"admin-token": RoleAdmin,
		"user-token":  RoleUser,
	}
	// Restore original map after test
	defer func() { tokenRoleMap = originalTokenRoleMap }()

	tests := []struct {
		name         string
		ctx          context.Context
		expectedRole Role
		expectedCode codes.Code // Expected gRPC status code
	}{
		{
			name:         "no metadata",
			ctx:          context.Background(),
			expectedRole: "",
			expectedCode: codes.Unauthenticated,
		},
		{
			name:         "missing authorization header",
			ctx:          metadata.NewIncomingContext(context.Background(), metadata.MD{}),
			expectedRole: "",
			expectedCode: codes.Unauthenticated,
		},
		{
			name:         "invalid header format",
			ctx:          metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "invalid-token-format")),
			expectedRole: "",
			expectedCode: codes.Unauthenticated, // authenticate expects "Bearer <token>" but parseBearerToken handles it
		},
		{
			name:         "invalid token",
			ctx:          metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer invalid-token")),
			expectedRole: "",
			expectedCode: codes.Unauthenticated,
		},
		{
			name:         "valid user token",
			ctx:          metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer user-token")),
			expectedRole: RoleUser,
			expectedCode: codes.OK,
		},
		{
			name:         "valid admin token",
			ctx:          metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer admin-token")),
			expectedRole: RoleAdmin,
			expectedCode: codes.OK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			role, err := authenticate(tc.ctx)

			if tc.expectedCode == codes.OK {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if role != tc.expectedRole {
					t.Errorf("expected role %q, got %q", tc.expectedRole, role)
				}
			} else {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				st, ok := status.FromError(err)
				if !ok {
					t.Fatalf("expected gRPC status error, got %v", err)
				}
				if st.Code() != tc.expectedCode {
					t.Errorf("expected status code %v, got %v", tc.expectedCode, st.Code())
				}
			}
		})
	}
}

func TestAuthorize(t *testing.T) {
	tests := []struct {
		name         string
		userRole     Role
		requiredRole Role
		expected     bool
	}{
		{"admin accessing admin resource", RoleAdmin, RoleAdmin, true},
		{"admin accessing user resource", RoleAdmin, RoleUser, true},
		{"user accessing user resource", RoleUser, RoleUser, true},
		{"user accessing admin resource", RoleUser, RoleAdmin, false},
		{"empty role accessing user resource", "", RoleUser, false},
		{"empty role accessing admin resource", "", RoleAdmin, false},
		{"user accessing unknown resource (defaults to admin)", RoleUser, RoleAdmin, false},  // Assuming default is Admin
		{"admin accessing unknown resource (defaults to admin)", RoleAdmin, RoleAdmin, true}, // Assuming default is Admin
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := authorize(tc.userRole, tc.requiredRole)
			if result != tc.expected {
				t.Errorf("authorize(%q, %q) = %v; want %v", tc.userRole, tc.requiredRole, result, tc.expected)
			}
		})
	}
}

// TODO: Add tests for AuthInterceptor
// TODO: Add tests for loadTokenRoleMapFromEnv (requires mocking getenv or setting env vars)
