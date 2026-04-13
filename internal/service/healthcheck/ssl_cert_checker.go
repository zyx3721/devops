package healthcheck

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"devops/pkg/logger"
)

// ErrorType 错误类型定义
type ErrorType string

const (
	ErrorTypeDNS            ErrorType = "dns_resolution"
	ErrorTypeTLSConnection  ErrorType = "tls_connection"
	ErrorTypeCertValidation ErrorType = "cert_validation"
	ErrorTypeTimeout        ErrorType = "timeout"
	ErrorTypeCertParsing    ErrorType = "cert_parsing"
	ErrorTypeUnknown        ErrorType = "unknown"
)

// CertCheckError 证书检查错误
type CertCheckError struct {
	Type    ErrorType
	Message string
	Err     error
	Retry   bool // 是否可以重试
}

func (e *CertCheckError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *CertCheckError) Unwrap() error {
	return e.Err
}

// SSLCertChecker SSL证书检查器
type SSLCertChecker struct {
	timeout    time.Duration
	log        *logger.Logger
	retryCount int           // 重试次数
	retryDelay time.Duration // 初始重试延迟
}

// CertCheckResult 证书检查结果
type CertCheckResult struct {
	Status         string    // healthy/unhealthy
	ErrorMsg       string    // 错误信息
	ErrorType      string    // 错误类型
	ResponseTimeMs int64     // 响应时间（毫秒）
	ExpiryDate     time.Time // 证书过期时间
	DaysRemaining  int       // 剩余天数
	Issuer         string    // 颁发者
	Subject        string    // 主题
	SerialNumber   string    // 序列号
	AlertLevel     string    // 告警级别
	RetryAttempts  int       // 重试次数
}

// NewSSLCertChecker 创建SSL证书检查器
func NewSSLCertChecker(timeout time.Duration) *SSLCertChecker {
	return &SSLCertChecker{
		timeout:    timeout,
		log:        logger.NewLogger("ssl-cert-checker"),
		retryCount: 3,               // 默认重试3次
		retryDelay: 1 * time.Second, // 初始延迟1秒
	}
}

// parseDomain 解析域名和端口
// 输入: domain (example.com 或 example.com:8443 或 https://example.com)
// 输出: host, port
func (c *SSLCertChecker) parseDomain(domain string) (string, string) {
	// 移除协议前缀
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "http://")

	// 移除路径部分
	if idx := strings.Index(domain, "/"); idx != -1 {
		domain = domain[:idx]
	}

	// 检查是否包含端口
	if strings.Contains(domain, ":") {
		host, port, err := net.SplitHostPort(domain)
		if err != nil {
			// 如果解析失败，返回原域名和默认端口
			return domain, "443"
		}
		return host, port
	}

	// 没有端口，使用默认端口443
	return domain, "443"
}

// calculateDaysRemaining 计算剩余天数
// 输入: expiryDate
// 输出: days (int)
func (c *SSLCertChecker) calculateDaysRemaining(expiryDate time.Time) int {
	duration := time.Until(expiryDate)
	days := int(duration.Hours() / 24)
	return days
}

// determineAlertLevel 判断告警级别
// 输入: daysRemaining, criticalDays, warningDays, noticeDays
// 输出: alertLevel (expired/critical/warning/notice/normal)
func (c *SSLCertChecker) determineAlertLevel(daysRemaining, criticalDays, warningDays, noticeDays int) string {
	if daysRemaining < 0 {
		return "expired"
	} else if daysRemaining < criticalDays {
		return "critical"
	} else if daysRemaining < warningDays {
		return "warning"
	} else if daysRemaining < noticeDays {
		return "notice"
	}
	return "normal"
}

// getCertificate 获取证书信息
// 输入: host, port
// 输出: *x509.Certificate
func (c *SSLCertChecker) getCertificate(host, port string) (*x509.Certificate, error) {
	// 设置TLS配置，支持SNI
	config := &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: false, // 默认验证证书
	}

	// 建立连接
	dialer := &net.Dialer{
		Timeout: c.timeout,
	}

	address := net.JoinHostPort(host, port)

	c.log.WithFields(map[string]interface{}{
		"host":    host,
		"port":    port,
		"address": address,
		"timeout": c.timeout,
	}).Debug("Attempting TLS connection")

	conn, err := tls.DialWithDialer(dialer, "tcp", address, config)
	if err != nil {
		return nil, c.classifyConnectionError(err, host, port)
	}
	defer conn.Close()

	// 获取连接状态
	state := conn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		certErr := &CertCheckError{
			Type:    ErrorTypeCertParsing,
			Message: "No certificates found in TLS connection",
			Retry:   false,
		}
		c.log.WithFields(map[string]interface{}{
			"host": host,
			"port": port,
		}).Error("No certificates found")
		return nil, certErr
	}

	// 返回第一个证书（服务器证书）
	cert := state.PeerCertificates[0]

	c.log.WithFields(map[string]interface{}{
		"host":      host,
		"port":      port,
		"subject":   cert.Subject.String(),
		"issuer":    cert.Issuer.String(),
		"not_after": cert.NotAfter,
		"serial":    cert.SerialNumber.String(),
	}).Debug("Certificate retrieved successfully")

	return cert, nil
}

// classifyConnectionError 分类连接错误
func (c *SSLCertChecker) classifyConnectionError(err error, host, port string) error {
	var netErr net.Error
	var dnsErr *net.DNSError
	var opErr *net.OpError

	// DNS解析错误
	if errors.As(err, &dnsErr) {
		certErr := &CertCheckError{
			Type:    ErrorTypeDNS,
			Message: fmt.Sprintf("DNS resolution failed for %s", host),
			Err:     err,
			Retry:   true, // DNS错误可以重试
		}
		c.log.WithFields(map[string]interface{}{
			"host":       host,
			"port":       port,
			"error_type": ErrorTypeDNS,
			"error":      err.Error(),
		}).Warn("DNS resolution failed")
		return certErr
	}

	// 超时错误
	if errors.As(err, &netErr) && netErr.Timeout() {
		certErr := &CertCheckError{
			Type:    ErrorTypeTimeout,
			Message: fmt.Sprintf("Connection timeout after %v", c.timeout),
			Err:     err,
			Retry:   true, // 超时可以重试
		}
		c.log.WithFields(map[string]interface{}{
			"host":       host,
			"port":       port,
			"error_type": ErrorTypeTimeout,
			"timeout":    c.timeout,
		}).Warn("Connection timeout")
		return certErr
	}

	// 操作错误（连接被拒绝等）
	if errors.As(err, &opErr) {
		retry := strings.Contains(err.Error(), "connection refused") ||
			strings.Contains(err.Error(), "network is unreachable")

		certErr := &CertCheckError{
			Type:    ErrorTypeTLSConnection,
			Message: fmt.Sprintf("TLS connection failed: %v", opErr.Err),
			Err:     err,
			Retry:   retry,
		}
		c.log.WithFields(map[string]interface{}{
			"host":       host,
			"port":       port,
			"error_type": ErrorTypeTLSConnection,
			"error":      err.Error(),
			"retry":      retry,
		}).Warn("TLS connection failed")
		return certErr
	}

	// 证书验证错误
	if strings.Contains(err.Error(), "certificate") ||
		strings.Contains(err.Error(), "x509") {
		certErr := &CertCheckError{
			Type:    ErrorTypeCertValidation,
			Message: fmt.Sprintf("Certificate validation failed: %v", err),
			Err:     err,
			Retry:   false, // 证书验证失败不重试
		}
		c.log.WithFields(map[string]interface{}{
			"host":       host,
			"port":       port,
			"error_type": ErrorTypeCertValidation,
			"error":      err.Error(),
		}).Warn("Certificate validation failed")
		return certErr
	}

	// 未知错误
	certErr := &CertCheckError{
		Type:    ErrorTypeUnknown,
		Message: fmt.Sprintf("Unknown error: %v", err),
		Err:     err,
		Retry:   false,
	}
	c.log.WithFields(map[string]interface{}{
		"host":       host,
		"port":       port,
		"error_type": ErrorTypeUnknown,
		"error":      err.Error(),
	}).Error("Unknown error occurred")
	return certErr
}

// CheckSSLCert 检查SSL证书
// 输入: domain (域名:端口格式，端口可选，默认443)
// 输出: CertCheckResult
func (c *SSLCertChecker) CheckSSLCert(domain string) (*CertCheckResult, error) {
	startTime := time.Now()
	result := &CertCheckResult{
		Status: "unhealthy",
	}

	// 解析域名和端口
	host, port := c.parseDomain(domain)

	c.log.WithFields(map[string]interface{}{
		"domain": domain,
		"host":   host,
		"port":   port,
	}).Info("Starting SSL certificate check")

	// 使用重试机制获取证书
	cert, err := c.getCertificateWithRetry(host, port)
	if err != nil {
		var certErr *CertCheckError
		if errors.As(err, &certErr) {
			result.ErrorMsg = certErr.Error()
			result.ErrorType = string(certErr.Type)
		} else {
			result.ErrorMsg = err.Error()
			result.ErrorType = string(ErrorTypeUnknown)
		}
		result.ResponseTimeMs = time.Since(startTime).Milliseconds()

		c.log.WithFields(map[string]interface{}{
			"domain":      domain,
			"error_type":  result.ErrorType,
			"error":       result.ErrorMsg,
			"duration_ms": result.ResponseTimeMs,
		}).Error("SSL certificate check failed")

		return result, err
	}

	// 提取证书信息
	result.ExpiryDate = cert.NotAfter
	result.Issuer = cert.Issuer.String()
	result.Subject = cert.Subject.String()
	result.SerialNumber = cert.SerialNumber.String()

	// 计算剩余天数
	result.DaysRemaining = c.calculateDaysRemaining(cert.NotAfter)

	// 设置状态为健康
	result.Status = "healthy"
	result.ResponseTimeMs = time.Since(startTime).Milliseconds()

	c.log.WithFields(map[string]interface{}{
		"domain":         domain,
		"days_remaining": result.DaysRemaining,
		"expiry_date":    result.ExpiryDate.Format("2006-01-02"),
		"issuer":         result.Issuer,
		"duration_ms":    result.ResponseTimeMs,
	}).Info("SSL certificate check completed successfully")

	return result, nil
}

// getCertificateWithRetry 使用重试机制获取证书
func (c *SSLCertChecker) getCertificateWithRetry(host, port string) (*x509.Certificate, error) {
	var lastErr error
	delay := c.retryDelay

	for attempt := 0; attempt <= c.retryCount; attempt++ {
		if attempt > 0 {
			c.log.WithFields(map[string]interface{}{
				"host":    host,
				"port":    port,
				"attempt": attempt,
				"delay":   delay,
			}).Info("Retrying certificate retrieval")

			time.Sleep(delay)
			// 指数退避，最大10秒
			delay *= 2
			if delay > 10*time.Second {
				delay = 10 * time.Second
			}
		}

		cert, err := c.getCertificate(host, port)
		if err == nil {
			if attempt > 0 {
				c.log.WithFields(map[string]interface{}{
					"host":    host,
					"port":    port,
					"attempt": attempt,
				}).Info("Certificate retrieval succeeded after retry")
			}
			return cert, nil
		}

		lastErr = err

		// 检查是否可以重试
		var certErr *CertCheckError
		if errors.As(err, &certErr) {
			if !certErr.Retry {
				c.log.WithFields(map[string]interface{}{
					"host":       host,
					"port":       port,
					"error_type": certErr.Type,
					"attempt":    attempt,
				}).Warn("Error is not retryable, stopping retry attempts")
				return nil, err
			}
		} else {
			// 未知错误，不重试
			return nil, err
		}
	}

	c.log.WithFields(map[string]interface{}{
		"host":        host,
		"port":        port,
		"retry_count": c.retryCount,
		"final_error": lastErr.Error(),
	}).Error("All retry attempts exhausted")

	return nil, lastErr
}

// CheckSSLCertWithAlertLevel 检查SSL证书并判断告警级别
// 输入: domain, criticalDays, warningDays, noticeDays
// 输出: CertCheckResult (包含告警级别)
func (c *SSLCertChecker) CheckSSLCertWithAlertLevel(domain string, criticalDays, warningDays, noticeDays int) (*CertCheckResult, error) {
	result, err := c.CheckSSLCert(domain)
	if err != nil {
		return result, err
	}

	// 判断告警级别
	result.AlertLevel = c.determineAlertLevel(result.DaysRemaining, criticalDays, warningDays, noticeDays)

	c.log.WithFields(map[string]interface{}{
		"domain":         domain,
		"days_remaining": result.DaysRemaining,
		"alert_level":    result.AlertLevel,
		"critical_days":  criticalDays,
		"warning_days":   warningDays,
		"notice_days":    noticeDays,
	}).Debug("Alert level determined")

	return result, nil
}
