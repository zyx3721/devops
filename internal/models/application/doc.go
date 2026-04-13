// Package application 定义应用管理相关的数据模型
//
// 本包包含与应用生命周期管理相关的所有数据模型，包括：
//   - 应用：应用基本信息、配置、环境
//   - 应用组：应用分组、项目管理
//
// 使用示例:
//
//	import "devops/internal/models/application"
//
//	// 创建应用
//	app := &application.Application{
//	    Name:        "my-service",
//	    DisplayName: "我的服务",
//	    GitRepo:     "https://github.com/example/my-service",
//	}
package application
