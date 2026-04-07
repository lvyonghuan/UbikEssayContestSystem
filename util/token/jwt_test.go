package token

import (
	"main/conf"
	"testing"
	"time"
)

func backupTokenGlobals(t *testing.T) {
	origAccessKey := accessTokenKey
	origRefreshKey := refreshTokenKey
	origAccessExpiry := accessTokenExpiry
	origRefreshExpiry := refreshTokenExpiry

	t.Cleanup(func() {
		accessTokenKey = origAccessKey
		refreshTokenKey = origRefreshKey
		accessTokenExpiry = origAccessExpiry
		refreshTokenExpiry = origRefreshExpiry
	})
}

func TestInitJWT(t *testing.T) {
	backupTokenGlobals(t)

	t.Setenv("Ubik_JWT_Access_Key", "access-key")
	t.Setenv("Ubik_JWT_Refresh_Key", "refresh-key")
	err := InitJWT(conf.TokenConfig{AccessTokenExpire: 3, RefreshTokenExpire: 5})
	if err != nil {
		t.Fatalf("InitJWT failed: %v", err)
	}
	if accessTokenExpiry != 3*time.Hour {
		t.Fatalf("unexpected access token expiry: %v", accessTokenExpiry)
	}
	if refreshTokenExpiry != 5*time.Hour {
		t.Fatalf("unexpected refresh token expiry: %v", refreshTokenExpiry)
	}

	t.Setenv("Ubik_JWT_Access_Key", "")
	t.Setenv("Ubik_JWT_Refresh_Key", "")
	err = InitJWT(conf.TokenConfig{AccessTokenExpire: 1, RefreshTokenExpire: 1})
	if err == nil {
		t.Fatal("InitJWT should fail when keys are empty")
	}
}

func TestGenAndParseTokens(t *testing.T) {
	backupTokenGlobals(t)

	accessTokenKey = []byte("a-very-secret-access-key")
	refreshTokenKey = []byte("a-very-secret-refresh-key")
	accessTokenExpiry = time.Hour
	refreshTokenExpiry = 2 * time.Hour

	resp, err := GenTokenAndRefreshToken(42, "admin")
	if err != nil {
		t.Fatalf("GenTokenAndRefreshToken failed: %v", err)
	}
	if resp.Token == "" || resp.RefreshToken == "" {
		t.Fatal("generated token fields should not be empty")
	}

	accessClaims, err := ParseAccessToken(resp.Token)
	if err != nil {
		t.Fatalf("ParseAccessToken failed: %v", err)
	}
	if accessClaims.ID != 42 || accessClaims.Role != "admin" {
		t.Fatalf("unexpected access token claims: %+v", accessClaims)
	}

	refreshClaims, err := ParseRefreshToken(resp.RefreshToken)
	if err != nil {
		t.Fatalf("ParseRefreshToken failed: %v", err)
	}
	if refreshClaims.ID != 42 || refreshClaims.Role != "admin" {
		t.Fatalf("unexpected refresh token claims: %+v", refreshClaims)
	}
}

func TestJWTErrorBranches(t *testing.T) {
	backupTokenGlobals(t)

	if _, err := genToken(UserClaims{}, []byte{}); err == nil {
		t.Fatal("genToken should fail when key is empty")
	}

	if _, err := parseJWT("not-a-token", []byte("key")); err == nil {
		t.Fatal("parseJWT should fail for invalid token string")
	}
}
