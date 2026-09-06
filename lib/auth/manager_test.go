package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-secret"

func newTestManager(ttl time.Duration, secure bool) *Manager {
	return New(Config{Secret: []byte(testSecret), TTL: ttl, Secure: secure})
}

func signWith(t *testing.T, m *Manager, c claims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	s, err := token.SignedString(m.secret)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestSignVerifyRoundTrip(t *testing.T) {
	m := newTestManager(time.Hour, false)

	token, err := m.Sign(42, "alice")
	if err != nil {
		t.Fatal(err)
	}

	userID, err := m.Verify(token)
	if err != nil {
		t.Fatal(err)
	}
	if userID != 42 {
		t.Fatalf("userID = %d, want 42", userID)
	}
}

func TestSignEmbedsSubjectAndUsername(t *testing.T) {
	m := newTestManager(time.Hour, false)

	token, err := m.Sign(7, "bob")
	if err != nil {
		t.Fatal(err)
	}

	c := &claims{}
	_, err = jwt.ParseWithClaims(token, c,
		func(*jwt.Token) (any, error) { return m.secret, nil },
		jwt.WithoutClaimsValidation())
	if err != nil {
		t.Fatal(err)
	}
	if c.Subject != "7" || c.Username != "bob" {
		t.Fatalf("claims = %+v, want subject 7 username bob", c)
	}
	if c.Issuer != DefaultIssuer {
		t.Fatalf("issuer = %q, want %q", c.Issuer, DefaultIssuer)
	}
}

func TestVerifyExpiredToken(t *testing.T) {
	m := newTestManager(-time.Hour, false)

	token, err := m.Sign(1, "alice")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := m.Verify(token); !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("err = %v, want ErrExpiredToken", err)
	}
}

func TestVerifyWrongSecret(t *testing.T) {
	m := newTestManager(time.Hour, false)
	other := New(Config{Secret: []byte("other-secret"), TTL: time.Hour})

	token, err := m.Sign(1, "alice")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := other.Verify(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("err = %v, want ErrInvalidToken", err)
	}
}

func TestVerifyTamperedToken(t *testing.T) {
	m := newTestManager(time.Hour, false)

	token, err := m.Sign(1, "alice")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := m.Verify(token + "x"); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("err = %v, want ErrInvalidToken", err)
	}
}

func TestVerifyMalformedToken(t *testing.T) {
	m := newTestManager(time.Hour, false)

	if _, err := m.Verify("not.a.token"); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("err = %v, want ErrInvalidToken", err)
	}
}

func TestVerifyWrongIssuer(t *testing.T) {
	m := newTestManager(time.Hour, false)
	c := claims{RegisteredClaims: jwt.RegisteredClaims{
		Subject:   "1",
		Issuer:    "other",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}}
	token := signWith(t, m, c)

	if _, err := m.Verify(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("err = %v, want ErrInvalidToken (wrong issuer)", err)
	}
}

func TestVerifyMissingExpiration(t *testing.T) {
	m := newTestManager(time.Hour, false)
	c := claims{RegisteredClaims: jwt.RegisteredClaims{
		Subject: "1",
		Issuer:  DefaultIssuer,
	}}
	token := signWith(t, m, c)

	if _, err := m.Verify(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("err = %v, want ErrInvalidToken", err)
	}
}
