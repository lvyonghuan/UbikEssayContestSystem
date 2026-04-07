package password

import "testing"

func hasAnyRuneFromSet(s, set string) bool {
	for _, ch := range s {
		for _, target := range set {
			if ch == target {
				return true
			}
		}
	}
	return false
}

func TestGenerateHasRequiredCharacterClasses(t *testing.T) {
	pwd := Generate()
	if len(pwd) != defaultLen {
		t.Fatalf("unexpected password length: %d", len(pwd))
	}
	if !hasAnyRuneFromSet(pwd, lower) {
		t.Fatal("generated password should contain lowercase letters")
	}
	if !hasAnyRuneFromSet(pwd, upper) {
		t.Fatal("generated password should contain uppercase letters")
	}
	if !hasAnyRuneFromSet(pwd, digits) {
		t.Fatal("generated password should contain digits")
	}
	if !hasAnyRuneFromSet(pwd, symbols) {
		t.Fatal("generated password should contain symbols")
	}
}

func TestBatchGenerate(t *testing.T) {
	pwds := BatchGenerate(5)
	if len(pwds) != 5 {
		t.Fatalf("unexpected password count: %d", len(pwds))
	}
	for i, pwd := range pwds {
		if len(pwd) != defaultLen {
			t.Fatalf("password %d has unexpected length: %d", i, len(pwd))
		}
	}

	empty := BatchGenerate(0)
	if len(empty) != 0 {
		t.Fatalf("expected empty result for zero count, got %d", len(empty))
	}
}
