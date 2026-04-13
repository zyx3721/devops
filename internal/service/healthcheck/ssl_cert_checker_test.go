package healthcheck

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSSLCertChecker_ParseDomain(t *testing.T) {
	checker := NewSSLCertChecker(10 * time.Second)

	tests := []struct {
		name         string
		domain       string
		expectedHost string
		expectedPort string
	}{
		{
			name:         "domain without port",
			domain:       "example.com",
			expectedHost: "example.com",
			expectedPort: "443",
		},
		{
			name:         "domain with port",
			domain:       "example.com:8443",
			expectedHost: "example.com",
			expectedPort: "8443",
		},
		{
			name:         "domain with https prefix",
			domain:       "https://example.com",
			expectedHost: "example.com",
			expectedPort: "443",
		},
		{
			name:         "domain with https prefix and port",
			domain:       "https://example.com:8443",
			expectedHost: "example.com",
			expectedPort: "8443",
		},
		{
			name:         "domain with path",
			domain:       "example.com/path/to/resource",
			expectedHost: "example.com",
			expectedPort: "443",
		},
		{
			name:         "domain with port and path",
			domain:       "example.com:8443/path",
			expectedHost: "example.com",
			expectedPort: "8443",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, port := checker.parseDomain(tt.domain)
			assert.Equal(t, tt.expectedHost, host, "Host should match")
			assert.Equal(t, tt.expectedPort, port, "Port should match")
		})
	}
}

func TestSSLCertChecker_CalculateDaysRemaining(t *testing.T) {
	checker := NewSSLCertChecker(10 * time.Second)

	tests := []struct {
		name         string
		expiryDate   time.Time
		expectedDays int
		description  string
	}{
		{
			name:         "certificate expires in 30 days",
			expiryDate:   time.Now().Add(30 * 24 * time.Hour),
			expectedDays: 30,
			description:  "Should return 30 days",
		},
		{
			name:         "certificate expires in 7 days",
			expiryDate:   time.Now().Add(7 * 24 * time.Hour),
			expectedDays: 7,
			description:  "Should return 7 days",
		},
		{
			name:         "certificate expires in 1 day",
			expiryDate:   time.Now().Add(24 * time.Hour),
			expectedDays: 1,
			description:  "Should return 1 day",
		},
		{
			name:         "certificate expired 1 day ago",
			expiryDate:   time.Now().Add(-24 * time.Hour),
			expectedDays: -1,
			description:  "Should return -1 day",
		},
		{
			name:         "certificate expired 30 days ago",
			expiryDate:   time.Now().Add(-30 * 24 * time.Hour),
			expectedDays: -30,
			description:  "Should return -30 days",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			days := checker.calculateDaysRemaining(tt.expiryDate)
			// Allow for 1 day difference due to timing
			assert.InDelta(t, tt.expectedDays, days, 1, tt.description)
		})
	}
}

func TestSSLCertChecker_DetermineAlertLevel(t *testing.T) {
	checker := NewSSLCertChecker(10 * time.Second)

	tests := []struct {
		name          string
		daysRemaining int
		criticalDays  int
		warningDays   int
		noticeDays    int
		expectedLevel string
	}{
		{
			name:          "expired certificate",
			daysRemaining: -1,
			criticalDays:  7,
			warningDays:   30,
			noticeDays:    60,
			expectedLevel: "expired",
		},
		{
			name:          "critical level - 0 days",
			daysRemaining: 0,
			criticalDays:  7,
			warningDays:   30,
			noticeDays:    60,
			expectedLevel: "critical",
		},
		{
			name:          "critical level - 5 days",
			daysRemaining: 5,
			criticalDays:  7,
			warningDays:   30,
			noticeDays:    60,
			expectedLevel: "critical",
		},
		{
			name:          "warning level - 7 days",
			daysRemaining: 7,
			criticalDays:  7,
			warningDays:   30,
			noticeDays:    60,
			expectedLevel: "warning",
		},
		{
			name:          "warning level - 20 days",
			daysRemaining: 20,
			criticalDays:  7,
			warningDays:   30,
			noticeDays:    60,
			expectedLevel: "warning",
		},
		{
			name:          "notice level - 30 days",
			daysRemaining: 30,
			criticalDays:  7,
			warningDays:   30,
			noticeDays:    60,
			expectedLevel: "notice",
		},
		{
			name:          "notice level - 45 days",
			daysRemaining: 45,
			criticalDays:  7,
			warningDays:   30,
			noticeDays:    60,
			expectedLevel: "notice",
		},
		{
			name:          "normal level - 60 days",
			daysRemaining: 60,
			criticalDays:  7,
			warningDays:   30,
			noticeDays:    60,
			expectedLevel: "normal",
		},
		{
			name:          "normal level - 90 days",
			daysRemaining: 90,
			criticalDays:  7,
			warningDays:   30,
			noticeDays:    60,
			expectedLevel: "normal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level := checker.determineAlertLevel(tt.daysRemaining, tt.criticalDays, tt.warningDays, tt.noticeDays)
			assert.Equal(t, tt.expectedLevel, level, "Alert level should match")
		})
	}
}

func TestSSLCertChecker_DetermineAlertLevel_EdgeCases(t *testing.T) {
	checker := NewSSLCertChecker(10 * time.Second)

	tests := []struct {
		name          string
		daysRemaining int
		criticalDays  int
		warningDays   int
		noticeDays    int
		expectedLevel string
		description   string
	}{
		{
			name:          "boundary - exactly critical threshold",
			daysRemaining: 7,
			criticalDays:  7,
			warningDays:   30,
			noticeDays:    60,
			expectedLevel: "warning",
			description:   "When days remaining equals critical threshold, should be warning",
		},
		{
			name:          "boundary - one day before critical",
			daysRemaining: 6,
			criticalDays:  7,
			warningDays:   30,
			noticeDays:    60,
			expectedLevel: "critical",
			description:   "When days remaining is less than critical threshold, should be critical",
		},
		{
			name:          "boundary - exactly warning threshold",
			daysRemaining: 30,
			criticalDays:  7,
			warningDays:   30,
			noticeDays:    60,
			expectedLevel: "notice",
			description:   "When days remaining equals warning threshold, should be notice",
		},
		{
			name:          "boundary - exactly notice threshold",
			daysRemaining: 60,
			criticalDays:  7,
			warningDays:   30,
			noticeDays:    60,
			expectedLevel: "normal",
			description:   "When days remaining equals notice threshold, should be normal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level := checker.determineAlertLevel(tt.daysRemaining, tt.criticalDays, tt.warningDays, tt.noticeDays)
			assert.Equal(t, tt.expectedLevel, level, tt.description)
		})
	}
}

// TestSSLCertChecker_CheckSSLCert_RealDomain tests with a real domain
// This test may fail if there are network issues or the domain is unreachable
func TestSSLCertChecker_CheckSSLCert_RealDomain(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	checker := NewSSLCertChecker(10 * time.Second)

	// Test with multiple well-known domains in case one is unreachable
	domains := []string{"www.baidu.com", "www.github.com", "www.cloudflare.com"}

	var lastErr error
	for _, domain := range domains {
		result, err := checker.CheckSSLCert(domain)

		if err == nil {
			// Success! Verify the result
			assert.NotNil(t, result, "Result should not be nil")
			assert.Equal(t, "healthy", result.Status, "Status should be healthy")
			assert.NotEmpty(t, result.Issuer, "Issuer should not be empty")
			assert.NotEmpty(t, result.Subject, "Subject should not be empty")
			assert.NotEmpty(t, result.SerialNumber, "Serial number should not be empty")
			assert.False(t, result.ExpiryDate.IsZero(), "Expiry date should not be zero")
			assert.Greater(t, result.DaysRemaining, 0, "Days remaining should be positive")
			assert.Greater(t, result.ResponseTimeMs, int64(0), "Response time should be positive")
			return // Test passed
		}

		lastErr = err
		t.Logf("Failed to check %s: %v, trying next domain...", domain, err)
	}

	// If all domains failed, skip the test (likely network issue)
	t.Skipf("All test domains unreachable, likely network issue. Last error: %v", lastErr)
}

func TestSSLCertChecker_CheckSSLCertWithAlertLevel(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	checker := NewSSLCertChecker(10 * time.Second)

	// Test with multiple well-known domains
	domains := []string{"www.baidu.com", "www.github.com", "www.cloudflare.com"}

	var lastErr error
	for _, domain := range domains {
		result, err := checker.CheckSSLCertWithAlertLevel(domain, 7, 30, 60)

		if err == nil {
			// Success! Verify the result
			assert.NotNil(t, result, "Result should not be nil")
			assert.Equal(t, "healthy", result.Status, "Status should be healthy")
			assert.NotEmpty(t, result.AlertLevel, "Alert level should not be empty")
			// Most major sites should have more than 60 days remaining
			assert.Contains(t, []string{"normal", "notice"}, result.AlertLevel, "Alert level should be normal or notice")
			return // Test passed
		}

		lastErr = err
		t.Logf("Failed to check %s: %v, trying next domain...", domain, err)
	}

	// If all domains failed, skip the test (likely network issue)
	t.Skipf("All test domains unreachable, likely network issue. Last error: %v", lastErr)
}

func TestSSLCertChecker_CheckSSLCert_InvalidDomain(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	checker := NewSSLCertChecker(5 * time.Second)

	// Test with an invalid domain
	result, err := checker.CheckSSLCert("invalid-domain-that-does-not-exist-12345.com")

	assert.Error(t, err, "Should return error for invalid domain")
	assert.NotNil(t, result, "Result should not be nil")
	assert.Equal(t, "unhealthy", result.Status, "Status should be unhealthy")
	assert.NotEmpty(t, result.ErrorMsg, "Error message should not be empty")
}

func TestSSLCertChecker_CheckSSLCert_WithCustomPort(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	checker := NewSSLCertChecker(10 * time.Second)

	// Test with multiple domains and custom port (443 is standard HTTPS port)
	domains := []string{"www.baidu.com:443", "www.github.com:443", "www.cloudflare.com:443"}

	var lastErr error
	for _, domain := range domains {
		result, err := checker.CheckSSLCert(domain)

		if err == nil {
			// Success! Verify the result
			assert.NotNil(t, result, "Result should not be nil")
			assert.Equal(t, "healthy", result.Status, "Status should be healthy")
			return // Test passed
		}

		lastErr = err
		t.Logf("Failed to check %s: %v, trying next domain...", domain, err)
	}

	// If all domains failed, skip the test (likely network issue)
	t.Skipf("All test domains unreachable, likely network issue. Last error: %v", lastErr)
}
