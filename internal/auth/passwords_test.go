package auth

import "testing"

func TestHashPassword_NonDeterministic(t *testing.T) {
	p := "correct horse battery staple"
	h1, err := HashPassword(p)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	h2, err := HashPassword(p)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if h1 == h2 {
		t.Fatalf("expected different hashes for same password")
	}
}

func TestVerifyPassword(t *testing.T) {
	p := "correct horse battery staple"
	h, err := HashPassword(p)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	ok, err := VerifyPassword(h, p)
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if !ok {
		t.Fatalf("expected password to verify")
	}

	ok, err = VerifyPassword(h, "wrong password")
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if ok {
		t.Fatalf("expected wrong password to fail verification")
	}
}
