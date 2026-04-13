package middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"

	apperrors "devops/pkg/errors"
)

// RequireSuperAdmin 检查是否是超级管理员
func RequireSuperAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := GetRole(c)
		if !exists {
			Error(c, apperrors.ErrUnauthorized)
			c.Abort()
			return
		}

		if role != "admin" && role != "super_admin" {
			Error(c, apperrors.New(apperrors.ErrCodeForbidden, fmt.Sprintf("权限不足，请检查用户是否有访问权限。当前角色: %s, 需要: admin 或 super_admin", role)))
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAdmin 检查是否是管理员（别名，兼容旧代码）
func RequireAdmin() gin.HandlerFunc {
	return RequireSuperAdmin()
}
