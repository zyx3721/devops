package security

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"devops/internal/models"
	"devops/pkg/dto"
	"devops/pkg/logger"
)

// TrivyScanner Trivy扫描器
type TrivyScanner struct {
	trivyPath string
}

// NewTrivyScanner 创建Trivy扫描器
func NewTrivyScanner() *TrivyScanner {
	// 尝试多个可能的路径
	paths := []string{
		"trivy",                              // PATH 中
		"D:\\tools\\trivy\\trivy\\trivy.exe", // Windows 自定义路径
		"/usr/local/bin/trivy",               // Linux
		"/usr/bin/trivy",                     // Linux
	}

	for _, p := range paths {
		cmd := exec.Command(p, "--version")
		if err := cmd.Run(); err == nil {
			return &TrivyScanner{trivyPath: p}
		}
	}

	return &TrivyScanner{trivyPath: "trivy"}
}

// TrivyOutput Trivy输出结构
type TrivyOutput struct {
	Results []TrivyResult `json:"Results"`
}

// TrivyResult Trivy结果
type TrivyResult struct {
	Target          string               `json:"Target"`
	Vulnerabilities []TrivyVulnerability `json:"Vulnerabilities"`
}

// TrivyVulnerability Trivy漏洞
type TrivyVulnerability struct {
	VulnerabilityID  string   `json:"VulnerabilityID"`
	PkgName          string   `json:"PkgName"`
	InstalledVersion string   `json:"InstalledVersion"`
	FixedVersion     string   `json:"FixedVersion"`
	Severity         string   `json:"Severity"`
	Title            string   `json:"Title"`
	Description      string   `json:"Description"`
	References       []string `json:"References"`
}

// ScanResult 扫描结果
type ScanResult struct {
	RiskLevel       string
	VulnSummary     dto.VulnSummary
	Vulnerabilities []dto.Vulnerability
}

// Scan 执行扫描
func (s *TrivyScanner) Scan(ctx context.Context, image string, registry *models.ImageRegistry) (*ScanResult, error) {
	log := logger.L().WithField("image", image)

	// 构建命令参数
	args := []string{"image", "--format", "json", "--quiet"}

	// 如果有仓库凭证，设置环境变量
	var env []string
	if registry != nil && registry.Username != "" {
		env = append(env, fmt.Sprintf("TRIVY_USERNAME=%s", registry.Username))
		env = append(env, fmt.Sprintf("TRIVY_PASSWORD=%s", registry.Password))
	}

	args = append(args, image)

	// 执行Trivy命令
	cmd := exec.CommandContext(ctx, s.trivyPath, args...)
	if len(env) > 0 {
		cmd.Env = append(cmd.Environ(), env...)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	log.Info("执行Trivy扫描")
	err := cmd.Run()
	if err != nil {
		// 检查是否是Trivy未安装
		if strings.Contains(err.Error(), "executable file not found") {
			log.Error("Trivy未安装")
			return nil, fmt.Errorf("Trivy未安装，请先安装Trivy: https://aquasecurity.github.io/trivy")
		}
		log.WithField("stderr", stderr.String()).Error("Trivy扫描失败")
		return nil, fmt.Errorf("扫描失败: %s", stderr.String())
	}

	// 解析输出
	var output TrivyOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		log.WithField("error", err).Error("解析Trivy输出失败")
		return nil, fmt.Errorf("解析扫描结果失败: %v", err)
	}

	// 转换结果
	result := &ScanResult{
		Vulnerabilities: make([]dto.Vulnerability, 0),
	}

	for _, r := range output.Results {
		for _, v := range r.Vulnerabilities {
			vuln := dto.Vulnerability{
				VulnID:       v.VulnerabilityID,
				PkgName:      v.PkgName,
				InstalledVer: v.InstalledVersion,
				FixedVer:     v.FixedVersion,
				Severity:     strings.ToLower(v.Severity),
				Title:        v.Title,
				Description:  v.Description,
				References:   v.References,
			}
			result.Vulnerabilities = append(result.Vulnerabilities, vuln)

			// 统计
			switch strings.ToLower(v.Severity) {
			case "critical":
				result.VulnSummary.Critical++
			case "high":
				result.VulnSummary.High++
			case "medium":
				result.VulnSummary.Medium++
			case "low":
				result.VulnSummary.Low++
			}
		}
	}

	result.VulnSummary.Total = result.VulnSummary.Critical + result.VulnSummary.High +
		result.VulnSummary.Medium + result.VulnSummary.Low

	// 确定风险等级
	result.RiskLevel = s.determineRiskLevel(result.VulnSummary)

	return result, nil
}

// determineRiskLevel 确定风险等级
func (s *TrivyScanner) determineRiskLevel(summary dto.VulnSummary) string {
	if summary.Critical > 0 {
		return "critical"
	}
	if summary.High > 0 {
		return "high"
	}
	if summary.Medium > 0 {
		return "medium"
	}
	if summary.Low > 0 {
		return "low"
	}
	return "none"
}

// CheckInstalled 检查Trivy是否安装
func (s *TrivyScanner) CheckInstalled() bool {
	cmd := exec.Command(s.trivyPath, "--version")
	return cmd.Run() == nil
}
