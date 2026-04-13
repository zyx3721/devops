// Package traffic 定义流量治理相关的数据模型
//
// 本包包含与服务流量管理相关的所有数据模型，包括：
//   - 限流规则：QPS限制、突发流量控制
//   - 熔断规则：错误率熔断、慢调用熔断
//   - 路由规则：权重路由、条件路由
//   - 负载均衡：轮询、一致性哈希、最少连接
//   - 超时重试：请求超时、重试策略
//   - 流量镜像：影子流量、流量复制
//   - 故障注入：延迟注入、错误注入
//   - 金丝雀发布：灰度发布配置
//   - 蓝绿部署：蓝绿切换配置
//
// 使用示例:
//
//	import "devops/internal/models/traffic"
//
//	// 创建限流规则
//	rule := &traffic.TrafficRateLimitRule{
//	    Name:      "API限流",
//	    Threshold: 1000,
//	}
package traffic
