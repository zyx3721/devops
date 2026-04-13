// Package system 定义系统管理相关的数据模型
//
// 本包包含与系统管理和安全相关的所有数据模型，包括：
//   - 用户：用户账号、认证信息
//   - RBAC：角色、权限、用户角色关联
//   - 权限：资源权限、操作权限
//   - 审计：操作日志、变更记录
//   - OA：OA系统集成、地址配置
//   - 安全：安全扫描、漏洞报告
//
// 使用示例:
//
//	import "devops/internal/models/system"
//
//	// 创建用户
//	user := &system.User{
//	    Username: "admin",
//	    Email:    "admin@example.com",
//	}
package system
