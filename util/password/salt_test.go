package password

import "testing"

func TestHashPasswordRoundTrip(t *testing.T) {
	raw := "S3cure!Pass"
	hash, err := HashPassword(raw)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	if hash == raw {
		t.Fatal("hash should not equal raw password")
	}
	if !CheckPasswordHash(raw, hash) {
		t.Fatal("CheckPasswordHash should return true for correct password")
	}
	if CheckPasswordHash("wrong", hash) {
		t.Fatal("CheckPasswordHash should return false for wrong password")
	}
}
