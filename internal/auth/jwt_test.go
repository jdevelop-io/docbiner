package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestGenerateAndValidate(t *testing.T) {
	svc := New("test-secret-key", 1*time.Hour)

	userID := uuid.New()
	orgID := uuid.New()
	role := "owner"

	token, err := svc.Generate(userID, orgID, role)
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if token == "" {
		t.Fatal("Generate() returned empty token")
	}

	claims, err := svc.Validate(token)
	if err != nil {
		t.Fatalf("Validate() error: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("UserID = %v, want %v", claims.UserID, userID)
	}
	if claims.OrgID != orgID {
		t.Errorf("OrgID = %v, want %v", claims.OrgID, orgID)
	}
	if claims.Role != role {
		t.Errorf("Role = %q, want %q", claims.Role, role)
	}
	if claims.Issuer != "docbiner" {
		t.Errorf("Issuer = %q, want %q", claims.Issuer, "docbiner")
	}
	if claims.Subject != userID.String() {
		t.Errorf("Subject = %q, want %q", claims.Subject, userID.String())
	}
}

func TestValidate_ExpiredToken(t *testing.T) {
	// Use a negative expiration so the token is immediately expired.
	svc := New("test-secret-key", -1*time.Hour)

	token, err := svc.Generate(uuid.New(), uuid.New(), "member")
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	_, err = svc.Validate(token)
	if err == nil {
		t.Fatal("Validate() expected error for expired token, got nil")
	}
}

func TestValidate_InvalidToken(t *testing.T) {
	svc := New("test-secret-key", 1*time.Hour)

	_, err := svc.Validate("this.is.not.a.valid.token")
	if err == nil {
		t.Fatal("Validate() expected error for invalid token, got nil")
	}
}

func TestValidate_TamperedToken(t *testing.T) {
	svc := New("test-secret-key", 1*time.Hour)

	token, err := svc.Generate(uuid.New(), uuid.New(), "admin")
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	// Tamper with the token by modifying the last character.
	tampered := token[:len(token)-1] + "X"

	_, err = svc.Validate(tampered)
	if err == nil {
		t.Fatal("Validate() expected error for tampered token, got nil")
	}
}

func TestValidate_WrongSecret(t *testing.T) {
	svc1 := New("secret-one", 1*time.Hour)
	svc2 := New("secret-two", 1*time.Hour)

	token, err := svc1.Generate(uuid.New(), uuid.New(), "member")
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	_, err = svc2.Validate(token)
	if err == nil {
		t.Fatal("Validate() expected error for wrong secret, got nil")
	}
}

func TestValidate_WrongSigningMethod(t *testing.T) {
	svc := New("test-secret-key", 1*time.Hour)

	// Create a token with a different signing method (none).
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "docbiner",
		},
		UserID: uuid.New(),
		OrgID:  uuid.New(),
		Role:   "member",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenString, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("create none-signed token: %v", err)
	}

	_, err = svc.Validate(tokenString)
	if err == nil {
		t.Fatal("Validate() expected error for none-signed token, got nil")
	}
}

func TestHashPassword_And_CheckPassword(t *testing.T) {
	password := "securepass123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error: %v", err)
	}

	if hash == "" {
		t.Fatal("HashPassword() returned empty hash")
	}
	if hash == password {
		t.Fatal("HashPassword() returned plaintext password")
	}

	if !CheckPassword(hash, password) {
		t.Error("CheckPassword() returned false for correct password")
	}

	if CheckPassword(hash, "wrong-password") {
		t.Error("CheckPassword() returned true for wrong password")
	}
}

func TestHashPassword_DifferentHashes(t *testing.T) {
	password := "securepass123"

	hash1, _ := HashPassword(password)
	hash2, _ := HashPassword(password)

	if hash1 == hash2 {
		t.Error("HashPassword() produced identical hashes for same password (bcrypt should use random salt)")
	}

	// Both should still match the original password.
	if !CheckPassword(hash1, password) {
		t.Error("CheckPassword() failed for hash1")
	}
	if !CheckPassword(hash2, password) {
		t.Error("CheckPassword() failed for hash2")
	}
}
