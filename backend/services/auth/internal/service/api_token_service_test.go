package service

import (
	"crypto/sha256"
	"errors"
	"net/http"
	"strings"
	"testing"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/services/auth/internal/domain"
)

// These tests cover the parts of api_token_service.go that don't need a
// database: the token generator, the parser, the base62 encoder, and the
// error mapper. End-to-end IssueAPIToken / RevokeAPIToken / LookupAPIToken
// flows are verified in the integration test suite (see §8 of the plan).

func TestGenerateAPIToken_Structure(t *testing.T) {
	const prefix = "sk_live_"
	plaintext, lookup, hash, err := generateAPIToken(prefix)
	if err != nil {
		t.Fatalf("generateAPIToken: %v", err)
	}

	// Plaintext must start with the configured prefix so the gateway's
	// HasPrefix check in APITokenOrJWT middleware actually matches.
	if !strings.HasPrefix(plaintext, prefix) {
		t.Errorf("plaintext = %q, want prefix %q", plaintext, prefix)
	}
	// Lookup must be exactly 12 base62 chars — this is what the VARCHAR(16)
	// token_lookup column and the unique index on (prefix, lookup) expect.
	if len(lookup) != 12 {
		t.Errorf("lookup length = %d, want 12", len(lookup))
	}
	// Hash must be the 32-byte SHA-256 digest so the DB BYTEA column always
	// stores fixed-width rows and subtle.ConstantTimeCompare works.
	if len(hash) != sha256.Size {
		t.Errorf("hash length = %d, want %d", len(hash), sha256.Size)
	}
}

func TestGenerateAPIToken_RoundTrip(t *testing.T) {
	const prefix = "sk_live_"
	plaintext, lookup, hash, err := generateAPIToken(prefix)
	if err != nil {
		t.Fatalf("generateAPIToken: %v", err)
	}

	// Re-parse the plaintext and confirm the lookup matches the stored
	// lookup and the SHA-256 of the secret matches the stored hash. This is
	// the round-trip the gateway performs on every request.
	gotLookup, secret, err := ParseAPIToken(plaintext, prefix)
	if err != nil {
		t.Fatalf("ParseAPIToken: %v", err)
	}
	if gotLookup != lookup {
		t.Errorf("parsed lookup = %q, want %q", gotLookup, lookup)
	}
	computed := sha256.Sum256([]byte(secret))
	if string(computed[:]) != string(hash) {
		t.Errorf("sha256(secret) did not match stored hash")
	}
}

func TestGenerateAPIToken_Uniqueness(t *testing.T) {
	seen := make(map[string]struct{}, 100)
	for i := 0; i < 100; i++ {
		_, lookup, _, err := generateAPIToken("sk_live_")
		if err != nil {
			t.Fatalf("generateAPIToken iteration %d: %v", i, err)
		}
		if _, dup := seen[lookup]; dup {
			t.Errorf("duplicate lookup produced at iteration %d: %q", i, lookup)
		}
		seen[lookup] = struct{}{}
	}
}

func TestParseAPIToken(t *testing.T) {
	cases := []struct {
		name       string
		raw        string
		prefix     string
		wantErr    bool
		wantLookup string
		wantSecret string
	}{
		{
			name:       "happy path",
			raw:        "sk_live_abc123_topsecret",
			prefix:     "sk_live_",
			wantLookup: "abc123",
			wantSecret: "topsecret",
		},
		{
			name:    "missing prefix",
			raw:     "abc123_topsecret",
			prefix:  "sk_live_",
			wantErr: true,
		},
		{
			name:    "empty string",
			raw:     "",
			prefix:  "sk_live_",
			wantErr: true,
		},
		{
			name:    "empty prefix config rejected",
			raw:     "abc_secret",
			prefix:  "",
			wantErr: true,
		},
		{
			name:    "no separator",
			raw:     "sk_live_abc123topsecret",
			prefix:  "sk_live_",
			wantErr: true,
		},
		{
			name:    "separator at end",
			raw:     "sk_live_abc123_",
			prefix:  "sk_live_",
			wantErr: true,
		},
		{
			name:    "separator at start (empty lookup)",
			raw:     "sk_live__topsecret",
			prefix:  "sk_live_",
			wantErr: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			lookup, secret, err := ParseAPIToken(c.raw, c.prefix)
			if c.wantErr {
				if err == nil {
					t.Errorf("expected error, got lookup=%q secret=%q", lookup, secret)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if lookup != c.wantLookup {
				t.Errorf("lookup = %q, want %q", lookup, c.wantLookup)
			}
			if secret != c.wantSecret {
				t.Errorf("secret = %q, want %q", secret, c.wantSecret)
			}
		})
	}
}

func TestGenerateBase62_LengthAndAlphabet(t *testing.T) {
	// Exact width is a load-bearing invariant: token_lookup is indexed as
	// VARCHAR(16), and downstream parsers split on the "_" between lookup
	// and secret, so both halves must never exceed their nominal length.
	cases := []struct {
		name   string
		n      int
		trials int
	}{
		{"lookup width", apiTokenLookupChars, 64},
		{"secret width", apiTokenSecretChars, 32},
		{"edge 1", 1, 32},
		{"edge 100", 100, 4},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for i := 0; i < c.trials; i++ {
				got, err := generateBase62(c.n)
				if err != nil {
					t.Fatalf("generateBase62: %v", err)
				}
				if len(got) != c.n {
					t.Errorf("trial %d len = %d, want %d (value %q)", i, len(got), c.n, got)
				}
				// Every character must be in the base62 alphabet.
				for j, r := range got {
					if !strings.ContainsRune(base62Alphabet, r) {
						t.Errorf("trial %d char %d (%q) not in base62 alphabet", i, j, r)
					}
				}
			}
		})
	}
}

func TestMapAPITokenError(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantNil  bool
	}{
		{"nil passes through", nil, 0, true},
		{"not found → 404", domain.ErrAPITokenNotFound, http.StatusNotFound, false},
		{"revoked → 403", domain.ErrAPITokenRevoked, http.StatusForbidden, false},
		{"expired → 403", domain.ErrAPITokenExpired, http.StatusForbidden, false},
		{"malformed → 400", domain.ErrAPITokenInvalidFormat, http.StatusBadRequest, false},
		{"scope denied → 403", domain.ErrAPITokenScopeNotGranted, http.StatusForbidden, false},
		{"insufficient role → 403", domain.ErrInsufficientRole, http.StatusForbidden, false},
		{"unknown → 500", errors.New("boom"), http.StatusInternalServerError, false},
		{
			name:     "wrapped not-found still maps",
			err:      wrap(domain.ErrAPITokenNotFound),
			wantCode: http.StatusNotFound,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := mapAPITokenError(c.err, "failed")
			if c.wantNil {
				if got != nil {
					t.Errorf("got %v, want nil", got)
				}
				return
			}
			var appErr *apperrors.AppError
			if !errors.As(got, &appErr) {
				t.Fatalf("got %T, want *apperrors.AppError", got)
			}
			if appErr.Status != c.wantCode {
				t.Errorf("Status = %d, want %d", appErr.Status, c.wantCode)
			}
		})
	}
}

// wrap returns an error that wraps inner so errors.Is still walks through.
// We use a trivial struct instead of fmt.Errorf to keep the test hermetic.
type wrappedErr struct{ inner error }

func (w wrappedErr) Error() string { return "wrapped: " + w.inner.Error() }
func (w wrappedErr) Unwrap() error { return w.inner }

func wrap(err error) error { return wrappedErr{inner: err} }
