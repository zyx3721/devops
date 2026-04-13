<template>
  <div class="app-traffic-management">
    <!-- 应用信息头部 -->
    <a-page-header :title="app?.display_name || app?.name || '应用流量治理'" :sub-title="(app?.k8s_namespace || '') + '/' + (app?.k8s_deployment || '')" @back="$router.back()">
      <template #extra>
        <a-space>
          <a-button @click="refreshAll"><ReloadOutlined /> 刷新</a-button>
          <a-tag v-if="istioEnabled" color="green">Istio 已启用</a-tag>
          <a-tag v-else color="orange">Istio 未检测到</a-tag>
        </a-space>
      </template>
    </a-page-header>

    <a-alert v-if="!istioEnabled" type="warning" show-icon closable style="margin-bottom: 16px">
      <template #message>
        <strong>Istio 服务网格未安装</strong> - 流量治理规则将保存到数据库，但不会在 K8s 集群中生效
      </template>
      <template #description>
        <div style="margin-top: 8px">
          <p style="margin-bottom: 8px">受影响的功能：限流、熔断、流量路由、负载均衡、超时重试、流量镜像、故障注入</p>
          <p style="margin: 0">
            <a href="https://istio.io/latest/docs/setup/getting-started/" target="_blank" style="margin-right: 16px">
              查看 Istio 安装文档
            </a>
            <a @click="checkIstio" style="cursor: pointer">重新检测</a>
          </p>
        </div>
      </template>
    </a-alert>

    <a-tabs v-model:activeKey="activeTab">
      <!-- 限流配置 Tab -->
      <a-tab-pane key="ratelimit" tab="限流配置">
        <!-- 限流统计 -->
        <a-row :gutter="16" style="margin-bottom: 16px">
          <a-col :span="6">
            <a-card size="small">
              <a-statistic title="QPS 限流" :value="rateLimitStats.qps_rules" suffix="条">
                <template #prefix><ThunderboltOutlined style="color: #1890ff" /></template>
              </a-statistic>
            </a-card>
          </a-col>
          <a-col :span="6">
            <a-card size="small">
              <a-statistic title="并发限流" :value="rateLimitStats.concurrent_rules" suffix="条">
                <template #prefix><TeamOutlined style="color: #52c41a" /></template>
              </a-statistic>
            </a-card>
          </a-col>
          <a-col :span="6">
            <a-card size="small">
              <a-statistic title="令牌桶/漏桶" :value="rateLimitStats.bucket_rules" suffix="条">
                <template #prefix><FireOutlined style="color: #fa8c16" /></template>
              </a-statistic>
            </a-card>
          </a-col>
          <a-col :span="6">
            <a-card size="small">
              <a-statistic title="已启用" :value="rateLimitStats.enabled_rules" suffix="条">
                <template #prefix><CheckCircleOutlined style="color: #52c41a" /></template>
              </a-statistic>
            </a-card>
          </a-col>
        </a-row>

        <a-card title="接口限流规则" :bordered="false">
          <template #extra>
            <a-button type="primary" @click="showRateLimitModal()"><PlusOutlined /> 添加规则</a-button>
          </template>
          <a-table :columns="rateLimitColumns" :data-source="rateLimitRules" :loading="loadingRateLimit" row-key="id" size="small">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'name'">
                <div>
                  <span>{{ record.name || '限流规则' }}</span>
                  <div style="color: #999; font-size: 12px">{{ record.description || record.resource }}</div>
                </div>
              </template>
              <template v-if="column.key === 'resource'">
                <a-tag v-if="record.resource_type" color="blue" size="small">{{ record.resource_type }}</a-tag>
                <code style="margin-left: 4px">{{ record.resource }}</code>
                <a-tag v-if="record.method" size="small" style="margin-left: 4px">{{ record.method }}</a-tag>
              </template>
              <template v-if="column.key === 'strategy'">
                <a-tag :color="getRateLimitStrategyColor(record.strategy)">{{ getRateLimitStrategyText(record.strategy) }}</a-tag>
              </template>
              <template v-if="column.key === 'threshold'">
                <span v-if="record.strategy === 'qps' || !record.strategy">{{ record.threshold }} req/s</span>
                <span v-else-if="record.strategy === 'concurrent'">{{ record.threshold }} 并发</span>
                <span v-else-if="record.strategy === 'token_bucket'">{{ record.threshold }} tokens/s (容量: {{ record.burst }})</span>
                <span v-else-if="record.strategy === 'leaky_bucket'">{{ record.threshold }} req/s (队列: {{ record.queue_size }})</span>
                <span v-else>{{ record.threshold }}</span>
                <span v-if="record.burst && !record.strategy" style="color: #999"> (burst: {{ record.burst }})</span>
              </template>
              <template v-if="column.key === 'control_behavior'">
                <a-tag size="small">{{ getRateLimitBehaviorText(record.control_behavior) }}</a-tag>
              </template>
              <template v-if="column.key === 'enabled'">
                <a-switch v-model:checked="record.enabled" size="small" @change="toggleRateLimit(record)" />
              </template>
              <template v-if="column.key === 'action'">
                <a-space>
                  <a-button type="link" size="small" @click="showRateLimitModal(record)">编辑</a-button>
                  <a-popconfirm title="确定删除？" @confirm="deleteRateLimit(record.id)">
                    <a-button type="link" size="small" danger>删除</a-button>
                  </a-popconfirm>
                </a-space>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>

      <!-- 熔断配置 Tab -->
      <a-tab-pane key="circuitbreaker" tab="熔断配置">
        <!-- 熔断统计 -->
        <a-row :gutter="16" style="margin-bottom: 16px">
          <a-col :span="6">
            <a-card size="small">
              <a-statistic title="慢调用熔断" :value="circuitBreakerStats.slow_request" suffix="条">
                <template #prefix><ClockCircleOutlined style="color: #fa8c16" /></template>
              </a-statistic>
            </a-card>
          </a-col>
          <a-col :span="6">
            <a-card size="small">
              <a-statistic title="异常比例熔断" :value="circuitBreakerStats.error_ratio" suffix="条">
                <template #prefix><PercentageOutlined style="color: #f5222d" /></template>
              </a-statistic>
            </a-card>
          </a-col>
          <a-col :span="6">
            <a-card size="small">
              <a-statistic title="异常数熔断" :value="circuitBreakerStats.error_count" suffix="条">
                <template #prefix><ExclamationCircleOutlined style="color: #ff4d4f" /></template>
              </a-statistic>
            </a-card>
          </a-col>
          <a-col :span="6">
            <a-card size="small">
              <a-statistic title="已启用" :value="circuitBreakerStats.enabled" suffix="条">
                <template #prefix><CheckCircleOutlined style="color: #52c41a" /></template>
              </a-statistic>
            </a-card>
          </a-col>
        </a-row>

        <a-card title="熔断规则 (DestinationRule)" :bordered="false">
          <template #extra>
            <a-button type="primary" @click="showCircuitBreakerModal()"><PlusOutlined /> 配置熔断</a-button>
          </template>
          <a-descriptions v-if="circuitBreaker" :column="2" bordered size="small">
            <a-descriptions-item label="状态">
              <a-badge :status="circuitBreaker.enabled ? 'success' : 'default'" :text="circuitBreaker.enabled ? '已启用' : '未启用'" />
            </a-descriptions-item>
            <a-descriptions-item label="连续错误数">{{ circuitBreaker.consecutive_errors || 5 }}</a-descriptions-item>
            <a-descriptions-item label="检测间隔">{{ circuitBreaker.interval || '10s' }}</a-descriptions-item>
            <a-descriptions-item label="熔断时间">{{ circuitBreaker.base_ejection_time || '30s' }}</a-descriptions-item>
            <a-descriptions-item label="最大熔断比例">{{ circuitBreaker.max_ejection_percent || 100 }}%</a-descriptions-item>
            <a-descriptions-item label="最小健康实例">{{ circuitBreaker.min_health_percent || 0 }}%</a-descriptions-item>
          </a-descriptions>
          <a-empty v-else description="暂未配置熔断规则" />
        </a-card>

        <a-card title="连接池配置" :bordered="false" style="margin-top: 16px">
          <a-descriptions v-if="connectionPool" :column="2" bordered size="small">
            <a-descriptions-item label="HTTP1 最大连接数">{{ connectionPool.http1_max_pending || 1024 }}</a-descriptions-item>
            <a-descriptions-item label="HTTP2 最大请求数">{{ connectionPool.http2_max_requests || 1024 }}</a-descriptions-item>
            <a-descriptions-item label="每连接最大请求">{{ connectionPool.max_requests_per_connection || 0 }}</a-descriptions-item>
            <a-descriptions-item label="连接超时">{{ connectionPool.connect_timeout || '10s' }}</a-descriptions-item>
            <a-descriptions-item label="TCP 最大连接数">{{ connectionPool.tcp_max_connections || 1024 }}</a-descriptions-item>
            <a-descriptions-item label="空闲超时">{{ connectionPool.idle_timeout || '1h' }}</a-descriptions-item>
          </a-descriptions>
          <a-empty v-else description="暂未配置连接池" />
        </a-card>
      </a-tab-pane>

      <!-- 流量路由 Tab -->
      <a-tab-pane key="routing" tab="流量路由">
        <!-- 路由类型统计 -->
        <a-row :gutter="16" style="margin-bottom: 16px">
          <a-col :span="6">
            <a-card size="small">
              <a-statistic title="权重路由" :value="routingStats.weight" suffix="条">
                <template #prefix><PieChartOutlined style="color: #1890ff" /></template>
              </a-statistic>
              <div style="color: #999; font-size: 12px">按比例分配流量</div>
            </a-card>
          </a-col>
          <a-col :span="6">
            <a-card size="small">
              <a-statistic title="Header 路由" :value="routingStats.header" suffix="条">
                <template #prefix><TagOutlined style="color: #52c41a" /></template>
              </a-statistic>
              <div style="color: #999; font-size: 12px">按请求头匹配</div>
            </a-card>
          </a-col>
          <a-col :span="6">
            <a-card size="small">
              <a-statistic title="Cookie 路由" :value="routingStats.cookie" suffix="条">
                <template #prefix><CoffeeOutlined style="color: #fa8c16" /></template>
              </a-statistic>
              <div style="color: #999; font-size: 12px">按 Cookie 匹配</div>
            </a-card>
          </a-col>
          <a-col :span="6">
            <a-card size="small">
              <a-statistic title="参数路由" :value="routingStats.param" suffix="条">
                <template #prefix><FilterOutlined style="color: #722ed1" /></template>
              </a-statistic>
              <div style="color: #999; font-size: 12px">按请求参数匹配</div>
            </a-card>
          </a-col>
        </a-row>

        <a-card title="流量路由规则 (VirtualService)" :bordered="false">
          <template #extra>
            <a-button type="primary" @click="showRoutingModal()"><PlusOutlined /> 添加路由</a-button>
          </template>
          <a-table :columns="routingColumns" :data-source="routingRules" :loading="loadingRouting" row-key="id" size="small">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'name'">
                <div>
                  <span>{{ record.name }}</span>
                  <div style="color: #999; font-size: 12px">{{ record.description }}</div>
                </div>
              </template>
              <template v-if="column.key === 'type'">
                <a-tag :color="getRoutingTypeColor(record.route_type)">{{ getRoutingTypeText(record.route_type) }}</a-tag>
              </template>
              <template v-if="column.key === 'config'">
                <div v-if="record.route_type === 'weight'">
                  <div v-for="(dest, idx) in record.destinations" :key="idx" style="margin-bottom: 4px">
                    <a-tag>{{ dest.subset || dest.version }}</a-tag>
                    <a-progress :percent="dest.weight" size="small" style="width: 80px; display: inline-block" />
                  </div>
                </div>
                <div v-else>
                  <code>{{ record.match_key }} {{ record.match_operator }} {{ record.match_value }}</code>
                  <span style="margin-left: 8px">→</span>
                  <a-tag color="blue">{{ record.target_subset || record.target_version }}</a-tag>
                </div>
              </template>
              <template v-if="column.key === 'priority'">
                <a-tag>{{ record.priority }}</a-tag>
              </template>
              <template v-if="column.key === 'enabled'">
                <a-switch v-model:checked="record.enabled" size="small" @change="toggleRouting(record)" />
              </template>
              <template v-if="column.key === 'action'">
                <a-space>
                  <a-button type="link" size="small" @click="showRoutingModal(record)">编辑</a-button>
                  <a-popconfirm title="确定删除？" @confirm="deleteRouting(record.id)">
                    <a-button type="link" size="small" danger>删除</a-button>
                  </a-popconfirm>
                </a-space>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>

      <!-- 负载均衡 Tab -->
      <a-tab-pane key="loadbalance" tab="负载均衡">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-card title="负载均衡策略" :bordered="false" :loading="loadingLoadBalance">
              <template #extra>
                <a-button type="primary" size="small" @click="showLoadBalanceModal()"><SettingOutlined /> 配置</a-button>
              </template>
              <a-descriptions :column="1" bordered size="small" v-if="loadBalanceConfig">
                <a-descriptions-item label="负载均衡算法">
                  <a-tag :color="getLbPolicyColor(loadBalanceConfig.lb_policy)">{{ getLbPolicyText(loadBalanceConfig.lb_policy) }}</a-tag>
                </a-descriptions-item>
                <a-descriptions-item label="会话保持" v-if="loadBalanceConfig.lb_policy === 'consistent_hash'">
                  <a-tag>{{ getHashKeyText(loadBalanceConfig.hash_key) }}</a-tag>
                </a-descriptions-item>
                <a-descriptions-item label="哈希环大小" v-if="loadBalanceConfig.lb_policy === 'consistent_hash'">
                  {{ loadBalanceConfig.ring_size || 1024 }}
                </a-descriptions-item>
                <a-descriptions-item label="选择数量" v-if="loadBalanceConfig.lb_policy === 'least_request'">
                  {{ loadBalanceConfig.choice_count || 2 }}
                </a-descriptions-item>
                <a-descriptions-item label="预热时间">
                  {{ loadBalanceConfig.warmup_duration || '0s' }}
                </a-descriptions-item>
              </a-descriptions>
              <a-empty v-else description="使用默认负载均衡策略 (Round Robin)" />
            </a-card>
          </a-col>
          <a-col :span="12">
            <a-card title="健康检查" :bordered="false" :loading="loadingLoadBalance">
              <a-descriptions :column="1" bordered size="small" v-if="loadBalanceConfig?.health_check_enabled">
                <a-descriptions-item label="状态">
                  <a-badge status="success" text="已启用" />
                </a-descriptions-item>
                <a-descriptions-item label="检查路径">
                  {{ loadBalanceConfig.health_check_path || '/health' }}
                </a-descriptions-item>
                <a-descriptions-item label="检查间隔">
                  {{ loadBalanceConfig.health_check_interval || '10s' }}
                </a-descriptions-item>
                <a-descriptions-item label="超时时间">
                  {{ loadBalanceConfig.health_check_timeout || '5s' }}
                </a-descriptions-item>
                <a-descriptions-item label="健康阈值">
                  {{ loadBalanceConfig.healthy_threshold || 2 }} 次成功
                </a-descriptions-item>
                <a-descriptions-item label="不健康阈值">
                  {{ loadBalanceConfig.unhealthy_threshold || 3 }} 次失败
                </a-descriptions-item>
              </a-descriptions>
              <a-empty v-else description="未启用健康检查" />
            </a-card>
          </a-col>
        </a-row>

        <a-card title="连接池配置" :bordered="false" style="margin-top: 16px" :loading="loadingLoadBalance">
          <a-row :gutter="16">
            <a-col :span="12">
              <a-descriptions title="HTTP 连接池" :column="1" bordered size="small" v-if="loadBalanceConfig">
                <a-descriptions-item label="最大连接数">{{ loadBalanceConfig.http_max_connections || 1024 }}</a-descriptions-item>
                <a-descriptions-item label="每连接最大请求">{{ loadBalanceConfig.http_max_requests_per_conn || 0 }}</a-descriptions-item>
                <a-descriptions-item label="最大等待请求">{{ loadBalanceConfig.http_max_pending_requests || 1024 }}</a-descriptions-item>
                <a-descriptions-item label="最大重试次数">{{ loadBalanceConfig.http_max_retries || 3 }}</a-descriptions-item>
                <a-descriptions-item label="空闲超时">{{ loadBalanceConfig.http_idle_timeout || '1h' }}</a-descriptions-item>
              </a-descriptions>
              <a-empty v-else description="使用默认配置" />
            </a-col>
            <a-col :span="12">
              <a-descriptions title="TCP 连接池" :column="1" bordered size="small" v-if="loadBalanceConfig">
                <a-descriptions-item label="最大连接数">{{ loadBalanceConfig.tcp_max_connections || 1024 }}</a-descriptions-item>
                <a-descriptions-item label="连接超时">{{ loadBalanceConfig.tcp_connect_timeout || '10s' }}</a-descriptions-item>
                <a-descriptions-item label="TCP Keepalive">{{ loadBalanceConfig.tcp_keepalive_enabled ? '启用' : '禁用' }}</a-descriptions-item>
                <a-descriptions-item label="Keepalive 间隔" v-if="loadBalanceConfig.tcp_keepalive_enabled">{{ loadBalanceConfig.tcp_keepalive_interval || '60s' }}</a-descriptions-item>
              </a-descriptions>
              <a-empty v-else description="使用默认配置" />
            </a-col>
          </a-row>
        </a-card>
      </a-tab-pane>

      <!-- 超时重试 Tab -->
      <a-tab-pane key="timeout" tab="超时重试">
        <a-card title="超时配置 (VirtualService)" :bordered="false">
          <template #extra>
            <a-button type="primary" @click="showTimeoutModal()"><SettingOutlined /> 配置</a-button>
          </template>
          <a-descriptions v-if="timeoutConfig" :column="2" bordered size="small">
            <a-descriptions-item label="请求超时">{{ timeoutConfig.timeout || '30s' }}</a-descriptions-item>
            <a-descriptions-item label="重试次数">{{ timeoutConfig.retries || 3 }}</a-descriptions-item>
            <a-descriptions-item label="重试超时">{{ timeoutConfig.per_try_timeout || '10s' }}</a-descriptions-item>
            <a-descriptions-item label="重试条件">{{ (timeoutConfig.retry_on || ['5xx']).join(', ') }}</a-descriptions-item>
          </a-descriptions>
          <a-empty v-else description="使用默认超时配置" />
        </a-card>
      </a-tab-pane>

      <!-- 流量镜像 Tab -->
      <a-tab-pane key="mirror" tab="流量镜像">
        <a-card title="流量镜像配置" :bordered="false">
          <template #extra>
            <a-button type="primary" @click="showMirrorModal()"><CopyOutlined /> 配置镜像</a-button>
          </template>
          <a-table :columns="mirrorColumns" :data-source="mirrorRules" :loading="loadingMirror" row-key="id" size="small">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'target'">
                <span>{{ record.target_service }}</span>
                <a-tag v-if="record.target_subset" size="small" style="margin-left: 4px">{{ record.target_subset }}</a-tag>
              </template>
              <template v-if="column.key === 'percentage'">
                <a-progress :percent="record.percentage" size="small" style="width: 100px" />
              </template>
              <template v-if="column.key === 'enabled'">
                <a-switch v-model:checked="record.enabled" size="small" @change="toggleMirror(record)" />
              </template>
              <template v-if="column.key === 'action'">
                <a-popconfirm title="确定删除？" @confirm="deleteMirror(record.id)">
                  <a-button type="link" size="small" danger>删除</a-button>
                </a-popconfirm>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>

      <!-- 故障注入 Tab -->
      <a-tab-pane key="fault" tab="故障注入">
        <a-alert type="warning" show-icon style="margin-bottom: 16px">
          <template #message>故障注入仅用于测试环境，请勿在生产环境使用！</template>
        </a-alert>
        <a-card title="故障注入规则" :bordered="false">
          <template #extra>
            <a-button type="primary" @click="showFaultModal()"><BugOutlined /> 添加故障</a-button>
          </template>
          <a-table :columns="faultColumns" :data-source="faultRules" :loading="loadingFault" row-key="id" size="small">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'type'">
                <a-tag :color="record.type === 'delay' ? 'blue' : 'red'">{{ record.type === 'delay' ? '延迟' : '中断' }}</a-tag>
              </template>
              <template v-if="column.key === 'config'">
                <span v-if="record.type === 'delay'">延迟 {{ record.delay_duration }}</span>
                <span v-else>返回 {{ record.abort_code }} {{ record.abort_message }}</span>
              </template>
              <template v-if="column.key === 'percentage'">{{ record.percentage }}%</template>
              <template v-if="column.key === 'enabled'">
                <a-switch v-model:checked="record.enabled" size="small" @change="toggleFault(record)" />
              </template>
              <template v-if="column.key === 'action'">
                <a-popconfirm title="确定删除？" @confirm="deleteFault(record.id)">
                  <a-button type="link" size="small" danger>删除</a-button>
                </a-popconfirm>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>
    </a-tabs>

    <!-- 限流规则弹窗 -->
    <a-modal v-model:open="rateLimitModalVisible" :title="editingRateLimit ? '编辑限流规则' : '添加限流规则'" @ok="saveRateLimit" :confirm-loading="saving">
      <a-form :model="rateLimitForm" :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }">
        <a-form-item label="接口路径" required>
          <a-input v-model:value="rateLimitForm.resource" placeholder="/api/v1/users" />
        </a-form-item>
        <a-form-item label="请求方法">
          <a-select v-model:value="rateLimitForm.method" placeholder="全部">
            <a-select-option value="">全部</a-select-option>
            <a-select-option value="GET">GET</a-select-option>
            <a-select-option value="POST">POST</a-select-option>
            <a-select-option value="PUT">PUT</a-select-option>
            <a-select-option value="DELETE">DELETE</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="每秒请求数" required>
          <a-input-number v-model:value="rateLimitForm.threshold" :min="1" :max="100000" style="width: 150px" />
        </a-form-item>
        <a-form-item label="突发容量">
          <a-input-number v-model:value="rateLimitForm.burst" :min="0" :max="10000" style="width: 150px" />
        </a-form-item>
        <a-form-item label="启用">
          <a-switch v-model:checked="rateLimitForm.enabled" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 熔断配置弹窗 -->
    <a-modal v-model:open="circuitBreakerModalVisible" title="熔断配置" @ok="saveCircuitBreaker" :confirm-loading="saving" width="600px">
      <a-form :model="circuitBreakerForm" :label-col="{ span: 8 }" :wrapper-col="{ span: 14 }">
        <a-form-item label="启用熔断">
          <a-switch v-model:checked="circuitBreakerForm.enabled" />
        </a-form-item>
        <a-divider>异常检测</a-divider>
        <a-form-item label="连续错误数">
          <a-input-number v-model:value="circuitBreakerForm.consecutive_errors" :min="1" :max="100" />
          <span style="margin-left: 8px; color: #999">触发熔断的连续错误次数</span>
        </a-form-item>
        <a-form-item label="检测间隔">
          <a-input v-model:value="circuitBreakerForm.interval" placeholder="10s" style="width: 120px" />
        </a-form-item>
        <a-form-item label="熔断时间">
          <a-input v-model:value="circuitBreakerForm.base_ejection_time" placeholder="30s" style="width: 120px" />
        </a-form-item>
        <a-form-item label="最大熔断比例">
          <a-slider v-model:value="circuitBreakerForm.max_ejection_percent" :min="0" :max="100" />
        </a-form-item>
        <a-divider>连接池</a-divider>
        <a-form-item label="HTTP 最大请求数">
          <a-input-number v-model:value="circuitBreakerForm.http2_max_requests" :min="1" :max="100000" />
        </a-form-item>
        <a-form-item label="TCP 最大连接数">
          <a-input-number v-model:value="circuitBreakerForm.tcp_max_connections" :min="1" :max="100000" />
        </a-form-item>
        <a-form-item label="连接超时">
          <a-input v-model:value="circuitBreakerForm.connect_timeout" placeholder="10s" style="width: 120px" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 超时配置弹窗 -->
    <a-modal v-model:open="timeoutModalVisible" title="超时重试配置" @ok="saveTimeout" :confirm-loading="saving">
      <a-form :model="timeoutForm" :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }">
        <a-form-item label="请求超时">
          <a-input v-model:value="timeoutForm.timeout" placeholder="30s" style="width: 120px" />
        </a-form-item>
        <a-form-item label="重试次数">
          <a-input-number v-model:value="timeoutForm.retries" :min="0" :max="10" />
        </a-form-item>
        <a-form-item label="单次重试超时">
          <a-input v-model:value="timeoutForm.per_try_timeout" placeholder="10s" style="width: 120px" />
        </a-form-item>
        <a-form-item label="重试条件">
          <a-select v-model:value="timeoutForm.retry_on" mode="multiple" placeholder="选择重试条件">
            <a-select-option value="5xx">5xx 错误</a-select-option>
            <a-select-option value="gateway-error">网关错误</a-select-option>
            <a-select-option value="connect-failure">连接失败</a-select-option>
            <a-select-option value="retriable-4xx">可重试 4xx</a-select-option>
            <a-select-option value="reset">连接重置</a-select-option>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 流量镜像弹窗 -->
    <a-modal v-model:open="mirrorModalVisible" title="流量镜像配置" @ok="saveMirror" :confirm-loading="saving">
      <a-form :model="mirrorForm" :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }">
        <a-form-item label="目标服务" required>
          <a-select v-model:value="mirrorForm.target_service" placeholder="选择服务" show-search>
            <a-select-option v-for="svc in availableServices" :key="svc" :value="svc">{{ svc }}</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="目标子集">
          <a-input v-model:value="mirrorForm.target_subset" placeholder="如: canary" />
        </a-form-item>
        <a-form-item label="镜像比例">
          <a-slider v-model:value="mirrorForm.percentage" :min="1" :max="100" />
        </a-form-item>
        <a-form-item label="启用">
          <a-switch v-model:checked="mirrorForm.enabled" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 故障注入弹窗 -->
    <a-modal v-model:open="faultModalVisible" title="故障注入配置" @ok="saveFault" :confirm-loading="saving">
      <a-form :model="faultForm" :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }">
        <a-form-item label="故障类型" required>
          <a-radio-group v-model:value="faultForm.type">
            <a-radio value="delay">延迟注入</a-radio>
            <a-radio value="abort">请求中断</a-radio>
          </a-radio-group>
        </a-form-item>
        <a-form-item label="接口路径">
          <a-input v-model:value="faultForm.path" placeholder="/ 表示全部" />
        </a-form-item>
        <template v-if="faultForm.type === 'delay'">
          <a-form-item label="延迟时间" required>
            <a-input v-model:value="faultForm.delay_duration" placeholder="5s" style="width: 120px" />
          </a-form-item>
        </template>
        <template v-else>
          <a-form-item label="HTTP 状态码" required>
            <a-select v-model:value="faultForm.abort_code" style="width: 150px">
              <a-select-option :value="500">500 Internal Error</a-select-option>
              <a-select-option :value="502">502 Bad Gateway</a-select-option>
              <a-select-option :value="503">503 Unavailable</a-select-option>
              <a-select-option :value="504">504 Timeout</a-select-option>
            </a-select>
          </a-form-item>
        </template>
        <a-form-item label="影响比例">
          <a-slider v-model:value="faultForm.percentage" :min="1" :max="100" />
        </a-form-item>
        <a-form-item label="启用">
          <a-switch v-model:checked="faultForm.enabled" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 流量路由弹窗 -->
    <a-modal v-model:open="routingModalVisible" :title="editingRouting ? '编辑路由规则' : '添加路由规则'" @ok="saveRouting" :confirm-loading="saving" width="700px">
      <a-form :model="routingForm" :label-col="{ span: 5 }" :wrapper-col="{ span: 17 }">
        <a-form-item label="规则名称" required>
          <a-input v-model:value="routingForm.name" placeholder="如：灰度发布-v2" />
        </a-form-item>
        <a-form-item label="规则描述">
          <a-input v-model:value="routingForm.description" placeholder="规则用途说明" />
        </a-form-item>
        <a-form-item label="优先级">
          <a-input-number v-model:value="routingForm.priority" :min="1" :max="1000" style="width: 100%" />
          <div style="color: #999; font-size: 12px">数字越小优先级越高</div>
        </a-form-item>
        <a-divider>路由类型</a-divider>
        <a-form-item label="路由类型" required>
          <a-radio-group v-model:value="routingForm.route_type">
            <a-radio-button value="weight">权重路由</a-radio-button>
            <a-radio-button value="header">Header 匹配</a-radio-button>
            <a-radio-button value="cookie">Cookie 匹配</a-radio-button>
            <a-radio-button value="param">参数匹配</a-radio-button>
          </a-radio-group>
        </a-form-item>
        <template v-if="routingForm.route_type === 'weight'">
          <a-form-item label="流量分配">
            <div v-for="(dest, idx) in routingForm.destinations" :key="idx" style="margin-bottom: 8px; display: flex; align-items: center">
              <a-input v-model:value="dest.subset" placeholder="版本/子集" style="width: 120px" />
              <a-slider v-model:value="dest.weight" :min="0" :max="100" style="flex: 1; margin: 0 12px" />
              <span style="width: 40px">{{ dest.weight }}%</span>
            </div>
          </a-form-item>
        </template>
        <template v-else>
          <a-form-item label="匹配键" required>
            <a-input v-model:value="routingForm.match_key" placeholder="如: X-User-Type" />
          </a-form-item>
          <a-form-item label="匹配方式" required>
            <a-select v-model:value="routingForm.match_operator">
              <a-select-option value="exact">精确匹配</a-select-option>
              <a-select-option value="prefix">前缀匹配</a-select-option>
              <a-select-option value="regex">正则匹配</a-select-option>
            </a-select>
          </a-form-item>
          <a-form-item label="匹配值" required>
            <a-input v-model:value="routingForm.match_value" placeholder="匹配的值" />
          </a-form-item>
          <a-form-item label="目标版本" required>
            <a-input v-model:value="routingForm.target_subset" placeholder="如: v2, canary" />
          </a-form-item>
        </template>
        <a-form-item label="启用">
          <a-switch v-model:checked="routingForm.enabled" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 负载均衡弹窗 -->
    <a-modal v-model:open="loadBalanceModalVisible" title="负载均衡配置" @ok="saveLoadBalance" :confirm-loading="saving" width="700px">
      <a-form :model="loadBalanceForm" :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }">
        <a-divider>负载均衡策略</a-divider>
        <a-form-item label="负载均衡算法">
          <a-select v-model:value="loadBalanceForm.lb_policy">
            <a-select-option value="round_robin">轮询 (Round Robin)</a-select-option>
            <a-select-option value="random">随机 (Random)</a-select-option>
            <a-select-option value="least_request">最少请求 (Least Request)</a-select-option>
            <a-select-option value="consistent_hash">一致性哈希 (Consistent Hash)</a-select-option>
          </a-select>
        </a-form-item>
        <template v-if="loadBalanceForm.lb_policy === 'consistent_hash'">
          <a-form-item label="哈希键">
            <a-select v-model:value="loadBalanceForm.hash_key">
              <a-select-option value="header">请求头</a-select-option>
              <a-select-option value="cookie">Cookie</a-select-option>
              <a-select-option value="source_ip">源 IP</a-select-option>
            </a-select>
          </a-form-item>
          <a-form-item label="哈希键名" v-if="loadBalanceForm.hash_key !== 'source_ip'">
            <a-input v-model:value="loadBalanceForm.hash_key_name" placeholder="如: X-User-Id" />
          </a-form-item>
        </template>
        <template v-if="loadBalanceForm.lb_policy === 'least_request'">
          <a-form-item label="选择数量">
            <a-input-number v-model:value="loadBalanceForm.choice_count" :min="2" :max="10" style="width: 100%" />
          </a-form-item>
        </template>
        <a-divider>健康检查</a-divider>
        <a-form-item label="启用健康检查">
          <a-switch v-model:checked="loadBalanceForm.health_check_enabled" />
        </a-form-item>
        <template v-if="loadBalanceForm.health_check_enabled">
          <a-form-item label="检查路径">
            <a-input v-model:value="loadBalanceForm.health_check_path" placeholder="/health" />
          </a-form-item>
          <a-form-item label="检查间隔">
            <a-input v-model:value="loadBalanceForm.health_check_interval" placeholder="10s" />
          </a-form-item>
        </template>
        <a-divider>连接池</a-divider>
        <a-form-item label="HTTP 最大连接">
          <a-input-number v-model:value="loadBalanceForm.http_max_connections" :min="1" style="width: 100%" />
        </a-form-item>
        <a-form-item label="TCP 最大连接">
          <a-input-number v-model:value="loadBalanceForm.tcp_max_connections" :min="1" style="width: 100%" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { message } from 'ant-design-vue'
import { 
  ReloadOutlined, PlusOutlined, SettingOutlined, CopyOutlined, BugOutlined, 
  PieChartOutlined, TagOutlined, CoffeeOutlined, FilterOutlined,
  ThunderboltOutlined, TeamOutlined, FireOutlined, CheckCircleOutlined,
  ClockCircleOutlined, PercentageOutlined, ExclamationCircleOutlined, StopOutlined
} from '@ant-design/icons-vue'
import request from '@/utils/request'

const route = useRoute()
const appId = Number(route.params.id)

const activeTab = ref('ratelimit')
const loading = ref(false)
const loadingRateLimit = ref(false)
const loadingRouting = ref(false)
const loadingLoadBalance = ref(false)
const loadingMirror = ref(false)
const loadingFault = ref(false)
const saving = ref(false)
const istioEnabled = ref(false)

const app = ref<any>(null)
const rateLimitRules = ref<any[]>([])
const routingRules = ref<any[]>([])
const loadBalanceConfig = ref<any>(null)
const circuitBreaker = ref<any>(null)
const connectionPool = ref<any>(null)
const timeoutConfig = ref<any>(null)
const mirrorRules = ref<any[]>([])
const faultRules = ref<any[]>([])
const availableServices = ref<string[]>([])

// 弹窗状态
const rateLimitModalVisible = ref(false)
const routingModalVisible = ref(false)
const loadBalanceModalVisible = ref(false)
const circuitBreakerModalVisible = ref(false)
const timeoutModalVisible = ref(false)
const mirrorModalVisible = ref(false)
const faultModalVisible = ref(false)
const editingRateLimit = ref<any>(null)
const editingRouting = ref<any>(null)

// 表单
const rateLimitForm = reactive({ resource: '', method: '', threshold: 100, burst: 10, enabled: true })
const routingForm = reactive({
  name: '', description: '', priority: 100, route_type: 'weight',
  destinations: [{ subset: 'v1', weight: 90 }, { subset: 'v2', weight: 10 }],
  match_key: '', match_operator: 'exact', match_value: '', target_subset: '', enabled: true
})
const loadBalanceForm = reactive({
  lb_policy: 'round_robin', hash_key: 'header', hash_key_name: '', ring_size: 1024,
  choice_count: 2, warmup_duration: '60s', health_check_enabled: false,
  health_check_path: '/health', health_check_interval: '10s', health_check_timeout: '5s',
  healthy_threshold: 2, unhealthy_threshold: 3, http_max_connections: 1024,
  http_max_requests_per_conn: 0, tcp_max_connections: 1024, tcp_connect_timeout: '10s'
})
const circuitBreakerForm = reactive({
  enabled: false, consecutive_errors: 5, interval: '10s', base_ejection_time: '30s',
  max_ejection_percent: 100, http2_max_requests: 1024, tcp_max_connections: 1024, connect_timeout: '10s'
})
const timeoutForm = reactive({ timeout: '30s', retries: 3, per_try_timeout: '10s', retry_on: ['5xx'] })
const mirrorForm = reactive({ target_service: '', target_subset: '', percentage: 100, enabled: true })
const faultForm = reactive({ type: 'delay', path: '/', delay_duration: '5s', abort_code: 500, percentage: 10, enabled: false })

// 表格列定义
const rateLimitColumns = [
  { title: '规则名称', key: 'name', width: 200 },
  { title: '资源', key: 'resource' },
  { title: '策略', key: 'strategy', width: 120 },
  { title: '阈值', key: 'threshold', width: 200 },
  { title: '超限行为', key: 'control_behavior', width: 100 },
  { title: '启用', key: 'enabled', width: 80 },
  { title: '操作', key: 'action', width: 120 }
]

const routingColumns = [
  { title: '规则名称', key: 'name', width: 200 },
  { title: '类型', key: 'type', width: 120 },
  { title: '配置', key: 'config' },
  { title: '优先级', key: 'priority', width: 80 },
  { title: '启用', key: 'enabled', width: 80 },
  { title: '操作', key: 'action', width: 120 }
]

const mirrorColumns = [
  { title: '目标服务', key: 'target' },
  { title: '镜像比例', key: 'percentage', width: 150 },
  { title: '启用', key: 'enabled', width: 80 },
  { title: '操作', key: 'action', width: 80 }
]

const faultColumns = [
  { title: '类型', key: 'type', width: 80 },
  { title: '接口', dataIndex: 'path', key: 'path' },
  { title: '配置', key: 'config' },
  { title: '比例', key: 'percentage', width: 80 },
  { title: '启用', key: 'enabled', width: 80 },
  { title: '操作', key: 'action', width: 80 }
]

// 限流统计
const rateLimitStats = computed(() => ({
  qps_rules: rateLimitRules.value.filter(r => !r.strategy || r.strategy === 'qps').length,
  concurrent_rules: rateLimitRules.value.filter(r => r.strategy === 'concurrent').length,
  bucket_rules: rateLimitRules.value.filter(r => r.strategy === 'token_bucket' || r.strategy === 'leaky_bucket').length,
  enabled_rules: rateLimitRules.value.filter(r => r.enabled).length
}))

// 限流辅助函数
const rateLimitStrategyMap: Record<string, { text: string; color: string }> = {
  qps: { text: 'QPS限流', color: 'blue' },
  concurrent: { text: '并发限流', color: 'green' },
  token_bucket: { text: '令牌桶', color: 'purple' },
  leaky_bucket: { text: '漏桶', color: 'orange' }
}
const getRateLimitStrategyText = (s: string) => rateLimitStrategyMap[s]?.text || 'QPS限流'
const getRateLimitStrategyColor = (s: string) => rateLimitStrategyMap[s]?.color || 'blue'

const rateLimitBehaviorMap: Record<string, string> = {
  reject: '直接拒绝',
  warm_up: '预热',
  queue: '排队',
  warm_up_queue: '预热+排队'
}
const getRateLimitBehaviorText = (b: string) => rateLimitBehaviorMap[b] || '直接拒绝'

// 熔断统计（注意：熔断配置是单个对象，这里统计的是配置的策略类型）
const circuitBreakerStats = computed(() => {
  // 由于熔断是单个配置，这里显示配置的状态
  const hasConfig = !!circuitBreaker.value
  return {
    slow_request: hasConfig && circuitBreaker.value?.strategy === 'slow_request' ? 1 : 0,
    error_ratio: hasConfig && circuitBreaker.value?.strategy === 'error_ratio' ? 1 : 0,
    error_count: hasConfig && circuitBreaker.value?.strategy === 'error_count' ? 1 : 0,
    enabled: hasConfig && circuitBreaker.value?.enabled ? 1 : 0
  }
})

// 路由统计
const routingStats = computed(() => ({
  weight: routingRules.value.filter(r => r.route_type === 'weight').length,
  header: routingRules.value.filter(r => r.route_type === 'header').length,
  cookie: routingRules.value.filter(r => r.route_type === 'cookie').length,
  param: routingRules.value.filter(r => r.route_type === 'param').length
}))

// 路由类型辅助函数
const routingTypeMap: Record<string, { text: string; color: string }> = {
  weight: { text: '权重路由', color: 'blue' },
  header: { text: 'Header', color: 'green' },
  cookie: { text: 'Cookie', color: 'orange' },
  param: { text: '参数', color: 'purple' }
}
const getRoutingTypeText = (t: string) => routingTypeMap[t]?.text || t
const getRoutingTypeColor = (t: string) => routingTypeMap[t]?.color || 'default'

// 获取应用信息
const fetchApp = async () => {
  try {
    const res = await request.get(`/applications/${appId}`)
    app.value = res.data?.app || res.data?.data || res.data || res
  } catch (e) { console.error('获取应用信息失败', e) }
}

// 检查 Istio 状态
const checkIstio = async () => {
  if (!app.value?.k8s_cluster_id) return
  try {
    const res = await request.get(`/k8s/clusters/${app.value.k8s_cluster_id}/istio/status`)
    istioEnabled.value = res.data?.enabled || false
  } catch (e) { 
    istioEnabled.value = false
    // 静默处理，不打印错误
  }
}

// 获取限流规则
const fetchRateLimits = async () => {
  loadingRateLimit.value = true
  try {
    const res = await request.get(`/applications/${appId}/traffic/ratelimits`)
    rateLimitRules.value = res.data?.items || []
  } catch (e) { console.error('获取限流规则失败', e) }
  finally { loadingRateLimit.value = false }
}

// 获取流量路由规则
const fetchRoutingRules = async () => {
  loadingRouting.value = true
  try {
    const res = await request.get(`/applications/${appId}/traffic/routes`)
    routingRules.value = res.data?.items || []
  } catch (e) { console.error('获取流量路由规则失败', e) }
  finally { loadingRouting.value = false }
}

// 获取负载均衡配置
const fetchLoadBalance = async () => {
  loadingLoadBalance.value = true
  try {
    const res = await request.get(`/applications/${appId}/traffic/loadbalance`)
    loadBalanceConfig.value = res.data || null
  } catch (e) { console.error('获取负载均衡配置失败', e) }
  finally { loadingLoadBalance.value = false }
}

// 获取熔断配置
const fetchCircuitBreaker = async () => {
  try {
    const res = await request.get(`/applications/${appId}/traffic/circuitbreaker`)
    circuitBreaker.value = res.data?.circuit_breaker || null
    connectionPool.value = res.data?.connection_pool || null
  } catch (e) { console.error('获取熔断配置失败', e) }
}

// 获取超时配置
const fetchTimeout = async () => {
  try {
    const res = await request.get(`/applications/${appId}/traffic/timeout`)
    timeoutConfig.value = res.data || null
  } catch (e) { console.error('获取超时配置失败', e) }
}

// 获取流量镜像
const fetchMirrors = async () => {
  loadingMirror.value = true
  try {
    const res = await request.get(`/applications/${appId}/traffic/mirrors`)
    mirrorRules.value = res.data?.items || []
  } catch (e) { console.error('获取流量镜像失败', e) }
  finally { loadingMirror.value = false }
}

// 获取故障注入
const fetchFaults = async () => {
  loadingFault.value = true
  try {
    const res = await request.get(`/applications/${appId}/traffic/faults`)
    faultRules.value = res.data?.items || []
  } catch (e) { console.error('获取故障注入失败', e) }
  finally { loadingFault.value = false }
}

// 获取可用服务列表
const fetchServices = async () => {
  if (!app.value?.k8s_cluster_id || !app.value?.k8s_namespace) return
  try {
    const res = await request.get(`/k8s/clusters/${app.value.k8s_cluster_id}/namespaces/${app.value.k8s_namespace}/services`)
    availableServices.value = (res.data?.items || []).map((s: any) => s.metadata?.name || s.name)
  } catch (e) { 
    availableServices.value = []
    // 静默处理，不打印错误
  }
}

const refreshAll = () => {
  fetchApp().then(() => {
    checkIstio()
    fetchRateLimits()
    fetchRoutingRules()
    fetchLoadBalance()
    fetchCircuitBreaker()
    fetchTimeout()
    fetchMirrors()
    fetchFaults()
    fetchServices()
  })
}

// 限流规则操作
const showRateLimitModal = (record?: any) => {
  editingRateLimit.value = record || null
  if (record) {
    Object.assign(rateLimitForm, record)
  } else {
    Object.assign(rateLimitForm, { resource: '', method: '', threshold: 100, burst: 10, enabled: true })
  }
  rateLimitModalVisible.value = true
}

const saveRateLimit = async () => {
  if (!rateLimitForm.resource) { message.warning('请输入接口路径'); return }
  saving.value = true
  try {
    if (editingRateLimit.value) {
      await request.put(`/applications/${appId}/traffic/ratelimits/${editingRateLimit.value.id}`, rateLimitForm)
      message.success('保存成功')
    } else {
      const res = await request.post(`/applications/${appId}/traffic/ratelimits`, rateLimitForm)
      // 检查 K8s 同步状态
      if (res.k8s_synced === false && res.k8s_error) {
        message.warning(`规则已保存到数据库，但同步到 K8s 失败: ${res.k8s_error}`, 5)
      } else if (res.k8s_synced === false) {
        message.warning('规则已保存到数据库，但未同步到 K8s（可能 Istio 未安装）', 5)
      } else {
        message.success('保存成功并已同步到 K8s')
      }
    }
    rateLimitModalVisible.value = false
    fetchRateLimits()
  } catch (e) { message.error('保存失败') }
  finally { saving.value = false }
}

const toggleRateLimit = async (record: any) => {
  try {
    await request.put(`/applications/${appId}/traffic/ratelimits/${record.id}`, { enabled: record.enabled })
    message.success(record.enabled ? '已启用' : '已禁用')
  } catch (e) { record.enabled = !record.enabled; message.error('操作失败') }
}

const deleteRateLimit = async (id: number) => {
  try {
    await request.delete(`/applications/${appId}/traffic/ratelimits/${id}`)
    message.success('删除成功')
    fetchRateLimits()
  } catch (e) { message.error('删除失败') }
}

// 流量路由操作
const showRoutingModal = (record?: any) => {
  editingRouting.value = record || null
  if (record) {
    Object.assign(routingForm, record)
    if (!routingForm.destinations) routingForm.destinations = [{ subset: 'v1', weight: 100 }]
  } else {
    Object.assign(routingForm, {
      name: '', description: '', priority: 100, route_type: 'weight',
      destinations: [{ subset: 'v1', weight: 90 }, { subset: 'v2', weight: 10 }],
      match_key: '', match_operator: 'exact', match_value: '', target_subset: '', enabled: true
    })
  }
  routingModalVisible.value = true
}

const saveRouting = async () => {
  if (!routingForm.name) { message.warning('请填写规则名称'); return }
  saving.value = true
  try {
    if (editingRouting.value) {
      await request.put(`/applications/${appId}/traffic/routes/${editingRouting.value.id}`, routingForm)
      message.success('保存成功')
    } else {
      const res = await request.post(`/applications/${appId}/traffic/routes`, routingForm)
      // 检查 K8s 同步状态
      if (res.k8s_synced === false && res.k8s_error) {
        message.warning(`规则已保存到数据库，但同步到 K8s 失败: ${res.k8s_error}`, 5)
      } else if (res.k8s_synced === false) {
        message.warning('规则已保存到数据库，但未同步到 K8s（可能 Istio 未安装）', 5)
      } else {
        message.success('保存成功并已同步到 K8s')
      }
    }
    routingModalVisible.value = false
    fetchRoutingRules()
  } catch (e) { message.error('保存失败') }
  finally { saving.value = false }
}

const toggleRouting = async (record: any) => {
  try {
    await request.put(`/applications/${appId}/traffic/routes/${record.id}`, { enabled: record.enabled })
  } catch (e) { record.enabled = !record.enabled; message.error('操作失败') }
}

const deleteRouting = async (id: number) => {
  try {
    await request.delete(`/applications/${appId}/traffic/routes/${id}`)
    message.success('删除成功')
    fetchRoutingRules()
  } catch (e) { message.error('删除失败') }
}

// 负载均衡操作
const showLoadBalanceModal = () => {
  if (loadBalanceConfig.value) {
    Object.assign(loadBalanceForm, loadBalanceConfig.value)
  }
  loadBalanceModalVisible.value = true
}

const saveLoadBalance = async () => {
  saving.value = true
  try {
    await request.put(`/applications/${appId}/traffic/loadbalance`, loadBalanceForm)
    message.success('保存成功')
    loadBalanceModalVisible.value = false
    fetchLoadBalance()
  } catch (e) { message.error('保存失败') }
  finally { saving.value = false }
}

const getLbPolicyText = (policy: string) => {
  const map: Record<string, string> = {
    round_robin: '轮询', random: '随机', least_request: '最少请求',
    consistent_hash: '一致性哈希', passthrough: '直通'
  }
  return map[policy] || policy
}

const getLbPolicyColor = (policy: string) => {
  const map: Record<string, string> = {
    round_robin: 'blue', random: 'green', least_request: 'purple',
    consistent_hash: 'orange', passthrough: 'default'
  }
  return map[policy] || 'default'
}

const getHashKeyText = (key: string) => {
  const map: Record<string, string> = {
    header: '请求头', cookie: 'Cookie', source_ip: '源 IP', query_param: '查询参数'
  }
  return map[key] || key
}

// 熔断配置操作
const showCircuitBreakerModal = () => {
  if (circuitBreaker.value) {
    Object.assign(circuitBreakerForm, circuitBreaker.value, connectionPool.value || {})
  }
  circuitBreakerModalVisible.value = true
}

const saveCircuitBreaker = async () => {
  saving.value = true
  try {
    const res = await request.put(`/applications/${appId}/traffic/circuitbreaker`, circuitBreakerForm)
    // 检查 K8s 同步状态
    if (res.k8s_synced === false && res.k8s_error) {
      message.warning(`配置已保存到数据库，但同步到 K8s 失败: ${res.k8s_error}`, 5)
    } else if (res.k8s_synced === false) {
      message.warning('配置已保存到数据库，但未同步到 K8s（可能 Istio 未安装）', 5)
    } else {
      message.success('保存成功并已同步到 K8s')
    }
    circuitBreakerModalVisible.value = false
    fetchCircuitBreaker()
  } catch (e) { message.error('保存失败') }
  finally { saving.value = false }
}

// 超时配置操作
const showTimeoutModal = () => {
  if (timeoutConfig.value) {
    Object.assign(timeoutForm, timeoutConfig.value)
  }
  timeoutModalVisible.value = true
}

const saveTimeout = async () => {
  saving.value = true
  try {
    await request.put(`/applications/${appId}/traffic/timeout`, timeoutForm)
    message.success('保存成功')
    timeoutModalVisible.value = false
    fetchTimeout()
  } catch (e) { message.error('保存失败') }
  finally { saving.value = false }
}

// 流量镜像操作
const showMirrorModal = () => {
  Object.assign(mirrorForm, { target_service: '', target_subset: '', percentage: 100, enabled: true })
  mirrorModalVisible.value = true
}

const saveMirror = async () => {
  if (!mirrorForm.target_service) { message.warning('请选择目标服务'); return }
  saving.value = true
  try {
    await request.post(`/applications/${appId}/traffic/mirrors`, mirrorForm)
    message.success('保存成功')
    mirrorModalVisible.value = false
    fetchMirrors()
  } catch (e) { message.error('保存失败') }
  finally { saving.value = false }
}

const toggleMirror = async (record: any) => {
  try {
    await request.put(`/applications/${appId}/traffic/mirrors/${record.id}`, { enabled: record.enabled })
  } catch (e) { record.enabled = !record.enabled; message.error('操作失败') }
}

const deleteMirror = async (id: number) => {
  try {
    await request.delete(`/applications/${appId}/traffic/mirrors/${id}`)
    message.success('删除成功')
    fetchMirrors()
  } catch (e) { message.error('删除失败') }
}

// 故障注入操作
const showFaultModal = () => {
  Object.assign(faultForm, { type: 'delay', path: '/', delay_duration: '5s', abort_code: 500, percentage: 10, enabled: false })
  faultModalVisible.value = true
}

const saveFault = async () => {
  saving.value = true
  try {
    await request.post(`/applications/${appId}/traffic/faults`, faultForm)
    message.success('保存成功')
    faultModalVisible.value = false
    fetchFaults()
  } catch (e) { message.error('保存失败') }
  finally { saving.value = false }
}

const toggleFault = async (record: any) => {
  try {
    await request.put(`/applications/${appId}/traffic/faults/${record.id}`, { enabled: record.enabled })
  } catch (e) { record.enabled = !record.enabled; message.error('操作失败') }
}

const deleteFault = async (id: number) => {
  try {
    await request.delete(`/applications/${appId}/traffic/faults/${id}`)
    message.success('删除成功')
    fetchFaults()
  } catch (e) { message.error('删除失败') }
}

onMounted(() => { refreshAll() })
</script>

<style scoped>
.app-traffic-management { padding: 0; }
.app-traffic-management :deep(.ant-page-header) { padding: 16px 0; }
</style>
