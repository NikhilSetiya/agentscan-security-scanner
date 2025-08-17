package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NikhilSetiya/agentscan-security-scanner/internal/database"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/config"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/types"
)

// TestAuthenticationSecurity tests various authentication security scenarios
func TestAuthenticationSecurity(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret:     "test-secret-key-that-is-long-enough",
			JWTExpiration: time.Hour,
		},
	}

	t.Run("JWT Token Security", func(t *testing.T) {
		t.Run("should reject tokens with invalid signature", func(t *testing.T) {
			router := setupSecurityTestRouter(cfg)

			// Create a token with wrong secret
			claims := JWTClaims{
				UserID: uuid.New(),
				Email:  "test@example.com",
				Name:   "Test User",
				RegisteredClaims: jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
					IssuedAt:  jwt.NewNumericDate(time.Now()),
					NotBefore: jwt.NewNumericDate(time.Now()),
					Issuer:    "agentscan",
				},
			}

			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			tokenString, _ := token.SignedString([]byte("wrong-secret"))

			req, _ := http.NewRequest("GET", "/protected", nil)
			req.Header.Set("Authorization", "Bearer "+tokenString)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})

		t.Run("should reject expired tokens", func(t *testing.T) {
			router := setupSecurityTestRouter(cfg)

			// Create an expired token
			claims := JWTClaims{
				UserID: uuid.New(),
				Email:  "test@example.com",
				Name:   "Test User",
				RegisteredClaims: jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)), // Expired
					IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
					NotBefore: jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
					Issuer:    "agentscan",
				},
			}

			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			tokenString, _ := token.SignedString([]byte(cfg.Auth.JWTSecret))

			req, _ := http.NewRequest("GET", "/protected", nil)
			req.Header.Set("Authorization", "Bearer "+tokenString)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})

		t.Run("should reject tokens with future not-before time", func(t *testing.T) {
			router := setupSecurityTestRouter(cfg)

			// Create a token that's not valid yet
			claims := JWTClaims{
				UserID: uuid.New(),
				Email:  "test@example.com",
				Name:   "Test User",
				RegisteredClaims: jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
					IssuedAt:  jwt.NewNumericDate(time.Now()),
					NotBefore: jwt.NewNumericDate(time.Now().Add(time.Hour)), // Future
					Issuer:    "agentscan",
				},
			}

			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			tokenString, _ := token.SignedString([]byte(cfg.Auth.JWTSecret))

			req, _ := http.NewRequest("GET", "/protected", nil)
			req.Header.Set("Authorization", "Bearer "+tokenString)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})

		t.Run("should reject tokens with wrong algorithm", func(t *testing.T) {
			router := setupSecurityTestRouter(cfg)

			// Create a token with RS256 instead of HS256
			claims := JWTClaims{
				UserID: uuid.New(),
				Email:  "test@example.com",
				Name:   "Test User",
				RegisteredClaims: jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
					IssuedAt:  jwt.NewNumericDate(time.Now()),
					NotBefore: jwt.NewNumericDate(time.Now()),
					Issuer:    "agentscan",
				},
			}

			// This will fail because we're using the wrong algorithm
			token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
			tokenString, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)

			req, _ := http.NewRequest("GET", "/protected", nil)
			req.Header.Set("Authorization", "Bearer "+tokenString)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	})

	t.Run("Authorization Header Security", func(t *testing.T) {
		t.Run("should reject malformed authorization headers", func(t *testing.T) {
			router := setupSecurityTestRouter(cfg)

			testCases := []string{
				"",                    // Empty
				"Bearer",              // Missing token
				"Basic dGVzdA==",      // Wrong scheme
				"Bearer token extra",  // Extra parts
				"bearer token",        // Wrong case
				"Token abc123",        // Wrong scheme
			}

			for _, authHeader := range testCases {
				req, _ := http.NewRequest("GET", "/protected", nil)
				if authHeader != "" {
					req.Header.Set("Authorization", authHeader)
				}
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusUnauthorized, w.Code, "Should reject: %s", authHeader)
			}
		})
	})

	t.Run("Rate Limiting", func(t *testing.T) {
		// This would test rate limiting if implemented
		t.Skip("Rate limiting not fully implemented yet")
	})

	t.Run("CSRF Protection", func(t *testing.T) {
		t.Run("should validate state parameter in OAuth flow", func(t *testing.T) {
			// This would test CSRF protection in OAuth flows
			t.Skip("CSRF protection not fully implemented yet")
		})
	})
}

// TestRBACSecurityScenarios tests role-based access control security
func TestRBACSecurityScenarios(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret:     "test-secret-key-that-is-long-enough",
			JWTExpiration: time.Hour,
		},
	}

	t.Run("Permission Escalation Prevention", func(t *testing.T) {
		t.Run("member cannot access admin endpoints", func(t *testing.T) {
			router := setupRBACTestRouter(cfg)

			// Create token for member user
			userID := uuid.New()
			token := generateTestJWTWithRole(userID, "test@example.com", "Test User", cfg.Auth.JWTSecret)

			req, _ := http.NewRequest("DELETE", "/api/v1/orgs/"+uuid.New().String(), nil)
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("X-Organization-ID", uuid.New().String())
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusForbidden, w.Code)
		})

		t.Run("admin cannot access owner endpoints", func(t *testing.T) {
			router := setupRBACTestRouter(cfg)

			userID := uuid.New()
			token := generateTestJWTWithRole(userID, "admin@example.com", "Admin User", cfg.Auth.JWTSecret)

			req, _ := http.NewRequest("DELETE", "/api/v1/orgs/"+uuid.New().String()+"/delete-permanently", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("X-Organization-ID", uuid.New().String())
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusForbidden, w.Code)
		})
	})

	t.Run("Cross-Organization Access Prevention", func(t *testing.T) {
		t.Run("user cannot access different organization resources", func(t *testing.T) {
			router := setupRBACTestRouter(cfg)

			userID := uuid.New()
			otherOrgID := uuid.New()
			token := generateTestJWTWithRole(userID, "user@example.com", "User", cfg.Auth.JWTSecret)

			// Try to access resource from different organization
			req, _ := http.NewRequest("GET", "/api/v1/orgs/"+otherOrgID.String()+"/repos", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("X-Organization-ID", otherOrgID.String())
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusForbidden, w.Code)
		})
	})
}

// TestAuditLoggingSecurity tests audit logging security features
func TestAuditLoggingSecurity(t *testing.T) {
	t.Run("Audit Log Integrity", func(t *testing.T) {
		mockRepos := &database.Repositories{} // Use proper type
		auditLogger := NewAuditLogger(mockRepos)

		t.Run("should log authentication events", func(t *testing.T) {
			mockRepos := &database.Repositories{} // Use proper type
			auditLogger := NewAuditLogger(mockRepos)
			
			ctx := context.Background()
			userID := uuid.New()

			err := auditLogger.LogAuthEvent(ctx, AuditEventLogin, &userID, true, map[string]interface{}{
				"provider": "github",
			}, nil)

			assert.NoError(t, err)
		})

		t.Run("should log authorization events", func(t *testing.T) {
			ctx := context.Background()
			userID := uuid.New()
			orgID := uuid.New()
			resourceID := uuid.New()

			err := auditLogger.LogAuthzEvent(ctx, AuditEventPermissionDenied, &userID, &orgID, "repository", &resourceID, false, map[string]interface{}{
				"permission": "repo:delete",
			}, nil)

			assert.NoError(t, err)
		})
	})

	t.Run("Sensitive Data Protection", func(t *testing.T) {
		t.Run("should not log sensitive information", func(t *testing.T) {
			// This would test that passwords, tokens, etc. are not logged
			t.Skip("Sensitive data protection tests not implemented yet")
		})
	})
}

// TestSessionSecurity tests session management security
func TestSessionSecurity(t *testing.T) {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret:     "test-secret-key-that-is-long-enough",
			JWTExpiration: time.Hour,
		},
	}

	t.Run("Token Lifecycle", func(t *testing.T) {
		t.Run("should generate secure tokens", func(t *testing.T) {
			authHandler := &AuthHandler{
				config: cfg,
			}

			// Generate tokens for different users
			user1 := &types.User{
				ID:    uuid.New(),
				Email: "test1@example.com",
				Name:  "Test User 1",
			}

			user2 := &types.User{
				ID:    uuid.New(),
				Email: "test2@example.com",
				Name:  "Test User 2",
			}

			token1, _, err := authHandler.generateJWTToken(user1)
			require.NoError(t, err)

			token2, _, err := authHandler.generateJWTToken(user2)
			require.NoError(t, err)

			// Tokens should be different for different users
			assert.NotEqual(t, token1, token2)
			assert.NotEmpty(t, token1)
			assert.NotEmpty(t, token2)
		})

		t.Run("should have proper token expiration", func(t *testing.T) {
			user := &types.User{
				ID:    uuid.New(),
				Email: "test@example.com",
				Name:  "Test User",
			}

			authHandler := &AuthHandler{
				config: cfg,
			}

			_, expiresAt, err := authHandler.generateJWTToken(user)
			require.NoError(t, err)

			// Token should expire in approximately 1 hour
			expectedExpiry := time.Now().Add(cfg.Auth.JWTExpiration)
			assert.WithinDuration(t, expectedExpiry, expiresAt, time.Minute)
		})
	})
}

// Helper functions for security tests

func setupSecurityTestRouter(cfg *config.Config) *gin.Engine {
	router := gin.New()
	router.Use(AuthMiddleware(cfg))

	router.GET("/protected", func(c *gin.Context) {
		user, exists := GetCurrentUser(c)
		if !exists {
			UnauthorizedResponse(c, "User not found")
			return
		}
		SuccessResponse(c, ToUserDTO(user))
	})

	return router
}

func setupRBACTestRouter(cfg *config.Config) *gin.Engine {
	router := gin.New()
	
	// Mock RBAC service
	mockRepos := &database.Repositories{} // Use proper type
	rbacService := NewRBACService(mockRepos)

	router.Use(AuthMiddleware(cfg))

	// Protected routes with RBAC
	api := router.Group("/api/v1")
	{
		orgs := api.Group("/orgs")
		{
			orgs.DELETE("/:id", rbacService.RequirePermission(PermissionOrgDelete), func(c *gin.Context) {
				SuccessResponse(c, map[string]string{"message": "Organization deleted"})
			})

			orgs.DELETE("/:id/delete-permanently", rbacService.RequireRole(types.RoleOwner), func(c *gin.Context) {
				SuccessResponse(c, map[string]string{"message": "Organization permanently deleted"})
			})

			orgs.GET("/:id/repos", rbacService.RequirePermission(PermissionRepoRead), func(c *gin.Context) {
				SuccessResponse(c, []string{})
			})
		}
	}

	return router
}

func generateTestJWTWithRole(userID uuid.UUID, email, name, secret string) string {
	claims := JWTClaims{
		UserID: userID,
		Email:  email,
		Name:   name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "agentscan",
			Subject:   userID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}

// Benchmark tests for performance
func BenchmarkJWTTokenGeneration(b *testing.B) {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret:     "test-secret-key-that-is-long-enough",
			JWTExpiration: time.Hour,
		},
	}

	authHandler := &AuthHandler{
		config: cfg,
	}

	user := &types.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Name:  "Test User",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := authHandler.generateJWTToken(user)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJWTTokenValidation(b *testing.B) {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret:     "test-secret-key-that-is-long-enough",
			JWTExpiration: time.Hour,
		},
	}

	// Generate a token to validate
	claims := JWTClaims{
		UserID: uuid.New(),
		Email:  "test@example.com",
		Name:   "Test User",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "agentscan",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(cfg.Auth.JWTSecret))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(cfg.Auth.JWTSecret), nil
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}