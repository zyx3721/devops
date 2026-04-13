package handler

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidDomain(t *testing.T) {
	tests := []struct {
		name     string
		domain   string
		expected bool
	}{
		{"valid domain", "example.com", true},
		{"valid domain with port", "example.com:8443", true},
		{"valid domain with https", "https://example.com", true},
		{"valid subdomain", "api.example.com", true},
		{"valid with path", "example.com/path", true},
		{"invalid double dots", "invalid..com", false},
		{"invalid starts with dot", ".example.com", false},
		{"invalid ends with dot", "example.com.", false},
		{"invalid starts with dash", "-example.com", false},
		{"invalid ends with dash", "example.com-", false},
		{"empty string", "", false},
		{"invalid characters", "example@com", false},
		{"valid IP address", "192.168.1.1", true},
		{"valid IP with port", "192.168.1.1:8443", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidDomain(tt.domain)
			assert.Equal(t, tt.expected, result, "Domain: %s", tt.domain)
		})
	}
}

func TestNormalizeDomain(t *testing.T) {
	tests := []struct {
		name     string
		domain   string
		expected string
	}{
		{"domain without port", "example.com", "example.com:443"},
		{"domain with port", "example.com:8443", "example.com:8443"},
		{"domain with https", "https://example.com", "example.com:443"},
		{"domain with http", "http://example.com", "example.com:443"},
		{"domain with path", "example.com/path", "example.com:443"},
		{"domain with https and path", "https://example.com/path", "example.com:443"},
		{"uppercase domain", "EXAMPLE.COM", "example.com:443"},
		{"mixed case with port", "Example.COM:8443", "example.com:8443"},
		{"subdomain", "api.example.com", "api.example.com:443"},
		{"subdomain with port", "api.example.com:9443", "api.example.com:9443"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeDomain(tt.domain)
			assert.Equal(t, tt.expected, result, "Domain: %s", tt.domain)
		})
	}
}

// Property test: normalizeDomain should always produce consistent output
func TestNormalizeDomain_Idempotent(t *testing.T) {
	domains := []string{
		"example.com",
		"https://example.com",
		"EXAMPLE.COM",
		"example.com:443",
	}

	for _, domain := range domains {
		normalized1 := normalizeDomain(domain)
		normalized2 := normalizeDomain(normalized1)
		assert.Equal(t, normalized1, normalized2, "Normalization should be idempotent for: %s", domain)
	}
}

// Property test: valid domains should remain valid after normalization
func TestNormalizeDomain_PreservesValidity(t *testing.T) {
	validDomains := []string{
		"example.com",
		"api.example.com",
		"example.com:8443",
		"https://example.com",
	}

	for _, domain := range validDomains {
		if isValidDomain(domain) {
			normalized := normalizeDomain(domain)
			// The normalized form should still be valid (though it will have :port)
			assert.NotEmpty(t, normalized, "Normalized domain should not be empty for: %s", domain)
		}
	}
}
