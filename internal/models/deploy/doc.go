// Package deploy 定义部署流程相关的数据模型
//
// 本包包含与应用部署和发布流程相关的所有数据模型，包括：
//   - 部署记录：发布历史、版本信息、回滚记录
//   - 审批流程：审批链、审批节点、审批实例
//   - 流水线：Pipeline 定义、阶段、步骤、运行记录
//
// 使用示例:
//
//	import "devops/internal/models/deploy"
//
//	// 创建部署记录
//	record := &deploy.DeployRecord{
//	    AppID:   1,
//	    Version: "v1.2.0",
//	    Status:  "success",
//	}
package deploy
