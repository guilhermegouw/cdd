package claude

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"
)

func TestGetChallenge(t *testing.T) {
	verifier, challenge, err := GetChallenge()
	if err != nil {
		t.Fatalf("GetChallenge() error = %v", err)
	}

	// Verifier should be base64url encoded (43 chars from 32 bytes).
	if verifier == "" {
		t.Error("GetChallenge() returned empty verifier")
	}

	// Challenge should be base64url encoded SHA256 hash (43 chars).
	if challenge == "" {
		t.Error("GetChallenge() returned empty challenge")
	}

	// Both should be URL-safe base64 (no +, /, or =).
	if strings.ContainsAny(verifier, "+/=") {
		t.Errorf("verifier contains non-URL-safe characters: %s", verifier)
	}
	if strings.ContainsAny(challenge, "+/=") {
		t.Errorf("challenge contains non-URL-safe characters: %s", challenge)
	}
}

func TestGetChallenge_Uniqueness(t *testing.T) {
	// Generate multiple challenges and ensure they're unique.
	seen := make(map[string]bool)
	iterations := 100

	for i := 0; i < iterations; i++ {
		verifier, challenge, err := GetChallenge()
		if err != nil {
			t.Fatalf("GetChallenge() iteration %d error = %v", i, err)
		}

		if seen[verifier] {
			t.Errorf("duplicate verifier generated at iteration %d", i)
		}
		seen[verifier] = true

		if seen[challenge] {
			t.Errorf("duplicate challenge generated at iteration %d", i)
		}
		seen[challenge] = true
	}
}

func TestGetChallenge_VerifierAndChallengeAreDifferent(t *testing.T) {
	verifier, challenge, err := GetChallenge()
	if err != nil {
		t.Fatalf("GetChallenge() error = %v", err)
	}

	if verifier == challenge {
		t.Error("verifier and challenge should be different")
	}
}

func TestEncodeBase64(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
	}{
		{
			name:  "simple bytes",
			input: []byte("hello"),
		},
		{
			name:  "bytes with padding",
			input: []byte("a"), // Would normally have == padding.
		},
		{
			name:  "bytes with + and /",
			input: []byte{0xfb, 0xff, 0xfe}, // Contains chars that map to + and / in standard base64.
		},
		{
			name:  "empty input",
			input: []byte{},
		},
		{
			name:  "32 random bytes",
			input: make([]byte, 32),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := encodeBase64(tt.input)

			// Should not contain standard base64 padding or special chars.
			if strings.Contains(result, "=") {
				t.Errorf("encodeBase64() contains = padding: %s", result)
			}
			if strings.Contains(result, "+") {
				t.Errorf("encodeBase64() contains +: %s", result)
			}
			if strings.Contains(result, "/") {
				t.Errorf("encodeBase64() contains /: %s", result)
			}

			// Should only contain URL-safe base64 characters.
			for _, c := range result {
				if !isURLSafeBase64Char(c) {
					t.Errorf("encodeBase64() contains invalid char %c: %s", c, result)
				}
			}
		})
	}
}

func TestEncodeBase64_Reversible(t *testing.T) {
	// Test that we can decode the result back to original.
	input := []byte("test input data")
	encoded := encodeBase64(input)

	// Convert back from URL-safe to standard base64.
	standard := strings.ReplaceAll(encoded, "-", "+")
	standard = strings.ReplaceAll(standard, "_", "/")

	// Add padding if needed.
	switch len(standard) % 4 {
	case 2:
		standard += "=="
	case 3:
		standard += "="
	}

	decoded, err := base64.StdEncoding.DecodeString(standard)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	if !bytes.Equal(decoded, input) {
		t.Errorf("Round-trip failed: got %q, want %q", string(decoded), string(input))
	}
}

func isURLSafeBase64Char(c rune) bool {
	return (c >= 'A' && c <= 'Z') ||
		(c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9') ||
		c == '-' || c == '_'
}
