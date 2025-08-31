package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/config"
)

// TestSupabaseClient_ValidateJWT tests JWT validation
func TestSupabaseClient_ValidateJWT(t *testing.T) {
	cfg := &config.SupabaseConfig{
		URL:     "https://test.supabase.co",
		AnonKey: "test-anon-key",
	}
	jwtSecret := "test-secret"
	
	client := NewSupabaseClient(cfg, jwtSecret)
	
	// Create a valid test token
	now := time.Now()
	claims := &JWTClaims{
		Aud:   "authenticated",
		Iss:   cfg.URL + "/auth/v1",
		Sub:   "test-user-id",
		Email: "test@example.com",
		Role:  "user",
		Exp:   now.Add(time.Hour).Unix(),
		Iat:   now.Unix(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		t.Fatalf("Failed to create test token: %v", err)
	}
	
	// Test valid token
	validatedClaims, err := client.ValidateJWT(tokenString)
	if err != nil {
		t.Fatalf("Expected valid token to pass validation: %v", err)
	}
	
	if validatedClaims.Sub != "test-user-id" {
		t.Errorf("Expected user ID 'test-user-id', got '%s'", validatedClaims.Sub)
	}
	
	if validatedClaims.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%s'", validatedClaims.Email)
	}
	
	// Test invalid token
	_, err = client.ValidateJWT("invalid-token")
	if err == nil {
		t.Error("Expected invalid token to fail validation")
	}
	
	// Test expired token
	expiredTime := time.Now().Add(-time.Hour)
	expiredClaims := &JWTClaims{
		Aud:   "authenticated",
		Iss:   cfg.URL + "/auth/v1",
		Sub:   "test-user-id",
		Email: "test@example.com",
		Exp:   expiredTime.Unix(),
		Iat:   expiredTime.Add(-time.Hour).Unix(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiredTime),
			IssuedAt:  jwt.NewNumericDate(expiredTime.Add(-time.Hour)),
		},
	}
	
	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	expiredTokenString, err := expiredToken.SignedString([]byte(jwtSecret))
	if err != nil {
		t.Fatalf("Failed to create expired test token: %v", err)
	}
	
	_, err = client.ValidateJWT(expiredTokenString)
	if err == nil {
		t.Error("Expected expired token to fail validation")
	}
}

// TestSupabaseClient_GetUser tests user retrieval
func TestSupabaseClient_GetUser(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/v1/user" {
			t.Errorf("Expected path '/auth/v1/user', got '%s'", r.URL.Path)
		}
		
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Expected Authorization header 'Bearer test-token', got '%s'", r.Header.Get("Authorization"))
		}
		
		user := SupabaseUser{
			ID:    "test-user-id",
			Email: "test@example.com",
			Role:  "user",
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	}))
	defer server.Close()
	
	cfg := &config.SupabaseConfig{
		URL:     server.URL,
		AnonKey: "test-anon-key",
	}
	
	client := NewSupabaseClient(cfg, "test-secret")
	
	user, err := client.GetUser(context.Background(), "test-token")
	if err != nil {
		t.Fatalf("Expected GetUser to succeed: %v", err)
	}
	
	if user.ID != "test-user-id" {
		t.Errorf("Expected user ID 'test-user-id', got '%s'", user.ID)
	}
	
	if user.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%s'", user.Email)
	}
}

// TestSupabaseClient_RefreshToken tests token refresh
func TestSupabaseClient_RefreshToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/v1/token" {
			t.Errorf("Expected path '/auth/v1/token', got '%s'", r.URL.Path)
		}
		
		if r.URL.Query().Get("grant_type") != "refresh_token" {
			t.Error("Expected grant_type=refresh_token in query")
		}
		
		var req map[string]string
		json.NewDecoder(r.Body).Decode(&req)
		
		if req["refresh_token"] != "test-refresh-token" {
			t.Errorf("Expected refresh_token 'test-refresh-token', got '%s'", req["refresh_token"])
		}
		
		session := SupabaseSession{
			AccessToken:  "new-access-token",
			RefreshToken: "new-refresh-token",
			ExpiresIn:    3600,
			TokenType:    "bearer",
		}
		
		response := SupabaseAuthResponse{
			Session: &session,
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	cfg := &config.SupabaseConfig{
		URL:     server.URL,
		AnonKey: "test-anon-key",
	}
	
	client := NewSupabaseClient(cfg, "test-secret")
	
	session, err := client.RefreshToken(context.Background(), "test-refresh-token")
	if err != nil {
		t.Fatalf("Expected RefreshToken to succeed: %v", err)
	}
	
	if session.AccessToken != "new-access-token" {
		t.Errorf("Expected access token 'new-access-token', got '%s'", session.AccessToken)
	}
	
	if session.RefreshToken != "new-refresh-token" {
		t.Errorf("Expected refresh token 'new-refresh-token', got '%s'", session.RefreshToken)
	}
}

// TestSupabaseClient_GetUserRole tests role extraction
func TestSupabaseClient_GetUserRole(t *testing.T) {
	cfg := &config.SupabaseConfig{
		URL:     "https://test.supabase.co",
		AnonKey: "test-anon-key",
	}
	
	client := NewSupabaseClient(cfg, "test-secret")
	
	// Test role from app_metadata
	claims := &JWTClaims{
		AppMetadata: map[string]interface{}{
			"role": "admin",
		},
	}
	
	role := client.GetUserRole(claims)
	if role != "admin" {
		t.Errorf("Expected role 'admin', got '%s'", role)
	}
	
	// Test role from user_metadata
	claims = &JWTClaims{
		UserMetadata: map[string]interface{}{
			"role": "moderator",
		},
	}
	
	role = client.GetUserRole(claims)
	if role != "moderator" {
		t.Errorf("Expected role 'moderator', got '%s'", role)
	}
	
	// Test direct role claim
	claims = &JWTClaims{
		Role: "user",
	}
	
	role = client.GetUserRole(claims)
	if role != "user" {
		t.Errorf("Expected role 'user', got '%s'", role)
	}
	
	// Test default role
	claims = &JWTClaims{}
	
	role = client.GetUserRole(claims)
	if role != "user" {
		t.Errorf("Expected default role 'user', got '%s'", role)
	}
}

// TestExtractTokenFromHeader tests token extraction from headers
func TestExtractTokenFromHeader(t *testing.T) {
	// Test valid header
	token, err := ExtractTokenFromHeader("Bearer test-token")
	if err != nil {
		t.Fatalf("Expected valid header to succeed: %v", err)
	}
	
	if token != "test-token" {
		t.Errorf("Expected token 'test-token', got '%s'", token)
	}
	
	// Test empty header
	_, err = ExtractTokenFromHeader("")
	if err == nil {
		t.Error("Expected empty header to fail")
	}
	
	// Test invalid format
	_, err = ExtractTokenFromHeader("Invalid format")
	if err == nil {
		t.Error("Expected invalid format to fail")
	}
	
	// Test wrong scheme
	_, err = ExtractTokenFromHeader("Basic test-token")
	if err == nil {
		t.Error("Expected wrong scheme to fail")
	}
}

// TestSupabaseClient_IsTokenExpired tests token expiration check
func TestSupabaseClient_IsTokenExpired(t *testing.T) {
	cfg := &config.SupabaseConfig{
		URL:     "https://test.supabase.co",
		AnonKey: "test-anon-key",
	}
	jwtSecret := "test-secret"
	
	client := NewSupabaseClient(cfg, jwtSecret)
	
	// Create non-expired token
	futureTime := time.Now().Add(time.Hour)
	claims := &JWTClaims{
		Aud: "authenticated",
		Iss: cfg.URL + "/auth/v1",
		Exp: futureTime.Unix(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(futureTime),
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(jwtSecret))
	
	if client.IsTokenExpired(tokenString) {
		t.Error("Expected non-expired token to not be expired")
	}
	
	// Create expired token
	pastTime := time.Now().Add(-time.Hour)
	expiredClaims := &JWTClaims{
		Aud: "authenticated",
		Iss: cfg.URL + "/auth/v1",
		Exp: pastTime.Unix(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(pastTime),
		},
	}
	
	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	expiredTokenString, _ := expiredToken.SignedString([]byte(jwtSecret))
	
	if !client.IsTokenExpired(expiredTokenString) {
		t.Error("Expected expired token to be expired")
	}
	
	// Test invalid token
	if !client.IsTokenExpired("invalid-token") {
		t.Error("Expected invalid token to be considered expired")
	}
}