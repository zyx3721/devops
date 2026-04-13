package ai

import (
	"context"

	"gorm.io/gorm"

	"devops/internal/models/ai"
	"devops/internal/repository"
)

// InitKnowledgeBase 初始化知识库数据
func InitKnowledgeBase(ctx context.Context, db *gorm.DB) error {
	service := NewKnowledgeService(db)

	// 检查是否已有数据
	items, total, err := service.List(ctx, repository.AIKnowledgeFilter{}, 1, 1)
	if err != nil {
		return err
	}
	if total > 0 && len(items) > 0 {
		return nil // 已有数据，跳过初始化
	}

	// 初始化知识库数据
	knowledgeData := getInitialKnowledgeData()

	for _, doc := range knowledgeData {
		_, err := service.AddDocument(ctx, doc, 0)
		if err != nil {
			// 忽略错误，继续添加其他文档
			continue
		}
	}

	return nil
}

// getInitialKnowledgeData 获取初始知识库数据
func getInitialKnowledgeData() []ai.Document {
	return []ai.Document{
		// 应用管理
		{
			Title:    "应用管理概述",
			Category: string(ai.CategoryApplication),
			Tags:     []string{"应用", "管理", "入门"},
			Content: `# 应用管理

## 功能介绍
应用管理模块用于管理平台上的所有应用，包括应用的创建、配置、部署和监控。

## 主要功能
1. **应用列表**: 查看所有应用及其状态
2. **应用详情**: 查看应用的详细信息，包括环境配置、部署历史等
3. **环境管理**: 管理应用的不同环境（开发、测试、预发、生产）
4. **部署操作**: 执行应用的部署、回滚、重启等操作

## 常用操作
- 创建应用: 点击"新建应用"按钮，填写应用信息
- 部署应用: 在应用详情页选择环境，点击"部署"
- 查看日志: 在应用详情页点击"日志"标签
- 扩缩容: 在应用详情页调整副本数量`,
		},
		{
			Title:    "应用部署指南",
			Category: string(ai.CategoryApplication),
			Tags:     []string{"部署", "发布", "操作"},
			Content: `# 应用部署指南

## 部署流程
1. 选择要部署的应用和环境
2. 选择部署版本（镜像标签）
3. 确认部署参数
4. 提交部署（如需审批会进入审批流程）
5. 等待部署完成

## 部署方式
- **普通部署**: 直接更新所有Pod
- **滚动部署**: 逐步更新Pod，保证服务不中断
- **金丝雀发布**: 先部署少量实例验证，再全量发布

## 回滚操作
如果部署后发现问题，可以快速回滚到上一个版本：
1. 进入应用详情页
2. 点击"部署历史"
3. 选择要回滚的版本
4. 点击"回滚"按钮`,
		},

		// 流量治理
		{
			Title:    "流量治理概述",
			Category: string(ai.CategoryTraffic),
			Tags:     []string{"流量", "治理", "入门"},
			Content: `# 流量治理

## 功能介绍
流量治理模块提供服务网格级别的流量管理能力，基于Istio实现。

## 主要功能
1. **限流**: 控制服务的请求速率，防止过载
2. **熔断**: 当服务异常时自动熔断，防止故障扩散
3. **路由**: 基于请求特征进行流量路由
4. **负载均衡**: 配置服务的负载均衡策略
5. **超时重试**: 配置请求超时和重试策略
6. **流量镜像**: 复制流量用于测试
7. **故障注入**: 模拟故障进行混沌测试

## 使用场景
- 保护服务不被突发流量击垮
- 实现灰度发布和A/B测试
- 故障隔离和快速恢复`,
		},
		{
			Title:    "限流配置指南",
			Category: string(ai.CategoryTraffic),
			Tags:     []string{"限流", "配置"},
			Content: `# 限流配置指南

## 限流类型
1. **全局限流**: 对整个服务的请求进行限流
2. **路由限流**: 对特定路由的请求进行限流
3. **用户限流**: 基于用户维度进行限流

## 配置参数
- **QPS**: 每秒允许的请求数
- **并发数**: 同时处理的最大请求数
- **突发容量**: 允许的突发请求数

## 配置示例
限制服务每秒最多处理100个请求：
- QPS: 100
- 突发容量: 20
- 限流响应: 返回429状态码`,
		},

		// 审批流程
		{
			Title:    "审批流程概述",
			Category: string(ai.CategoryApproval),
			Tags:     []string{"审批", "流程", "入门"},
			Content: `# 审批流程

## 功能介绍
审批流程模块用于管理需要审批的操作，如生产环境部署、敏感配置变更等。

## 审批链配置
1. **审批节点**: 定义审批的步骤和审批人
2. **审批规则**: 定义什么操作需要审批
3. **发布窗口**: 定义允许发布的时间段

## 审批流程
1. 用户提交需要审批的操作
2. 系统根据规则创建审批实例
3. 通知审批人进行审批
4. 审批通过后自动执行操作
5. 审批拒绝则取消操作

## 审批状态
- **待审批**: 等待审批人处理
- **已通过**: 审批通过
- **已拒绝**: 审批被拒绝
- **已超时**: 审批超时自动取消`,
		},

		// K8s管理
		{
			Title:    "K8s集群管理",
			Category: string(ai.CategoryK8s),
			Tags:     []string{"K8s", "集群", "管理"},
			Content: `# K8s集群管理

## 功能介绍
K8s管理模块提供Kubernetes集群的可视化管理能力。

## 主要功能
1. **集群概览**: 查看集群资源使用情况
2. **工作负载**: 管理Deployment、StatefulSet等
3. **Pod管理**: 查看Pod状态、日志、执行命令
4. **服务发现**: 管理Service、Ingress
5. **配置管理**: 管理ConfigMap、Secret
6. **存储管理**: 管理PV、PVC

## 常用操作
- 查看Pod日志: 选择Pod，点击"日志"
- 进入容器: 选择Pod，点击"终端"
- 扩缩容: 选择Deployment，调整副本数
- 重启Pod: 选择Pod，点击"删除"（会自动重建）`,
		},

		// 监控告警
		{
			Title:    "监控告警概述",
			Category: string(ai.CategoryMonitoring),
			Tags:     []string{"监控", "告警", "入门"},
			Content: `# 监控告警

## 功能介绍
监控告警模块提供应用和基础设施的监控能力。

## 主要功能
1. **指标监控**: CPU、内存、网络、磁盘等指标
2. **告警配置**: 配置告警规则和通知方式
3. **告警历史**: 查看历史告警记录
4. **健康检查**: 配置服务健康检查

## 告警级别
- **Critical**: 严重告警，需要立即处理
- **Warning**: 警告，需要关注
- **Info**: 信息，仅供参考

## 告警处理
1. 收到告警通知
2. 查看告警详情
3. 分析问题原因
4. 执行修复操作
5. 确认告警恢复`,
		},

		// CI/CD
		{
			Title:    "CI/CD流水线",
			Category: string(ai.CategoryCICD),
			Tags:     []string{"CI/CD", "流水线", "构建"},
			Content: `# CI/CD流水线

## 功能介绍
CI/CD模块提供持续集成和持续部署能力。

## 流水线阶段
1. **代码检出**: 从Git仓库拉取代码
2. **构建**: 编译代码、构建镜像
3. **测试**: 运行单元测试、集成测试
4. **扫描**: 代码扫描、镜像扫描
5. **部署**: 部署到目标环境

## 触发方式
- **手动触发**: 手动点击运行
- **代码提交**: 代码推送时自动触发
- **定时触发**: 按计划定时运行
- **Webhook**: 外部系统触发

## 流水线模板
平台提供多种预置模板：
- Java Maven构建
- Node.js构建
- Go构建
- Docker镜像构建`,
		},

		// 通用
		{
			Title:    "平台使用入门",
			Category: string(ai.CategoryGeneral),
			Tags:     []string{"入门", "指南"},
			Content: `# DevOps平台使用入门

## 平台简介
DevOps平台是一站式的应用运维管理平台，提供应用管理、部署发布、流量治理、监控告警等功能。

## 快速开始
1. 登录平台
2. 创建或选择应用
3. 配置应用环境
4. 执行部署操作
5. 查看监控和日志

## 主要模块
- **应用管理**: 管理应用和环境配置
- **部署中心**: 执行部署和回滚操作
- **流量治理**: 配置限流、熔断等策略
- **监控告警**: 查看监控指标和告警
- **K8s管理**: 管理Kubernetes资源
- **CI/CD**: 配置和运行流水线

## 获取帮助
- 点击页面右下角的AI助手图标
- 用自然语言描述你的问题
- AI助手会提供操作指导和建议`,
		},
	}
}
