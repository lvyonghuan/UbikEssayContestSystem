package token

import (
	"testing"
	"time"
)

func issueTokensForCheckTest(t *testing.T) (string, string) {
	t.Helper()
	backupTokenGlobals(t)

	accessTokenKey = []byte("access-check-key")
	refreshTokenKey = []byte("refresh-check-key")
	accessTokenExpiry = time.Hour
	refreshTokenExpiry = 2 * time.Hour

	resp, err := GenTokenAndRefreshToken(7, "author")
	if err != nil {
		t.Fatalf("GenTokenAndRefreshToken failed: %v", err)
	}
	return resp.Token, resp.RefreshToken
}

func TestParseBearerToken(t *testing.T) {
	if _, err := parseBearerToken(""); err == nil {
		t.Fatal("parseBearerToken should fail for empty header")
	}
	if _, err := parseBearerToken("Token abc"); err == nil {
		t.Fatal("parseBearerToken should fail for invalid prefix")
	}
	if _, err := parseBearerToken("Bearer "); err == nil {
		t.Fatal("parseBearerToken should fail for empty bearer token")
	}

	tokenStr, err := parseBearerToken("Bearer abc123")
	if err != nil {
		t.Fatalf("parseBearerToken failed: %v", err)
	}
	if tokenStr != "abc123" {
		t.Fatalf("unexpected token string: %s", tokenStr)
	}
}

func TestCheckTokenAndRefreshToken(t *testing.T) {
	access, refresh := issueTokensForCheckTest(t)

	id, role, err := CheckToken("Bearer " + access)
	if err != nil {
		t.Fatalf("CheckToken failed: %v", err)
	}
	if id != 7 || role != "author" {
		t.Fatalf("unexpected CheckToken result: id=%d role=%s", id, role)
	}

	id, role, err = CheckRefreshToken("Bearer " + refresh)
	if err != nil {
		t.Fatalf("CheckRefreshToken failed: %v", err)
	}
	if id != 7 || role != "author" {
		t.Fatalf("unexpected CheckRefreshToken result: id=%d role=%s", id, role)
	}
}

func TestCheckTokenErrorBranches(t *testing.T) {
	backupTokenGlobals(t)
	accessTokenKey = []byte("access-check-key")
	refreshTokenKey = []byte("refresh-check-key")

	if _, _, err := CheckToken("Token invalid"); err == nil {
		t.Fatal("CheckToken should fail on invalid header format")
	}
	if _, _, err := CheckToken("Bearer invalid"); err == nil {
		t.Fatal("CheckToken should fail on invalid token")
	}

	if _, _, err := CheckRefreshToken("Token invalid"); err == nil {
		t.Fatal("CheckRefreshToken should fail on invalid header format")
	}
	if _, _, err := CheckRefreshToken("Bearer invalid"); err == nil {
		t.Fatal("CheckRefreshToken should fail on invalid token")
	}
}
