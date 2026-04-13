<template>
  <a-drawer :open="visible" :title="title" width="900" @close="$emit('close')" :body-style="{ padding: '16px' }">
    <template #extra>
      <a-space>
        <a-dropdown v-if="['deployment','statefulset','daemonset'].includes(resourceType)">
          <a-button>创建关联资源 <DownOutlined /></a-button>
          <template #overlay>
            <a-menu @click="handleCreateRelated">
              <a-menu-item key="service">创建 Service</a-menu-item>
              <a-menu-item key="ingress">创建 Ingress</a-menu-item>
            </a-menu>
          </template>
        </a-dropdown>
        <a-button type="primary" @click="handleEdit">编辑</a-button>
      </a-space>
    </template>
    <a-spin :spinning="loading">
      <!-- 基本信息 -->
      <a-card title="基本信息" size="small" class="detail-card">
        <a-descriptions :column="2" size="small" bordered>
          <a-descriptions-item label="名称">{{ detail?.name }}</a-descriptions-item>
          <a-descriptions-item label="命名空间" v-if="detail?.namespace">{{ detail?.namespace }}</a-descriptions-item>
          <a-descriptions-item label="创建时间">{{ detail?.created_at }}</a-descriptions-item>
          <a-descriptions-item label="UID">{{ detail?.uid }}</a-descriptions-item>
          <template v-if="resourceType === 'deployment'">
            <a-descriptions-item label="副本数">{{ detail?.replicas }}</a-descriptions-item>
            <a-descriptions-item label="就绪">{{ detail?.ready }}/{{ detail?.replicas }}</a-descriptions-item>
            <a-descriptions-item label="更新策略">{{ detail?.strategy }}</a-descriptions-item>
          </template>
          <template v-if="resourceType === 'service'">
            <a-descriptions-item label="类型">{{ detail?.type }}</a-descriptions-item>
            <a-descriptions-item label="ClusterIP">{{ detail?.cluster_ip }}</a-descriptions-item>
          </template>
          <template v-if="resourceType === 'pod'">
            <a-descriptions-item label="状态"><a-tag :color="getPodStatusColor(detail?.status)">{{ detail?.status }}</a-tag></a-descriptions-item>
            <a-descriptions-item label="Pod IP">{{ detail?.ip }}</a-descriptions-item>
            <a-descriptions-item label="节点"><a class="link" @click="goToNode(detail?.node)">{{ detail?.node }}</a></a-descriptions-item>
            <a-descriptions-item label="重启次数">{{ detail?.restarts }}</a-descriptions-item>
          </template>
          <template v-if="resourceType === 'node'">
            <a-descriptions-item label="状态"><a-tag :color="detail?.status === 'Ready' ? 'green' : 'red'">{{ detail?.status }}</a-tag></a-descriptions-item>
            <a-descriptions-item label="可调度">{{ detail?.schedulable ? '是' : '否' }}</a-descriptions-item>
            <a-descriptions-item label="内部IP">{{ detail?.internal_ip }}</a-descriptions-item>
            <a-descriptions-item label="CPU">{{ detail?.cpu_allocatable }} / {{ detail?.cpu_capacity }}</a-descriptions-item>
            <a-descriptions-item label="内存">{{ detail?.memory_allocatable }} / {{ detail?.memory_capacity }}</a-descriptions-item>
            <a-descriptions-item label="Kubelet版本">{{ detail?.kubelet_version }}</a-descriptions-item>
          </template>
        </a-descriptions>
      </a-card>

      <!-- 标签 -->
      <a-card title="标签" size="small" class="detail-card" v-if="detail?.labels && Object.keys(detail.labels).length">
        <div class="tags-container">
          <a-tag v-for="(v, k) in detail.labels" :key="k" color="blue">{{ k }}={{ v }}</a-tag>
        </div>
      </a-card>

      <!-- 注解 -->
      <a-card title="注解" size="small" class="detail-card" v-if="detail?.annotations && Object.keys(detail.annotations).length">
        <a-collapse :bordered="false" ghost>
          <a-collapse-panel header="点击展开">
            <div v-for="(v, k) in detail.annotations" :key="k" class="annotation-item"><span class="anno-key">{{ k }}:</span> {{ v }}</div>
          </a-collapse-panel>
        </a-collapse>
      </a-card>

      <!-- 关联资源 -->
      <a-card title="关联资源" size="small" class="detail-card" v-if="hasRelatedResources">
        <template v-if="['deployment','statefulset','daemonset'].includes(resourceType)">
          <!-- Pods -->
          <div class="related-section" v-if="relatedPods.length">
            <div class="section-title">Pods ({{ relatedPods.length }})</div>
            <a-table :columns="podMiniColumns" :data-source="relatedPods" size="small" :pagination="false" row-key="name">
              <template #bodyCell="{ column, record }">
                <template v-if="column.key === 'name'"><a class="link" @click="goToResource('pod', record.namespace, record.name)">{{ record.name }}</a></template>
                <template v-if="column.key === 'status'"><a-tag :color="getPodStatusColor(record.status)">{{ record.status }}</a-tag></template>
                <template v-if="column.key === 'action'">
                  <a-space>
                    <a @click="showPodLogs(record)">日志</a>
                    <a @click="showPodTerminal(record)">终端</a>
                    <a-popconfirm title="确定删除此 Pod？" @confirm="deletePod(record)">
                      <a style="color: #ff4d4f">删除</a>
                    </a-popconfirm>
                  </a-space>
                </template>
              </template>
            </a-table>
          </div>
          
          <!-- Services -->
          <div class="related-section" v-if="relatedServices.length" style="margin-top: 16px">
            <div class="section-title">Services ({{ relatedServices.length }})</div>
            <a-table :data-source="relatedServices" size="small" :pagination="false" row-key="name">
              <a-table-column title="名称" key="name">
                <template #default="{ record }">
                  <a class="link" @click="goToResource('service', record.namespace, record.name)">{{ record.name }}</a>
                </template>
              </a-table-column>
              <a-table-column title="类型" dataIndex="type" key="type" width="120" />
              <a-table-column title="Cluster IP" dataIndex="cluster_ip" key="cluster_ip" width="150" />
              <a-table-column title="端口" key="ports" width="200">
                <template #default="{ record }">
                  <a-tag v-for="(port, idx) in record.ports" :key="idx" style="margin: 2px">
                    {{ port.port }}{{ port.node_port ? ':' + port.node_port : '' }}/{{ port.protocol }}
                  </a-tag>
                </template>
              </a-table-column>
            </a-table>
          </div>
          
          <!-- Ingresses -->
          <div class="related-section" v-if="relatedIngresses.length" style="margin-top: 16px">
            <div class="section-title">Ingresses ({{ relatedIngresses.length }})</div>
            <a-table :data-source="relatedIngresses" size="small" :pagination="false" row-key="name">
              <a-table-column title="名称" key="name">
                <template #default="{ record }">
                  <a class="link" @click="goToResource('ingress', record.namespace, record.name)">{{ record.name }}</a>
                </template>
              </a-table-column>
              <a-table-column title="Class" dataIndex="class_name" key="class_name" width="120" />
              <a-table-column title="规则" key="rules">
                <template #default="{ record }">
                  <div v-for="(rule, idx) in record.rules" :key="idx">
                    <a-tag color="blue">{{ rule.host || '*' }}{{ rule.paths?.[0]?.path || '/' }}</a-tag>
                  </div>
                </template>
              </a-table-column>
            </a-table>
          </div>
        </template>
        <template v-if="resourceType === 'service' && relatedPods.length">
          <div class="related-section">
            <div class="section-title">后端 Pods ({{ relatedPods.length }})</div>
            <a-table :columns="podMiniColumns" :data-source="relatedPods" size="small" :pagination="false" row-key="name">
              <template #bodyCell="{ column, record }">
                <template v-if="column.key === 'name'"><a class="link" @click="goToResource('pod', record.namespace, record.name)">{{ record.name }}</a></template>
                <template v-if="column.key === 'status'"><a-tag :color="getPodStatusColor(record.status)">{{ record.status }}</a-tag></template>
                <template v-if="column.key === 'action'">
                  <a-space>
                    <a @click="showPodLogs(record)">日志</a>
                    <a @click="showPodTerminal(record)">终端</a>
                    <a-popconfirm title="确定删除此 Pod？" @confirm="deletePod(record)">
                      <a style="color: #ff4d4f">删除</a>
                    </a-popconfirm>
                  </a-space>
                </template>
              </template>
            </a-table>
          </div>
        </template>
        <template v-if="resourceType === 'pod' && detail?.owner_references?.length">
          <div class="related-section">
            <div class="section-title">所属控制器</div>
            <div v-for="owner in detail.owner_references" :key="owner.uid"><a class="link" @click="goToOwner(owner)">{{ owner.kind }}/{{ owner.name }}</a></div>
          </div>
        </template>
        <template v-if="resourceType === 'node' && relatedPods.length">
          <div class="related-section">
            <div class="section-title">运行的 Pods ({{ relatedPods.length }})</div>
            <a-table :columns="nodePodColumns" :data-source="relatedPods" size="small" :pagination="{ pageSize: 10 }" row-key="name">
              <template #bodyCell="{ column, record }">
                <template v-if="column.key === 'name'"><a class="link" @click="goToResource('pod', record.namespace, record.name)">{{ record.name }}</a></template>
                <template v-if="column.key === 'namespace'"><a class="link" @click="goToNamespace(record.namespace)">{{ record.namespace }}</a></template>
                <template v-if="column.key === 'status'"><a-tag :color="getPodStatusColor(record.status)">{{ record.status }}</a-tag></template>
              </template>
            </a-table>
          </div>
        </template>
      </a-card>

      <!-- 容器信息 -->
      <a-card title="容器" size="small" class="detail-card" v-if="detail?.containers?.length">
        <a-table :columns="containerColumns" :data-source="detail.containers" size="small" :pagination="false" row-key="name">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'image'"><a-tooltip :title="record.image"><span class="ellipsis-text">{{ record.image }}</span></a-tooltip></template>
            <template v-if="column.key === 'resources'">
              <div v-if="record.resources && (record.resources.requests || record.resources.limits)">
                <div v-if="record.resources.requests">请求: CPU {{ record.resources.requests.cpu || '-' }}, 内存 {{ record.resources.requests.memory || '-' }}</div>
                <div v-if="record.resources.limits">限制: CPU {{ record.resources.limits.cpu || '-' }}, 内存 {{ record.resources.limits.memory || '-' }}</div>
              </div>
              <span v-else>-</span>
            </template>
          </template>
        </a-table>
      </a-card>

      <!-- 端口信息 -->
      <a-card title="端口" size="small" class="detail-card" v-if="resourceType === 'service' && detail?.ports?.length">
        <a-table :columns="portColumns" :data-source="detail.ports" size="small" :pagination="false" row-key="port"></a-table>
      </a-card>

      <!-- HPA 管理 (仅 Deployment/StatefulSet) -->
      <a-card title="HPA 自动伸缩" size="small" class="detail-card" v-if="['deployment','statefulset'].includes(resourceType)">
        <template #extra>
          <a-space>
            <a-button v-if="!hpaData && !loadingHPA" type="primary" size="small" @click="showHPAModal('create')"><PlusOutlined /> 创建 HPA</a-button>
            <a-button v-if="!cronHPAData && !loadingCronHPA" size="small" @click="showCronHPAModal('create')"><ClockCircleOutlined /> 创建定时伸缩</a-button>
            <span v-else-if="!hpaData && loadingHPA">加载中...</span>
          </a-space>
        </template>
        <a-spin :spinning="loadingHPA || loadingCronHPA">
          <!-- 普通 HPA -->
          <template v-if="hpaData">
            <a-divider orientation="left" style="margin: 0 0 12px 0">基于指标的自动伸缩</a-divider>
            <a-descriptions :column="2" size="small" bordered>
              <a-descriptions-item label="HPA 名称">{{ hpaData.name }}</a-descriptions-item>
              <a-descriptions-item label="目标资源">{{ hpaData.target_kind }}/{{ hpaData.target_name }}</a-descriptions-item>
              <a-descriptions-item label="副本范围">{{ hpaData.min_replicas }} - {{ hpaData.max_replicas }}</a-descriptions-item>
              <a-descriptions-item label="当前/期望副本">
                <span v-if="hpaData.current_replicas === 0 && hpaData.desired_replicas === 0" style="color: #999">等待同步...</span>
                <span v-else>{{ hpaData.current_replicas }} / {{ hpaData.desired_replicas }}</span>
              </a-descriptions-item>
              <a-descriptions-item label="伸缩指标" :span="2">
                <a-tag v-for="m in hpaData.metrics" :key="m" color="blue">{{ m }}</a-tag>
              </a-descriptions-item>
            </a-descriptions>
            <div class="hpa-actions">
              <a-button type="link" size="small" @click="showHPAModal('edit')"><EditOutlined /> 编辑</a-button>
              <a-popconfirm title="确定删除此 HPA？" @confirm="deleteHPA">
                <a-button type="link" size="small" danger><DeleteOutlined /> 删除</a-button>
              </a-popconfirm>
            </div>
          </template>
          
          <!-- CronHPA -->
          <template v-if="cronHPAData">
            <a-divider orientation="left" :style="{ margin: hpaData ? '16px 0 12px 0' : '0 0 12px 0' }">定时自动伸缩</a-divider>
            <a-descriptions :column="2" size="small" bordered>
              <a-descriptions-item label="CronHPA 名称">{{ cronHPAData.name }}</a-descriptions-item>
              <a-descriptions-item label="状态">
                <a-tag :color="cronHPAData.enabled ? 'green' : 'default'">{{ cronHPAData.enabled ? '已启用' : '已禁用' }}</a-tag>
              </a-descriptions-item>
              <a-descriptions-item label="调度规则" :span="2">
                <div v-for="(schedule, idx) in cronHPAData.schedules" :key="idx" style="margin-bottom: 4px">
                  <a-tag color="purple">{{ schedule.name }}</a-tag>
                  <span style="margin: 0 8px">{{ schedule.cron }}</span>
                  <a-tag color="blue">副本数: {{ schedule.replicas }}</a-tag>
                  <span v-if="schedule.min_replicas && schedule.max_replicas" style="color: #999; font-size: 12px">
                    (范围: {{ schedule.min_replicas }}-{{ schedule.max_replicas }})
                  </span>
                </div>
              </a-descriptions-item>
            </a-descriptions>
            <div class="hpa-actions">
              <a-button type="link" size="small" @click="showCronHPAModal('edit')"><EditOutlined /> 编辑</a-button>
              <a-popconfirm title="确定删除此定时伸缩配置？" @confirm="deleteCronHPA">
                <a-button type="link" size="small" danger><DeleteOutlined /> 删除</a-button>
              </a-popconfirm>
            </div>
          </template>
          
          <template v-if="!hpaData && !cronHPAData && !loadingHPA && !loadingCronHPA">
            <a-empty description="未配置自动伸缩" :image="Empty.PRESENTED_IMAGE_SIMPLE" />
          </template>
        </a-spin>
      </a-card>

      <!-- 事件 -->
      <a-card title="事件" size="small" class="detail-card" v-if="events.length">
        <a-table :columns="eventColumns" :data-source="events" size="small" :pagination="{ pageSize: 5 }" row-key="name">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'type'"><a-tag :color="record.type === 'Normal' ? 'green' : 'orange'">{{ record.type }}</a-tag></template>
          </template>
        </a-table>
      </a-card>

      <!-- YAML -->
      <a-card title="YAML" size="small" class="detail-card">
        <a-button type="link" @click="showYAML = !showYAML" style="padding: 0">{{ showYAML ? '收起' : '展开' }} YAML</a-button>
        <pre v-if="showYAML" class="yaml-content">{{ yaml }}</pre>
      </a-card>
    </a-spin>

    <!-- 编辑弹窗 -->
    <a-modal v-model:open="editModalVisible" title="编辑资源" width="1000px" :footer="null" :destroyOnClose="true">
      <a-tabs v-model:activeKey="editMode" type="card">
        <a-tab-pane key="form" tab="表单编辑" v-if="canEditByForm">
          <div class="edit-form">
            <!-- Deployment/StatefulSet/DaemonSet 表单 -->
            <template v-if="['deployment','statefulset','daemonset'].includes(resourceType)">
              <a-collapse v-model:activeKey="formActiveKeys" :bordered="false">
                <a-collapse-panel key="basic" header="基本配置">
                  <a-row :gutter="16">
                    <a-col :span="8" v-if="resourceType !== 'daemonset'">
                      <a-form-item label="副本数"><a-input-number v-model:value="formData.replicas" :min="0" :max="100" style="width: 100%" /></a-form-item>
                    </a-col>
                    <a-col :span="8">
                      <a-form-item label="更新策略">
                        <a-select v-model:value="formData.strategy" style="width: 100%">
                          <a-select-option value="RollingUpdate">滚动更新</a-select-option>
                          <a-select-option value="Recreate" v-if="resourceType === 'deployment'">重建</a-select-option>
                          <a-select-option value="OnDelete" v-if="resourceType !== 'deployment'">删除时更新</a-select-option>
                        </a-select>
                      </a-form-item>
                    </a-col>
                    <a-col :span="8" v-if="formData.strategy === 'RollingUpdate'">
                      <a-form-item label="最大不可用"><a-input v-model:value="formData.maxUnavailable" placeholder="25%" /></a-form-item>
                    </a-col>
                  </a-row>
                </a-collapse-panel>
                <a-collapse-panel key="metadata" header="标签和注解">
                  <a-row :gutter="24">
                    <a-col :span="12">
                      <div class="form-section-title">Pod 标签</div>
                      <div v-for="(label, idx) in formData.podLabels" :key="idx" class="kv-row">
                        <a-input v-model:value="label.key" placeholder="Key" class="kv-input" />
                        <a-input v-model:value="label.value" placeholder="Value" class="kv-input" />
                        <a-button type="text" danger size="small" @click="formData.podLabels.splice(idx, 1)"><DeleteOutlined /></a-button>
                      </div>
                      <a-button type="dashed" size="small" block @click="formData.podLabels.push({ key: '', value: '' })"><PlusOutlined /> 添加</a-button>
                    </a-col>
                    <a-col :span="12">
                      <div class="form-section-title">Pod 注解</div>
                      <div v-for="(anno, idx) in formData.podAnnotations" :key="idx" class="kv-row">
                        <a-input v-model:value="anno.key" placeholder="Key" class="kv-input" />
                        <a-input v-model:value="anno.value" placeholder="Value" class="kv-input" />
                        <a-button type="text" danger size="small" @click="formData.podAnnotations.splice(idx, 1)"><DeleteOutlined /></a-button>
                      </div>
                      <a-button type="dashed" size="small" block @click="formData.podAnnotations.push({ key: '', value: '' })"><PlusOutlined /> 添加</a-button>
                    </a-col>
                  </a-row>
                </a-collapse-panel>
                <a-collapse-panel key="scheduling" header="调度配置">
                  <a-row :gutter="16">
                    <a-col :span="12">
                      <a-form-item label="服务账户"><a-input v-model:value="formData.serviceAccountName" placeholder="default" /></a-form-item>
                    </a-col>
                    <a-col :span="12">
                      <a-form-item label="重启策略">
                        <a-select v-model:value="formData.restartPolicy" style="width: 100%">
                          <a-select-option value="Always">Always</a-select-option>
                          <a-select-option value="OnFailure">OnFailure</a-select-option>
                          <a-select-option value="Never">Never</a-select-option>
                        </a-select>
                      </a-form-item>
                    </a-col>
                  </a-row>
                  <div class="form-section-title">节点选择器</div>
                  <div v-for="(sel, idx) in formData.nodeSelector" :key="idx" class="kv-row">
                    <a-input v-model:value="sel.key" placeholder="Key" class="kv-input" />
                    <a-input v-model:value="sel.value" placeholder="Value" class="kv-input" />
                    <a-button type="text" danger size="small" @click="formData.nodeSelector.splice(idx, 1)"><DeleteOutlined /></a-button>
                  </div>
                  <a-button type="dashed" size="small" @click="formData.nodeSelector.push({ key: '', value: '' })"><PlusOutlined /> 添加</a-button>
                </a-collapse-panel>

                <a-collapse-panel key="containers" header="容器配置">
                  <a-tabs v-model:activeKey="activeContainerIdx" type="card" size="small">
                    <a-tab-pane v-for="(container, idx) in formData.containers" :key="idx" :tab="container.name || `容器${idx+1}`">
                      <a-row :gutter="16">
                        <a-col :span="16"><a-form-item label="镜像" required><a-input v-model:value="container.image" placeholder="nginx:latest" /></a-form-item></a-col>
                        <a-col :span="8"><a-form-item label="拉取策略">
                          <a-select v-model:value="container.imagePullPolicy" style="width: 100%">
                            <a-select-option value="Always">Always</a-select-option>
                            <a-select-option value="IfNotPresent">IfNotPresent</a-select-option>
                            <a-select-option value="Never">Never</a-select-option>
                          </a-select>
                        </a-form-item></a-col>
                      </a-row>
                      <a-row :gutter="16">
                        <a-col :span="12"><a-form-item label="命令"><a-input v-model:value="container.command" placeholder='["sh","-c","..."]' /></a-form-item></a-col>
                        <a-col :span="12"><a-form-item label="参数"><a-input v-model:value="container.args" placeholder='["--port=8080"]' /></a-form-item></a-col>
                      </a-row>
                      <div class="form-section-title">资源配置</div>
                      <a-row :gutter="16">
                        <a-col :span="6"><a-form-item label="CPU请求"><a-input v-model:value="container.cpuRequest" placeholder="100m" /></a-form-item></a-col>
                        <a-col :span="6"><a-form-item label="CPU限制"><a-input v-model:value="container.cpuLimit" placeholder="500m" /></a-form-item></a-col>
                        <a-col :span="6"><a-form-item label="内存请求"><a-input v-model:value="container.memoryRequest" placeholder="128Mi" /></a-form-item></a-col>
                        <a-col :span="6"><a-form-item label="内存限制"><a-input v-model:value="container.memoryLimit" placeholder="512Mi" /></a-form-item></a-col>
                      </a-row>
                      <div class="form-section-title">端口</div>
                      <div v-for="(port, pIdx) in container.ports" :key="pIdx" class="inline-row">
                        <a-input v-model:value="port.name" placeholder="名称" style="width: 100px" />
                        <a-input-number v-model:value="port.containerPort" placeholder="端口" :min="1" :max="65535" style="width: 100px" />
                        <a-select v-model:value="port.protocol" style="width: 80px"><a-select-option value="TCP">TCP</a-select-option><a-select-option value="UDP">UDP</a-select-option></a-select>
                        <a-button type="text" danger size="small" @click="container.ports.splice(pIdx, 1)"><DeleteOutlined /></a-button>
                      </div>
                      <a-button type="dashed" size="small" @click="container.ports.push({ name: '', containerPort: 80, protocol: 'TCP' })"><PlusOutlined /> 添加端口</a-button>
                      <div class="form-section-title">环境变量</div>
                      <div v-for="(env, eIdx) in container.env" :key="eIdx" class="inline-row">
                        <a-input v-model:value="env.name" placeholder="变量名" style="width: 150px" />
                        <a-input v-model:value="env.value" placeholder="值" style="width: 200px" />
                        <a-button type="text" danger size="small" @click="container.env.splice(eIdx, 1)"><DeleteOutlined /></a-button>
                      </div>
                      <a-button type="dashed" size="small" @click="container.env.push({ name: '', value: '' })"><PlusOutlined /> 添加环境变量</a-button>
                      <div class="form-section-title">存储卷挂载</div>
                      <div v-for="(mount, mIdx) in container.volumeMounts" :key="mIdx" class="inline-row">
                        <a-input v-model:value="mount.name" placeholder="卷名称" style="width: 100px" />
                        <a-input v-model:value="mount.mountPath" placeholder="挂载路径" style="width: 150px" />
                        <a-input v-model:value="mount.subPath" placeholder="子路径" style="width: 100px" />
                        <a-checkbox v-model:checked="mount.readOnly">只读</a-checkbox>
                        <a-button type="text" danger size="small" @click="container.volumeMounts.splice(mIdx, 1)"><DeleteOutlined /></a-button>
                      </div>
                      <a-button type="dashed" size="small" @click="container.volumeMounts.push({ name: '', mountPath: '', subPath: '', readOnly: false })"><PlusOutlined /> 添加挂载</a-button>
                      <div class="form-section-title">健康检查</div>
                      <!-- 存活探针 -->
                      <div class="probe-section">
                        <div class="probe-header">
                          <span>存活探针 (Liveness)</span>
                          <a-select v-model:value="container.livenessProbe.type" style="width: 100px" size="small">
                            <a-select-option value="">禁用</a-select-option><a-select-option value="httpGet">HTTP</a-select-option><a-select-option value="tcpSocket">TCP</a-select-option><a-select-option value="exec">命令</a-select-option>
                          </a-select>
                        </div>
                        <template v-if="container.livenessProbe.type">
                          <div class="probe-config">
                            <div class="probe-row" v-if="container.livenessProbe.type === 'httpGet'">
                              <span class="probe-label">路径:</span><a-input v-model:value="container.livenessProbe.path" placeholder="/health" style="width: 100px" />
                              <span class="probe-label">端口:</span><a-input-number v-model:value="container.livenessProbe.port" :min="1" :max="65535" style="width: 80px" />
                              <span class="probe-label">协议:</span><a-select v-model:value="container.livenessProbe.scheme" style="width: 80px"><a-select-option value="HTTP">HTTP</a-select-option><a-select-option value="HTTPS">HTTPS</a-select-option></a-select>
                            </div>
                            <div class="probe-row" v-if="container.livenessProbe.type === 'tcpSocket'">
                              <span class="probe-label">端口:</span><a-input-number v-model:value="container.livenessProbe.port" :min="1" :max="65535" style="width: 100px" />
                            </div>
                            <div class="probe-row" v-if="container.livenessProbe.type === 'exec'">
                              <span class="probe-label">命令:</span><a-input v-model:value="container.livenessProbe.command" placeholder='["cat", "/tmp/healthy"]' style="width: 300px" />
                            </div>
                            <div class="probe-row">
                              <span class="probe-label">初始延迟:</span><a-input-number v-model:value="container.livenessProbe.initialDelaySeconds" :min="0" style="width: 60px" /><span class="probe-unit">秒</span>
                              <span class="probe-label">间隔:</span><a-input-number v-model:value="container.livenessProbe.periodSeconds" :min="1" style="width: 60px" /><span class="probe-unit">秒</span>
                              <span class="probe-label">超时:</span><a-input-number v-model:value="container.livenessProbe.timeoutSeconds" :min="1" style="width: 60px" /><span class="probe-unit">秒</span>
                              <span class="probe-label">失败阈值:</span><a-input-number v-model:value="container.livenessProbe.failureThreshold" :min="1" style="width: 60px" />
                            </div>
                          </div>
                        </template>
                      </div>
                      <!-- 就绪探针 -->
                      <div class="probe-section">
                        <div class="probe-header">
                          <span>就绪探针 (Readiness)</span>
                          <a-select v-model:value="container.readinessProbe.type" style="width: 100px" size="small">
                            <a-select-option value="">禁用</a-select-option><a-select-option value="httpGet">HTTP</a-select-option><a-select-option value="tcpSocket">TCP</a-select-option><a-select-option value="exec">命令</a-select-option>
                          </a-select>
                        </div>
                        <template v-if="container.readinessProbe.type">
                          <div class="probe-config">
                            <div class="probe-row" v-if="container.readinessProbe.type === 'httpGet'">
                              <span class="probe-label">路径:</span><a-input v-model:value="container.readinessProbe.path" placeholder="/ready" style="width: 100px" />
                              <span class="probe-label">端口:</span><a-input-number v-model:value="container.readinessProbe.port" :min="1" :max="65535" style="width: 80px" />
                              <span class="probe-label">协议:</span><a-select v-model:value="container.readinessProbe.scheme" style="width: 80px"><a-select-option value="HTTP">HTTP</a-select-option><a-select-option value="HTTPS">HTTPS</a-select-option></a-select>
                            </div>
                            <div class="probe-row" v-if="container.readinessProbe.type === 'tcpSocket'">
                              <span class="probe-label">端口:</span><a-input-number v-model:value="container.readinessProbe.port" :min="1" :max="65535" style="width: 100px" />
                            </div>
                            <div class="probe-row" v-if="container.readinessProbe.type === 'exec'">
                              <span class="probe-label">命令:</span><a-input v-model:value="container.readinessProbe.command" placeholder='["cat", "/tmp/ready"]' style="width: 300px" />
                            </div>
                            <div class="probe-row">
                              <span class="probe-label">初始延迟:</span><a-input-number v-model:value="container.readinessProbe.initialDelaySeconds" :min="0" style="width: 60px" /><span class="probe-unit">秒</span>
                              <span class="probe-label">间隔:</span><a-input-number v-model:value="container.readinessProbe.periodSeconds" :min="1" style="width: 60px" /><span class="probe-unit">秒</span>
                              <span class="probe-label">超时:</span><a-input-number v-model:value="container.readinessProbe.timeoutSeconds" :min="1" style="width: 60px" /><span class="probe-unit">秒</span>
                              <span class="probe-label">成功阈值:</span><a-input-number v-model:value="container.readinessProbe.successThreshold" :min="1" style="width: 60px" />
                            </div>
                          </div>
                        </template>
                      </div>
                      <!-- 启动探针 -->
                      <div class="probe-section">
                        <div class="probe-header">
                          <span>启动探针 (Startup)</span>
                          <a-select v-model:value="container.startupProbe.type" style="width: 100px" size="small">
                            <a-select-option value="">禁用</a-select-option><a-select-option value="httpGet">HTTP</a-select-option><a-select-option value="tcpSocket">TCP</a-select-option><a-select-option value="exec">命令</a-select-option>
                          </a-select>
                        </div>
                        <template v-if="container.startupProbe.type">
                          <div class="probe-config">
                            <div class="probe-row" v-if="container.startupProbe.type === 'httpGet'">
                              <span class="probe-label">路径:</span><a-input v-model:value="container.startupProbe.path" placeholder="/startup" style="width: 100px" />
                              <span class="probe-label">端口:</span><a-input-number v-model:value="container.startupProbe.port" :min="1" :max="65535" style="width: 80px" />
                              <span class="probe-label">协议:</span><a-select v-model:value="container.startupProbe.scheme" style="width: 80px"><a-select-option value="HTTP">HTTP</a-select-option><a-select-option value="HTTPS">HTTPS</a-select-option></a-select>
                            </div>
                            <div class="probe-row" v-if="container.startupProbe.type === 'tcpSocket'">
                              <span class="probe-label">端口:</span><a-input-number v-model:value="container.startupProbe.port" :min="1" :max="65535" style="width: 100px" />
                            </div>
                            <div class="probe-row" v-if="container.startupProbe.type === 'exec'">
                              <span class="probe-label">命令:</span><a-input v-model:value="container.startupProbe.command" placeholder='["cat", "/tmp/started"]' style="width: 300px" />
                            </div>
                            <div class="probe-row">
                              <span class="probe-label">初始延迟:</span><a-input-number v-model:value="container.startupProbe.initialDelaySeconds" :min="0" style="width: 60px" /><span class="probe-unit">秒</span>
                              <span class="probe-label">间隔:</span><a-input-number v-model:value="container.startupProbe.periodSeconds" :min="1" style="width: 60px" /><span class="probe-unit">秒</span>
                              <span class="probe-label">超时:</span><a-input-number v-model:value="container.startupProbe.timeoutSeconds" :min="1" style="width: 60px" /><span class="probe-unit">秒</span>
                              <span class="probe-label">失败阈值:</span><a-input-number v-model:value="container.startupProbe.failureThreshold" :min="1" style="width: 60px" />
                            </div>
                          </div>
                        </template>
                      </div>
                    </a-tab-pane>
                  </a-tabs>
                </a-collapse-panel>
                <a-collapse-panel key="volumes" header="存储卷">
                  <div v-for="(vol, vIdx) in formData.volumes" :key="vIdx" class="volume-row">
                    <a-input v-model:value="vol.name" placeholder="卷名称" style="width: 120px" />
                    <a-select v-model:value="vol.type" style="width: 120px">
                      <a-select-option value="emptyDir">EmptyDir</a-select-option><a-select-option value="configMap">ConfigMap</a-select-option>
                      <a-select-option value="secret">Secret</a-select-option><a-select-option value="pvc">PVC</a-select-option><a-select-option value="hostPath">HostPath</a-select-option>
                    </a-select>
                    <a-input v-if="vol.type === 'configMap'" v-model:value="vol.configMapName" placeholder="ConfigMap名称" style="width: 150px" />
                    <a-input v-if="vol.type === 'secret'" v-model:value="vol.secretName" placeholder="Secret名称" style="width: 150px" />
                    <a-input v-if="vol.type === 'pvc'" v-model:value="vol.pvcName" placeholder="PVC名称" style="width: 150px" />
                    <a-input v-if="vol.type === 'hostPath'" v-model:value="vol.hostPath" placeholder="主机路径" style="width: 150px" />
                    <a-button type="text" danger size="small" @click="formData.volumes.splice(vIdx, 1)"><DeleteOutlined /></a-button>
                  </div>
                  <a-button type="dashed" size="small" @click="formData.volumes.push({ name: '', type: 'emptyDir' })"><PlusOutlined /> 添加存储卷</a-button>
                </a-collapse-panel>
              </a-collapse>
            </template>

            <!-- Service 表单 -->
            <template v-if="resourceType === 'service'">
              <a-collapse v-model:activeKey="formActiveKeys" :bordered="false">
                <a-collapse-panel key="basic" header="基本配置">
                  <a-row :gutter="16">
                    <a-col :span="8"><a-form-item label="类型">
                      <a-select v-model:value="formData.type" style="width: 100%">
                        <a-select-option value="ClusterIP">ClusterIP</a-select-option><a-select-option value="NodePort">NodePort</a-select-option>
                        <a-select-option value="LoadBalancer">LoadBalancer</a-select-option><a-select-option value="ExternalName">ExternalName</a-select-option>
                      </a-select>
                    </a-form-item></a-col>
                    <a-col :span="8"><a-form-item label="会话亲和性">
                      <a-select v-model:value="formData.sessionAffinity" style="width: 100%"><a-select-option value="None">None</a-select-option><a-select-option value="ClientIP">ClientIP</a-select-option></a-select>
                    </a-form-item></a-col>
                    <a-col :span="8" v-if="formData.type === 'NodePort' || formData.type === 'LoadBalancer'"><a-form-item label="外部流量策略">
                      <a-select v-model:value="formData.externalTrafficPolicy" style="width: 100%"><a-select-option value="Cluster">Cluster</a-select-option><a-select-option value="Local">Local</a-select-option></a-select>
                    </a-form-item></a-col>
                  </a-row>
                </a-collapse-panel>
                <a-collapse-panel key="ports" header="端口配置">
                  <div v-for="(port, idx) in formData.ports" :key="idx" class="inline-row">
                    <a-input v-model:value="port.name" placeholder="名称" style="width: 80px" />
                    <a-input-number v-model:value="port.port" placeholder="端口" :min="1" :max="65535" style="width: 90px" />
                    <a-input v-model:value="port.targetPort" placeholder="目标端口" style="width: 90px" />
                    <a-select v-model:value="port.protocol" style="width: 70px"><a-select-option value="TCP">TCP</a-select-option><a-select-option value="UDP">UDP</a-select-option></a-select>
                    <a-input-number v-if="formData.type === 'NodePort'" v-model:value="port.nodePort" placeholder="NodePort" :min="30000" :max="32767" style="width: 100px" />
                    <a-button type="text" danger size="small" @click="formData.ports.splice(idx, 1)" v-if="formData.ports.length > 1"><DeleteOutlined /></a-button>
                  </div>
                  <a-button type="dashed" size="small" @click="formData.ports.push({ name: '', port: 80, targetPort: '80', protocol: 'TCP' })"><PlusOutlined /> 添加端口</a-button>
                </a-collapse-panel>
                <a-collapse-panel key="selector" header="选择器">
                  <div v-for="(sel, idx) in formData.selector" :key="idx" class="kv-row">
                    <a-input v-model:value="sel.key" placeholder="Key" class="kv-input" />
                    <a-input v-model:value="sel.value" placeholder="Value" class="kv-input" />
                    <a-button type="text" danger size="small" @click="formData.selector.splice(idx, 1)"><DeleteOutlined /></a-button>
                  </div>
                  <a-button type="dashed" size="small" @click="formData.selector.push({ key: '', value: '' })"><PlusOutlined /> 添加</a-button>
                </a-collapse-panel>
              </a-collapse>
            </template>

            <!-- ConfigMap/Secret 表单 -->
            <template v-if="resourceType === 'configmap' || resourceType === 'secret'">
              <a-alert v-if="resourceType === 'secret'" message="Secret 数据会自动进行 Base64 编码" type="info" show-icon style="margin-bottom: 16px" />
              <a-collapse v-model:activeKey="formActiveKeys" :bordered="false">
                <a-collapse-panel key="metadata" header="标签和注解">
                  <a-row :gutter="24">
                    <a-col :span="12">
                      <div class="form-section-title">标签</div>
                      <div v-for="(label, idx) in formData.labels" :key="idx" class="kv-row">
                        <a-input v-model:value="label.key" placeholder="Key" class="kv-input" />
                        <a-input v-model:value="label.value" placeholder="Value" class="kv-input" />
                        <a-button type="text" danger size="small" @click="formData.labels.splice(idx, 1)"><DeleteOutlined /></a-button>
                      </div>
                      <a-button type="dashed" size="small" block @click="formData.labels.push({ key: '', value: '' })"><PlusOutlined /> 添加</a-button>
                    </a-col>
                    <a-col :span="12">
                      <div class="form-section-title">注解</div>
                      <div v-for="(anno, idx) in formData.annotations" :key="idx" class="kv-row">
                        <a-input v-model:value="anno.key" placeholder="Key" class="kv-input" />
                        <a-input v-model:value="anno.value" placeholder="Value" class="kv-input" />
                        <a-button type="text" danger size="small" @click="formData.annotations.splice(idx, 1)"><DeleteOutlined /></a-button>
                      </div>
                      <a-button type="dashed" size="small" block @click="formData.annotations.push({ key: '', value: '' })"><PlusOutlined /> 添加</a-button>
                    </a-col>
                  </a-row>
                </a-collapse-panel>
                <a-collapse-panel key="data" header="数据">
                  <div v-for="(item, idx) in formData.data" :key="idx" class="data-item">
                    <div class="data-item-header">
                      <a-input v-model:value="item.key" placeholder="Key" style="width: 200px" />
                      <a-button type="text" danger size="small" @click="formData.data.splice(idx, 1)"><DeleteOutlined /></a-button>
                    </div>
                    <a-textarea v-model:value="item.value" :placeholder="resourceType === 'secret' ? '明文内容' : 'Value'" :rows="3" />
                  </div>
                  <a-button type="dashed" block @click="formData.data.push({ key: '', value: '' })"><PlusOutlined /> 添加数据</a-button>
                </a-collapse-panel>
              </a-collapse>
            </template>

            <!-- Ingress 表单 -->
            <template v-if="resourceType === 'ingress'">
              <a-collapse v-model:activeKey="formActiveKeys" :bordered="false">
                <a-collapse-panel key="basic" header="基本配置">
                  <a-form-item label="Ingress Class"><a-input v-model:value="formData.ingressClassName" placeholder="nginx" style="width: 200px" /></a-form-item>
                </a-collapse-panel>
                <a-collapse-panel key="annotations" header="注解">
                  <div v-for="(anno, idx) in formData.annotations" :key="idx" class="kv-row">
                    <a-input v-model:value="anno.key" placeholder="Key" style="width: 280px" />
                    <a-input v-model:value="anno.value" placeholder="Value" style="width: 200px" />
                    <a-button type="text" danger size="small" @click="formData.annotations.splice(idx, 1)"><DeleteOutlined /></a-button>
                  </div>
                  <a-button type="dashed" size="small" @click="formData.annotations.push({ key: '', value: '' })"><PlusOutlined /> 添加</a-button>
                </a-collapse-panel>
                <a-collapse-panel key="tls" header="TLS 配置">
                  <div v-for="(tls, idx) in formData.tls" :key="idx" class="inline-row">
                    <a-input v-model:value="tls.secretName" placeholder="Secret 名称" style="width: 150px" />
                    <a-input v-model:value="tls.hosts" placeholder="域名 (逗号分隔)" style="width: 250px" />
                    <a-button type="text" danger size="small" @click="formData.tls.splice(idx, 1)"><DeleteOutlined /></a-button>
                  </div>
                  <a-button type="dashed" size="small" @click="formData.tls.push({ secretName: '', hosts: '' })"><PlusOutlined /> 添加 TLS</a-button>
                </a-collapse-panel>
                <a-collapse-panel key="rules" header="路由规则">
                  <div v-for="(rule, ruleIdx) in formData.rules" :key="ruleIdx" class="rule-section">
                    <div class="rule-header">
                      <a-input v-model:value="rule.host" placeholder="域名" style="width: 250px" />
                      <a-button type="text" danger size="small" @click="formData.rules.splice(ruleIdx, 1)" v-if="formData.rules.length > 1"><DeleteOutlined /> 删除规则</a-button>
                    </div>
                    <div v-for="(path, pathIdx) in rule.paths" :key="pathIdx" class="inline-row">
                      <a-input v-model:value="path.path" placeholder="路径" style="width: 100px" />
                      <a-select v-model:value="path.pathType" style="width: 130px"><a-select-option value="Prefix">Prefix</a-select-option><a-select-option value="Exact">Exact</a-select-option></a-select>
                      <a-input v-model:value="path.serviceName" placeholder="服务名" style="width: 130px" />
                      <a-input-number v-model:value="path.servicePort" placeholder="端口" :min="1" :max="65535" style="width: 80px" />
                      <a-button type="text" danger size="small" @click="rule.paths.splice(pathIdx, 1)" v-if="rule.paths.length > 1"><DeleteOutlined /></a-button>
                    </div>
                    <a-button type="dashed" size="small" @click="rule.paths.push({ path: '/', pathType: 'Prefix', serviceName: '', servicePort: 80 })"><PlusOutlined /> 添加路径</a-button>
                  </div>
                  <a-button type="dashed" @click="formData.rules.push({ host: '', paths: [{ path: '/', pathType: 'Prefix', serviceName: '', servicePort: 80 }] })"><PlusOutlined /> 添加规则</a-button>
                </a-collapse-panel>
              </a-collapse>
            </template>

            <!-- PVC 表单 -->
            <template v-if="resourceType === 'pvc'">
              <a-row :gutter="16">
                <a-col :span="8"><a-form-item label="存储类"><a-input v-model:value="formData.storageClassName" placeholder="standard" /></a-form-item></a-col>
                <a-col :span="8"><a-form-item label="容量"><a-input v-model:value="formData.storage" placeholder="1Gi" /></a-form-item></a-col>
                <a-col :span="8"><a-form-item label="访问模式">
                  <a-checkbox-group v-model:value="formData.accessModes">
                    <a-checkbox value="ReadWriteOnce">RWO</a-checkbox><a-checkbox value="ReadOnlyMany">ROX</a-checkbox><a-checkbox value="ReadWriteMany">RWX</a-checkbox>
                  </a-checkbox-group>
                </a-form-item></a-col>
              </a-row>
            </template>
          </div>
        </a-tab-pane>
        <a-tab-pane key="yaml" tab="YAML 编辑">
          <a-alert message="直接编辑 YAML，保存后将应用到集群" type="warning" show-icon style="margin-bottom: 12px" />
          <a-textarea v-model:value="editYaml" :rows="22" class="yaml-editor" />
        </a-tab-pane>
      </a-tabs>
      <div class="modal-footer"><a-button @click="editModalVisible = false">取消</a-button><a-button type="primary" :loading="saving" @click="handleSave">保存</a-button></div>
    </a-modal>

    <!-- 创建关联资源弹窗 -->
    <a-modal v-model:open="createRelatedVisible" :title="createRelatedType === 'service' ? '创建 Service' : '创建 Ingress'" width="700px" @ok="handleCreateRelatedSubmit" :confirm-loading="creatingRelated">
      <!-- 创建 Service -->
      <template v-if="createRelatedType === 'service'">
        <a-form :label-col="{ span: 5 }">
          <a-form-item label="名称"><a-input v-model:value="relatedForm.name" :placeholder="props.name + '-svc'" /></a-form-item>
          <a-form-item label="类型">
            <a-select v-model:value="relatedForm.serviceType" style="width: 200px">
              <a-select-option value="ClusterIP">ClusterIP</a-select-option>
              <a-select-option value="NodePort">NodePort</a-select-option>
              <a-select-option value="LoadBalancer">LoadBalancer</a-select-option>
            </a-select>
          </a-form-item>
          <a-form-item label="端口映射">
            <div v-for="(port, idx) in relatedForm.ports" :key="idx" class="inline-row">
              <a-input v-model:value="port.name" placeholder="名称" style="width: 80px" />
              <a-input-number v-model:value="port.port" placeholder="Service端口" :min="1" :max="65535" style="width: 110px" />
              <a-input-number v-model:value="port.targetPort" placeholder="容器端口" :min="1" :max="65535" style="width: 110px" />
              <a-select v-model:value="port.protocol" style="width: 70px"><a-select-option value="TCP">TCP</a-select-option><a-select-option value="UDP">UDP</a-select-option></a-select>
              <a-button type="text" danger size="small" @click="relatedForm.ports.splice(idx, 1)" v-if="relatedForm.ports.length > 1"><DeleteOutlined /></a-button>
            </div>
            <a-button type="dashed" size="small" @click="relatedForm.ports.push({ name: '', port: 80, targetPort: 80, protocol: 'TCP' })"><PlusOutlined /> 添加端口</a-button>
          </a-form-item>
          <a-form-item label="选择器">
            <div class="selector-preview">
              <a-tag v-for="(v, k) in relatedForm.selector" :key="k" color="blue">{{ k }}={{ v }}</a-tag>
            </div>
            <a-typography-text type="secondary">自动从 {{ resourceType }} 的 Pod 标签获取</a-typography-text>
          </a-form-item>
        </a-form>
      </template>
      <!-- 创建 Ingress -->
      <template v-if="createRelatedType === 'ingress'">
        <a-form :label-col="{ span: 5 }">
          <a-form-item label="名称"><a-input v-model:value="relatedForm.name" :placeholder="props.name + '-ingress'" /></a-form-item>
          <a-form-item label="Ingress Class"><a-input v-model:value="relatedForm.ingressClassName" placeholder="nginx" /></a-form-item>
          <a-form-item label="域名"><a-input v-model:value="relatedForm.host" placeholder="example.com" /></a-form-item>
          <a-form-item label="路径"><a-input v-model:value="relatedForm.path" placeholder="/" /></a-form-item>
          <a-form-item label="后端服务">
            <a-select v-model:value="relatedForm.serviceName" style="width: 200px" placeholder="选择 Service">
              <a-select-option v-for="svc in availableServices" :key="svc.name" :value="svc.name">{{ svc.name }}</a-select-option>
            </a-select>
            <a-input-number v-model:value="relatedForm.servicePort" placeholder="端口" :min="1" :max="65535" style="width: 100px; margin-left: 8px" />
          </a-form-item>
          <a-form-item label="TLS">
            <a-switch v-model:checked="relatedForm.enableTls" />
            <template v-if="relatedForm.enableTls">
              <a-input v-model:value="relatedForm.tlsSecretName" placeholder="TLS Secret 名称" style="width: 200px; margin-left: 12px" />
            </template>
          </a-form-item>
        </a-form>
      </template>
    </a-modal>

    <!-- Pod 日志抽屉 -->
    <a-drawer
      v-model:open="logsDrawerVisible"
      :title="`日志 - ${currentPod?.name}`"
      width="80%"
      placement="right"
      :destroy-on-close="true"
    >
      <PodLogs
        v-if="logsDrawerVisible && currentPod"
        :cluster-id="clusterId"
        :namespace="currentPod.namespace || namespace"
        :pod-name="currentPod.name"
      />
    </a-drawer>

    <!-- Pod 终端抽屉 -->
    <a-drawer
      v-model:open="terminalDrawerVisible"
      :title="`终端 - ${currentPod?.name}`"
      width="80%"
      placement="right"
      :destroy-on-close="true"
    >
      <PodTerminal
        v-if="terminalDrawerVisible && currentPod"
        :cluster-id="clusterId"
        :namespace="currentPod.namespace || namespace"
        :pod-name="currentPod.name"
      />
    </a-drawer>

    <!-- HPA 创建/编辑弹窗 -->
    <a-modal v-model:open="hpaModalVisible" :title="hpaModalMode === 'create' ? '创建 HPA' : '编辑 HPA'" width="600px" @ok="handleHPASubmit" :confirm-loading="savingHPA">
      <a-form :label-col="{ span: 6 }">
        <a-form-item label="HPA 名称" required v-if="hpaModalMode === 'create'">
          <a-input v-model:value="hpaForm.name" :placeholder="name + '-hpa'" />
        </a-form-item>
        <a-form-item label="目标资源">
          <a-input :value="`${resourceType === 'deployment' ? 'Deployment' : 'StatefulSet'}/${name}`" disabled />
        </a-form-item>
        <a-form-item label="最小副本数" required>
          <a-input-number v-model:value="hpaForm.min_replicas" :min="1" :max="hpaForm.max_replicas" style="width: 100%" />
        </a-form-item>
        <a-form-item label="最大副本数" required>
          <a-input-number v-model:value="hpaForm.max_replicas" :min="hpaForm.min_replicas" :max="1000" style="width: 100%" />
        </a-form-item>
        <a-form-item label="CPU 目标使用率">
          <a-input-number v-model:value="hpaForm.cpu_target_percent" :min="1" :max="100" style="width: 100%" addon-after="%" placeholder="留空则不使用 CPU 指标" />
        </a-form-item>
        <a-form-item label="内存目标使用率">
          <a-input-number v-model:value="hpaForm.mem_target_percent" :min="1" :max="100" style="width: 100%" addon-after="%" placeholder="留空则不使用内存指标" />
        </a-form-item>
        <a-alert v-if="!hpaForm.cpu_target_percent && !hpaForm.mem_target_percent" message="请至少配置一个伸缩指标（CPU 或内存）" type="warning" show-icon />
      </a-form>
    </a-modal>

    <!-- CronHPA 创建/编辑弹窗 -->
    <a-modal v-model:open="cronHPAModalVisible" :title="cronHPAModalMode === 'create' ? '创建定时伸缩' : '编辑定时伸缩'" width="800px" @ok="handleCronHPASubmit" :confirm-loading="savingCronHPA">
      <a-form :label-col="{ span: 5 }">
        <a-form-item label="名称" required v-if="cronHPAModalMode === 'create'">
          <a-input v-model:value="cronHPAForm.name" :placeholder="name + '-cron-hpa'" />
        </a-form-item>
        <a-form-item label="目标资源">
          <a-input :value="`${resourceType === 'deployment' ? 'Deployment' : 'StatefulSet'}/${name}`" disabled />
        </a-form-item>
        <a-form-item label="启用状态">
          <a-switch v-model:checked="cronHPAForm.enabled" checked-children="启用" un-checked-children="禁用" />
        </a-form-item>
        <a-form-item label="调度规则" required>
          <div v-for="(schedule, idx) in cronHPAForm.schedules" :key="idx" class="cron-schedule-item">
            <a-card size="small" :title="`规则 ${idx + 1}`">
              <template #extra>
                <a-button type="text" danger size="small" @click="cronHPAForm.schedules.splice(idx, 1)" v-if="cronHPAForm.schedules.length > 1">
                  <DeleteOutlined />
                </a-button>
              </template>
              <a-row :gutter="16">
                <a-col :span="12">
                  <a-form-item label="规则名称" :label-col="{ span: 8 }">
                    <a-input v-model:value="schedule.name" placeholder="如：工作时间扩容" />
                  </a-form-item>
                </a-col>
                <a-col :span="12">
                  <a-form-item label="Cron 表达式" :label-col="{ span: 8 }">
                    <a-input v-model:value="schedule.cron" placeholder="0 9 * * 1-5" />
                  </a-form-item>
                </a-col>
              </a-row>
              <a-alert 
                message="CronHPA 用于定时调整 HPA 的伸缩范围" 
                description="在不同时间段设置不同的 min/max 副本数，让 HPA 在该范围内自动伸缩。例如：工作时间 min=10/max=50，下班时间 min=2/max=10。" 
                type="info" 
                show-icon 
                style="margin-bottom: 12px"
              />
              <a-row :gutter="16">
                <a-col :span="12">
                  <a-form-item label="最小副本数" :label-col="{ span: 10 }" required>
                    <a-input-number v-model:value="schedule.min_replicas" :min="1" style="width: 100%" placeholder="HPA 最小副本" />
                  </a-form-item>
                </a-col>
                <a-col :span="12">
                  <a-form-item label="最大副本数" :label-col="{ span: 10 }" required>
                    <a-input-number v-model:value="schedule.max_replicas" :min="1" style="width: 100%" placeholder="HPA 最大副本" />
                  </a-form-item>
                </a-col>
              </a-row>
            </a-card>
          </div>
          <a-button type="dashed" block @click="addCronSchedule" style="margin-top: 8px">
            <PlusOutlined /> 添加调度规则
          </a-button>
        </a-form-item>
      </a-form>
    </a-modal>
  </a-drawer>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { k8sResourceApi, k8sNodeApi, k8sEventApi, k8sPodApi, k8sHPAApi, k8sCronHPAApi } from '@/services/k8s'
import { message, Empty } from 'ant-design-vue'
import { DownOutlined, DeleteOutlined, PlusOutlined, EditOutlined, ClockCircleOutlined } from '@ant-design/icons-vue'
import * as jsYaml from 'js-yaml'
import PodLogs from './PodLogs.vue'
import PodTerminal from './PodTerminal.vue'

const props = defineProps<{ visible: boolean; clusterId: number; resourceType: string; namespace: string; name: string }>()
const emit = defineEmits(['close', 'navigate', 'refresh'])

const loading = ref(false)
const detail = ref<any>(null)
const yaml = ref('')
const showYAML = ref(false)
const events = ref<any[]>([])
const relatedPods = ref<any[]>([])
const relatedServices = ref<any[]>([])
const relatedIngresses = ref<any[]>([])
const editModalVisible = ref(false)
const editYaml = ref('')
const editMode = ref('form')
const saving = ref(false)
const formData = ref<any>({})
const formActiveKeys = ref(['basic', 'containers'])
const activeContainerIdx = ref(0)

// 创建关联资源
const createRelatedVisible = ref(false)
const createRelatedType = ref<'service' | 'ingress'>('service')
const creatingRelated = ref(false)
const relatedForm = ref<any>({})
const availableServices = ref<any[]>([])

const canEditByForm = computed(() => ['deployment', 'statefulset', 'daemonset', 'service', 'configmap', 'secret', 'ingress', 'pvc'].includes(props.resourceType))
const title = computed(() => {
  const labels: Record<string, string> = { deployment: 'Deployment', statefulset: 'StatefulSet', daemonset: 'DaemonSet', pod: 'Pod', service: 'Service', ingress: 'Ingress', configmap: 'ConfigMap', secret: 'Secret', node: '节点', pvc: 'PVC', pv: 'PV', storageclass: 'StorageClass', namespace: '命名空间' }
  return `${labels[props.resourceType] || props.resourceType} - ${props.name}`
})
const hasRelatedResources = computed(() => {
  const result = (
    (['deployment', 'statefulset', 'daemonset'].includes(props.resourceType) && (relatedPods.value.length || relatedServices.value.length || relatedIngresses.value.length)) ||
    (props.resourceType === 'service' && relatedPods.value.length) ||
    (props.resourceType === 'pod' && detail.value?.owner_references?.length) ||
    (props.resourceType === 'node' && relatedPods.value.length)
  )
  console.log('[hasRelatedResources]', {
    resourceType: props.resourceType,
    pods: relatedPods.value.length,
    services: relatedServices.value.length,
    ingresses: relatedIngresses.value.length,
    result
  })
  return result
})

const podMiniColumns = [
  { title: '名称', key: 'name', dataIndex: 'name' },
  { title: '状态', key: 'status', width: 100 },
  { title: 'IP', dataIndex: 'ip', key: 'ip', width: 120 },
  { title: '重启', dataIndex: 'restarts', key: 'restarts', width: 60 },
  { title: '操作', key: 'action', width: 180 }
]
const nodePodColumns = [{ title: '名称', key: 'name', dataIndex: 'name' }, { title: '命名空间', key: 'namespace', dataIndex: 'namespace', width: 120 }, { title: '状态', key: 'status', width: 100 }, { title: 'IP', dataIndex: 'ip', key: 'ip', width: 120 }]
const containerColumns = [{ title: '名称', dataIndex: 'name', key: 'name', width: 150 }, { title: '镜像', key: 'image', dataIndex: 'image' }, { title: '资源', key: 'resources', width: 250 }]
const portColumns = [{ title: '名称', dataIndex: 'name', key: 'name' }, { title: '端口', dataIndex: 'port', key: 'port' }, { title: '目标端口', dataIndex: 'target_port', key: 'target_port' }, { title: '协议', dataIndex: 'protocol', key: 'protocol' }, { title: 'NodePort', dataIndex: 'node_port', key: 'node_port' }]
const eventColumns = [{ title: '类型', key: 'type', width: 80 }, { title: '原因', dataIndex: 'reason', key: 'reason', width: 120 }, { title: '消息', dataIndex: 'message', key: 'message' }, { title: '时间', dataIndex: 'last_timestamp', key: 'last_timestamp', width: 150 }]

const getPodStatusColor = (status: string) => ({ Running: 'green', Pending: 'orange', Succeeded: 'blue', Failed: 'red', Unknown: 'default' }[status] || 'default')

const fetchDetail = async () => {
  if (!props.visible || !props.name) return
  loading.value = true
  detail.value = null; events.value = []; relatedPods.value = []; relatedServices.value = []; relatedIngresses.value = []; yaml.value = ''; showYAML.value = false
  try {
    const res = await k8sResourceApi.getResourceDetail(props.clusterId, props.resourceType, props.namespace, props.name)
    if (res.code === 0) detail.value = res.data
    const yamlRes = await k8sResourceApi.getResourceYAML(props.clusterId, props.resourceType, props.namespace || 'default', props.name)
    if (yamlRes.code === 0) yaml.value = yamlRes.data || ''
    await fetchEvents(); await fetchRelatedResources()
  } catch (e: any) { message.error(e.message || '获取详情失败') }
  finally { loading.value = false }
}
const fetchEvents = async () => { try { const res = await k8sEventApi.getResourceEvents(props.clusterId, props.resourceType, props.namespace, props.name); if (res.code === 0) events.value = res.data || [] } catch (e) {} }
const fetchRelatedResources = async () => {
  console.log('[fetchRelatedResources] Starting...')
  try {
    if (['deployment', 'statefulset', 'daemonset'].includes(props.resourceType)) {
      console.log('[fetchRelatedResources] Fetching for', props.resourceType)
      
      // 获取关联的 Pods
      const podsRes = await k8sResourceApi.getRelatedPods(props.clusterId, props.resourceType, props.namespace, props.name)
      if (podsRes.code === 0) relatedPods.value = podsRes.data || []
      console.log('[fetchRelatedResources] Pods loaded:', relatedPods.value.length)
      
      // 获取关联的 Services（通过 selector 匹配）
      console.log('[fetchRelatedResources] Fetching services...')
      const servicesRes = await k8sResourceApi.getServices(props.clusterId, props.namespace)
      console.log('[fetchRelatedResources] Services response:', servicesRes)
      if (servicesRes.code === 0) {
        const allServices = servicesRes.data || []
        console.log('[fetchRelatedResources] All services count:', allServices.length)
        // 从 detail 中获取 Pod 标签
        const podLabels = detail.value?.spec?.template?.metadata?.labels || {}
        console.log('[fetchRelatedResources] Pod labels:', podLabels)
        
        // 筛选出 selector 匹配的 Services
        relatedServices.value = allServices.filter((svc: any) => {
          const selector = svc.selector || {}
          const selectorKeys = Object.keys(selector)
          
          // 排除没有 selector 的 Service
          if (selectorKeys.length === 0) return false
          
          // 排除系统 Service（如 kube-dns, kube-proxy 等）
          if (svc.name.startsWith('kube-')) return false
          
          // 检查 selector 的所有键是否都在 Pod 标签中且值匹配
          return selectorKeys.every(key => podLabels[key] === selector[key])
        })
        console.log('[fetchRelatedResources] Related services:', relatedServices.value)
      }
      
      // 获取关联的 Ingresses（通过 backend service 匹配）
      console.log('[fetchRelatedResources] Fetching ingresses...')
      const ingressesRes = await k8sResourceApi.getIngresses(props.clusterId, props.namespace)
      console.log('[fetchRelatedResources] Ingresses response:', ingressesRes)
      if (ingressesRes.code === 0) {
        const allIngresses = ingressesRes.data || []
        const serviceNames = relatedServices.value.map((s: any) => s.name)
        console.log('[fetchRelatedResources] Service names for matching:', serviceNames)
        
        // 筛选出指向关联 Service 的 Ingresses
        relatedIngresses.value = allIngresses.filter((ing: any) => {
          const rules = ing.rules || []
          return rules.some((rule: any) => {
            const paths = rule.paths || []
            return paths.some((path: any) => serviceNames.includes(path.service_name))
          })
        })
        console.log('[fetchRelatedResources] Related ingresses:', relatedIngresses.value)
      }
    }
    else if (props.resourceType === 'service') { const res = await k8sResourceApi.getServicePods(props.clusterId, props.namespace, props.name); if (res.code === 0) relatedPods.value = res.data || [] }
    else if (props.resourceType === 'node') { const res = await k8sNodeApi.getNodeDetail(props.clusterId, props.name); if (res.code === 0 && res.data?.pods) relatedPods.value = res.data.pods }
  } catch (e) {
    console.error('[fetchRelatedResources] Error:', e)
  }
  console.log('[fetchRelatedResources] Finished')
}

// HPA 相关
const hpaData = ref<any>(null)
const loadingHPA = ref(false)
const hpaModalVisible = ref(false)
const hpaModalMode = ref<'create' | 'edit'>('create')
const savingHPA = ref(false)
const hpaForm = ref({
  name: '',
  min_replicas: 1,
  max_replicas: 10,
  cpu_target_percent: 80 as number | undefined,
  mem_target_percent: undefined as number | undefined
})

const fetchHPA = async () => {
  if (!['deployment', 'statefulset'].includes(props.resourceType)) return
  loadingHPA.value = true
  try {
    const res = await k8sHPAApi.list(props.clusterId, props.namespace)
    if (res.code === 0) {
      // 查找关联到当前资源的 HPA
      const targetKind = props.resourceType === 'deployment' ? 'Deployment' : 'StatefulSet'
      hpaData.value = (res.data || []).find((h: any) => h.target_kind === targetKind && h.target_name === props.name) || null
    }
  } catch (e) {
    hpaData.value = null
  } finally {
    loadingHPA.value = false
  }
}

const showHPAModal = (mode: 'create' | 'edit') => {
  hpaModalMode.value = mode
  if (mode === 'create') {
    hpaForm.value = {
      name: props.name + '-hpa',
      min_replicas: 1,
      max_replicas: 10,
      cpu_target_percent: 80,
      mem_target_percent: undefined
    }
  } else if (hpaData.value) {
    // 从 metrics 解析 CPU 和内存目标
    let cpuTarget: number | undefined
    let memTarget: number | undefined
    for (const m of hpaData.value.metrics || []) {
      if (m.includes('CPU')) {
        const match = m.match(/(\d+)%/)
        if (match) cpuTarget = parseInt(match[1])
      }
      if (m.includes('Memory') || m.includes('内存')) {
        const match = m.match(/(\d+)%/)
        if (match) memTarget = parseInt(match[1])
      }
    }
    hpaForm.value = {
      name: hpaData.value.name,
      min_replicas: hpaData.value.min_replicas,
      max_replicas: hpaData.value.max_replicas,
      cpu_target_percent: cpuTarget,
      mem_target_percent: memTarget
    }
  }
  hpaModalVisible.value = true
}

const handleHPASubmit = async () => {
  if (!hpaForm.value.cpu_target_percent && !hpaForm.value.mem_target_percent) {
    message.error('请至少配置一个伸缩指标')
    return
  }
  savingHPA.value = true
  try {
    if (hpaModalMode.value === 'create') {
      const res = await k8sHPAApi.create(props.clusterId, {
        name: hpaForm.value.name,
        namespace: props.namespace,
        target_kind: props.resourceType === 'deployment' ? 'Deployment' : 'StatefulSet',
        target_name: props.name,
        min_replicas: hpaForm.value.min_replicas,
        max_replicas: hpaForm.value.max_replicas,
        cpu_target_percent: hpaForm.value.cpu_target_percent,
        mem_target_percent: hpaForm.value.mem_target_percent
      })
      if (res.code === 0) {
        message.success('HPA 创建成功')
        hpaModalVisible.value = false
        await fetchHPA()
      } else {
        message.error(res.message || '创建失败')
      }
    } else {
      const res = await k8sHPAApi.update(props.clusterId, props.namespace, hpaForm.value.name, {
        min_replicas: hpaForm.value.min_replicas,
        max_replicas: hpaForm.value.max_replicas,
        cpu_target_percent: hpaForm.value.cpu_target_percent,
        mem_target_percent: hpaForm.value.mem_target_percent
      })
      if (res.code === 0) {
        message.success('HPA 更新成功')
        hpaModalVisible.value = false
        await fetchHPA()
      } else {
        message.error(res.message || '更新失败')
      }
    }
  } catch (e: any) {
    message.error(e.message || '操作失败')
  } finally {
    savingHPA.value = false
  }
}

const deleteHPA = async () => {
  if (!hpaData.value) return
  try {
    const res = await k8sHPAApi.delete(props.clusterId, props.namespace, hpaData.value.name)
    if (res.code === 0) {
      message.success('HPA 删除成功')
      hpaData.value = null
    } else {
      message.error(res.message || '删除失败')
    }
  } catch (e: any) {
    message.error(e.message || '删除失败')
  }
}

// CronHPA 相关
const cronHPAData = ref<any>(null)
const loadingCronHPA = ref(false)
const cronHPAModalVisible = ref(false)
const cronHPAModalMode = ref<'create' | 'edit'>('create')
const savingCronHPA = ref(false)
const cronHPAForm = ref({
  name: '',
  enabled: true,
  schedules: [
    {
      name: '工作时间',
      cron: '0 9 * * 1-5',
      replicas: 3,
      min_replicas: undefined as number | undefined,
      max_replicas: undefined as number | undefined
    }
  ]
})

const fetchCronHPA = async () => {
  if (!['deployment', 'statefulset'].includes(props.resourceType)) return
  loadingCronHPA.value = true
  try {
    const res = await k8sCronHPAApi.list(props.clusterId, props.namespace)
    if (res.code === 0) {
      const targetKind = props.resourceType === 'deployment' ? 'Deployment' : 'StatefulSet'
      cronHPAData.value = (res.data || []).find((h: any) => h.target_kind === targetKind && h.target_name === props.name) || null
    }
  } catch (e) {
    cronHPAData.value = null
  } finally {
    loadingCronHPA.value = false
  }
}

const showCronHPAModal = (mode: 'create' | 'edit') => {
  cronHPAModalMode.value = mode
  if (mode === 'create') {
    cronHPAForm.value = {
      name: props.name + '-cron-hpa',
      enabled: true,
      schedules: [
        {
          name: '工作时间',
          cron: '0 9 * * 1-5',
          replicas: 3,
          min_replicas: undefined,
          max_replicas: undefined
        }
      ]
    }
  } else if (cronHPAData.value) {
    cronHPAForm.value = {
      name: cronHPAData.value.name,
      enabled: cronHPAData.value.enabled,
      schedules: JSON.parse(JSON.stringify(cronHPAData.value.schedules))
    }
  }
  cronHPAModalVisible.value = true
}

const addCronSchedule = () => {
  cronHPAForm.value.schedules.push({
    name: '',
    cron: '0 9 * * *',
    min_replicas: 1,
    max_replicas: 10
  })
}

const handleCronHPASubmit = async () => {
  if (!cronHPAForm.value.schedules.length) {
    message.error('请至少添加一个调度规则')
    return
  }
  
  for (const schedule of cronHPAForm.value.schedules) {
    if (!schedule.name || !schedule.cron || !schedule.min_replicas || !schedule.max_replicas) {
      message.error('请填写完整的调度规则信息')
      return
    }
  }
  
  savingCronHPA.value = true
  try {
    if (cronHPAModalMode.value === 'create') {
      const res = await k8sCronHPAApi.create(props.clusterId, {
        name: cronHPAForm.value.name,
        namespace: props.namespace,
        target_kind: props.resourceType === 'deployment' ? 'Deployment' : 'StatefulSet',
        target_name: props.name,
        enabled: cronHPAForm.value.enabled,
        schedules: cronHPAForm.value.schedules
      })
      if (res.code === 0) {
        message.success('定时伸缩创建成功')
        cronHPAModalVisible.value = false
        await fetchCronHPA()
      } else {
        message.error(res.message || '创建失败')
      }
    } else {
      const res = await k8sCronHPAApi.update(props.clusterId, props.namespace, cronHPAForm.value.name, {
        enabled: cronHPAForm.value.enabled,
        schedules: cronHPAForm.value.schedules
      })
      if (res.code === 0) {
        message.success('定时伸缩更新成功')
        cronHPAModalVisible.value = false
        await fetchCronHPA()
      } else {
        message.error(res.message || '更新失败')
      }
    }
  } catch (e: any) {
    message.error(e.message || '操作失败')
  } finally {
    savingCronHPA.value = false
  }
}

const deleteCronHPA = async () => {
  if (!cronHPAData.value) return
  try {
    const res = await k8sCronHPAApi.delete(props.clusterId, props.namespace, cronHPAData.value.name)
    if (res.code === 0) {
      message.success('定时伸缩删除成功')
      cronHPAData.value = null
    } else {
      message.error(res.message || '删除失败')
    }
  } catch (e: any) {
    message.error(e.message || '删除失败')
  }
}

const goToResource = (type: string, namespace: string, name: string) => emit('navigate', { type, namespace, name })
const goToNode = (nodeName: string) => { if (nodeName) emit('navigate', { type: 'node', namespace: '', name: nodeName }) }
const goToNamespace = (namespace: string) => emit('navigate', { type: 'namespace-filter', namespace, name: '' })

// Pod 操作相关
const currentPod = ref<any>(null)
const logsDrawerVisible = ref(false)
const terminalDrawerVisible = ref(false)

const showPodLogs = (pod: any) => {
  currentPod.value = pod
  logsDrawerVisible.value = true
}

const showPodTerminal = (pod: any) => {
  currentPod.value = pod
  terminalDrawerVisible.value = true
}

const deletePod = async (pod: any) => {
  try {
    const ns = pod.namespace || props.namespace
    const res = await k8sPodApi.delete(props.clusterId, ns, pod.name)
    if (res.code === 0) {
      message.success('删除成功')
      await fetchRelatedResources()
    } else {
      message.error(res.message || '删除失败')
    }
  } catch (e: any) {
    message.error(e.message || '删除失败')
  }
}
const goToOwner = (owner: { kind: string; name: string }) => {
  const typeMap: Record<string, string> = { Deployment: 'deployment', StatefulSet: 'statefulset', DaemonSet: 'daemonset', ReplicaSet: 'replicaset', Job: 'job', CronJob: 'cronjob' }
  if (typeMap[owner.kind]) emit('navigate', { type: typeMap[owner.kind], namespace: props.namespace, name: owner.name })
}
watch(() => [props.visible, props.name], () => { if (props.visible && props.name) { fetchDetail(); fetchHPA(); fetchCronHPA() } })

// 创建关联资源
const handleCreateRelated = async ({ key }: { key: string }) => {
  createRelatedType.value = key as 'service' | 'ingress'
  if (key === 'service') {
    // 从 YAML 解析 Pod 标签和容器端口
    try {
      const obj: any = jsYaml.load(yaml.value)
      const podLabels = obj?.spec?.template?.metadata?.labels || {}
      const containers = obj?.spec?.template?.spec?.containers || []
      const ports = containers.flatMap((c: any) => (c.ports || []).map((p: any) => ({ name: p.name || '', port: p.containerPort, targetPort: p.containerPort, protocol: p.protocol || 'TCP' })))
      relatedForm.value = {
        name: props.name + '-svc',
        serviceType: 'ClusterIP',
        selector: podLabels,
        ports: ports.length ? ports : [{ name: '', port: 80, targetPort: 80, protocol: 'TCP' }]
      }
    } catch { relatedForm.value = { name: props.name + '-svc', serviceType: 'ClusterIP', selector: {}, ports: [{ name: '', port: 80, targetPort: 80, protocol: 'TCP' }] } }
  } else {
    // 获取可用的 Service 列表
    try {
      const res = await k8sResourceApi.getServices(props.clusterId, props.namespace)
      if (res.code === 0) availableServices.value = res.data || []
    } catch { availableServices.value = [] }
    relatedForm.value = {
      name: props.name + '-ingress',
      ingressClassName: 'nginx',
      host: '',
      path: '/',
      serviceName: props.name + '-svc',
      servicePort: 80,
      enableTls: false,
      tlsSecretName: ''
    }
  }
  createRelatedVisible.value = true
}

const handleCreateRelatedSubmit = async () => {
  creatingRelated.value = true
  try {
    let yamlContent = ''
    if (createRelatedType.value === 'service') {
      const svc = {
        apiVersion: 'v1', kind: 'Service',
        metadata: { name: relatedForm.value.name, namespace: props.namespace },
        spec: {
          type: relatedForm.value.serviceType,
          selector: relatedForm.value.selector,
          ports: relatedForm.value.ports.map((p: any) => ({ name: p.name || undefined, port: p.port, targetPort: p.targetPort, protocol: p.protocol }))
        }
      }
      yamlContent = jsYaml.dump(svc, { lineWidth: -1 })
    } else {
      const ing: any = {
        apiVersion: 'networking.k8s.io/v1', kind: 'Ingress',
        metadata: { name: relatedForm.value.name, namespace: props.namespace },
        spec: {
          ingressClassName: relatedForm.value.ingressClassName || undefined,
          rules: [{ host: relatedForm.value.host || undefined, http: { paths: [{ path: relatedForm.value.path, pathType: 'Prefix', backend: { service: { name: relatedForm.value.serviceName, port: { number: relatedForm.value.servicePort } } } }] } }]
        }
      }
      if (relatedForm.value.enableTls && relatedForm.value.tlsSecretName) {
        ing.spec.tls = [{ secretName: relatedForm.value.tlsSecretName, hosts: relatedForm.value.host ? [relatedForm.value.host] : undefined }]
      }
      yamlContent = jsYaml.dump(ing, { lineWidth: -1 })
    }
    const res = await k8sResourceApi.applyResource(props.clusterId, yamlContent)
    if (res.code === 0) {
      message.success(`${createRelatedType.value === 'service' ? 'Service' : 'Ingress'} 创建成功`)
      createRelatedVisible.value = false
      emit('refresh')
    } else { message.error(res.message || '创建失败') }
  } catch (e: any) { message.error(e.message || '创建失败') }
  finally { creatingRelated.value = false }
}

// 辅助函数
const objToKvArray = (obj: Record<string, string> | undefined) => obj ? Object.entries(obj).map(([key, value]) => ({ key, value })) : []
const kvArrayToObj = (arr: { key: string; value: string }[]) => arr.filter(i => i.key).reduce((acc, i) => ({ ...acc, [i.key]: i.value }), {} as Record<string, string>)

const defaultProbe = () => ({ type: '', path: '', port: 80, command: '', scheme: 'HTTP', initialDelaySeconds: 0, periodSeconds: 10, timeoutSeconds: 1, successThreshold: 1, failureThreshold: 3 })

const parseProbe = (probe: any) => {
  if (!probe) return defaultProbe()
  const base = {
    initialDelaySeconds: probe.initialDelaySeconds || 0,
    periodSeconds: probe.periodSeconds || 10,
    timeoutSeconds: probe.timeoutSeconds || 1,
    successThreshold: probe.successThreshold || 1,
    failureThreshold: probe.failureThreshold || 3
  }
  if (probe.httpGet) return { ...base, type: 'httpGet', path: probe.httpGet.path || '/', port: probe.httpGet.port || 80, scheme: probe.httpGet.scheme || 'HTTP', command: '' }
  if (probe.tcpSocket) return { ...base, type: 'tcpSocket', path: '', port: probe.tcpSocket.port || 80, scheme: 'HTTP', command: '' }
  if (probe.exec) return { ...base, type: 'exec', path: '', port: 80, scheme: 'HTTP', command: JSON.stringify(probe.exec.command || []) }
  return defaultProbe()
}

const buildProbe = (probe: any) => {
  if (!probe.type) return undefined
  const base = {
    initialDelaySeconds: probe.initialDelaySeconds || 0,
    periodSeconds: probe.periodSeconds || 10,
    timeoutSeconds: probe.timeoutSeconds || 1,
    successThreshold: probe.successThreshold || 1,
    failureThreshold: probe.failureThreshold || 3
  }
  if (probe.type === 'httpGet') return { ...base, httpGet: { path: probe.path || '/', port: probe.port || 80, scheme: probe.scheme || 'HTTP' } }
  if (probe.type === 'tcpSocket') return { ...base, tcpSocket: { port: probe.port || 80 } }
  if (probe.type === 'exec') { try { return { ...base, exec: { command: JSON.parse(probe.command) } } } catch { return undefined } }
  return undefined
}

const parseYamlToForm = () => {
  try {
    const obj: any = jsYaml.load(yaml.value)
    if (!obj) return
    switch (props.resourceType) {
      case 'deployment': case 'statefulset': case 'daemonset': {
        const spec = obj.spec || {}; const podSpec = spec.template?.spec || {}; const containers = podSpec.containers || []
        const strategy = spec.strategy || spec.updateStrategy || {}
        formData.value = {
          replicas: spec.replicas || 1, strategy: strategy.type || 'RollingUpdate',
          maxUnavailable: strategy.rollingUpdate?.maxUnavailable?.toString() || '25%', maxSurge: strategy.rollingUpdate?.maxSurge?.toString() || '25%',
          podLabels: objToKvArray(spec.template?.metadata?.labels), podAnnotations: objToKvArray(spec.template?.metadata?.annotations),
          nodeSelector: objToKvArray(podSpec.nodeSelector), serviceAccountName: podSpec.serviceAccountName || '', restartPolicy: podSpec.restartPolicy || 'Always',
          containers: containers.map((c: any) => ({
            name: c.name, image: c.image, imagePullPolicy: c.imagePullPolicy || 'IfNotPresent',
            command: c.command ? JSON.stringify(c.command) : '', args: c.args ? JSON.stringify(c.args) : '',
            cpuRequest: c.resources?.requests?.cpu || '', cpuLimit: c.resources?.limits?.cpu || '',
            memoryRequest: c.resources?.requests?.memory || '', memoryLimit: c.resources?.limits?.memory || '',
            ports: (c.ports || []).map((p: any) => ({ name: p.name || '', containerPort: p.containerPort, protocol: p.protocol || 'TCP' })),
            env: (c.env || []).filter((e: any) => e.value !== undefined).map((e: any) => ({ name: e.name, value: e.value || '' })),
            volumeMounts: (c.volumeMounts || []).map((m: any) => ({ name: m.name, mountPath: m.mountPath, subPath: m.subPath || '', readOnly: m.readOnly || false })),
            livenessProbe: parseProbe(c.livenessProbe),
            readinessProbe: parseProbe(c.readinessProbe),
            startupProbe: parseProbe(c.startupProbe)
          })),
          volumes: (podSpec.volumes || []).map((v: any) => {
            if (v.emptyDir !== undefined) return { name: v.name, type: 'emptyDir' }
            if (v.configMap) return { name: v.name, type: 'configMap', configMapName: v.configMap.name }
            if (v.secret) return { name: v.name, type: 'secret', secretName: v.secret.secretName }
            if (v.persistentVolumeClaim) return { name: v.name, type: 'pvc', pvcName: v.persistentVolumeClaim.claimName }
            if (v.hostPath) return { name: v.name, type: 'hostPath', hostPath: v.hostPath.path }
            return { name: v.name, type: 'emptyDir' }
          })
        }
        if (formData.value.containers.length === 0) formData.value.containers = [{ name: 'main', image: '', imagePullPolicy: 'IfNotPresent', command: '', args: '', cpuRequest: '', cpuLimit: '', memoryRequest: '', memoryLimit: '', ports: [], env: [], volumeMounts: [], livenessProbe: defaultProbe(), readinessProbe: defaultProbe(), startupProbe: defaultProbe() }]
        break
      }
      case 'service': {
        const spec = obj.spec || {}
        formData.value = { type: spec.type || 'ClusterIP', sessionAffinity: spec.sessionAffinity || 'None', externalTrafficPolicy: spec.externalTrafficPolicy || 'Cluster', externalName: spec.externalName || '',
          ports: (spec.ports || []).map((p: any) => ({ name: p.name || '', port: p.port, targetPort: String(p.targetPort || p.port), protocol: p.protocol || 'TCP', nodePort: p.nodePort })),
          selector: objToKvArray(spec.selector) }
        if (formData.value.ports.length === 0) formData.value.ports = [{ name: '', port: 80, targetPort: '80', protocol: 'TCP' }]
        break
      }
      case 'configmap': case 'secret': {
        const isSecret = props.resourceType === 'secret'
        formData.value = { labels: objToKvArray(obj.metadata?.labels), annotations: objToKvArray(obj.metadata?.annotations),
          data: Object.entries(obj.data || {}).map(([key, value]) => ({ key, value: isSecret ? atob(value as string) : value as string })) }
        if (formData.value.data.length === 0) formData.value.data = [{ key: '', value: '' }]
        break
      }
      case 'ingress': {
        const spec = obj.spec || {}
        formData.value = { ingressClassName: spec.ingressClassName || '', annotations: objToKvArray(obj.metadata?.annotations),
          tls: (spec.tls || []).map((t: any) => ({ secretName: t.secretName || '', hosts: (t.hosts || []).join(',') })),
          rules: (spec.rules || []).map((r: any) => ({ host: r.host || '', paths: (r.http?.paths || []).map((p: any) => ({ path: p.path || '/', pathType: p.pathType || 'Prefix', serviceName: p.backend?.service?.name || '', servicePort: p.backend?.service?.port?.number || 80 })) })) }
        if (formData.value.rules.length === 0) formData.value.rules = [{ host: '', paths: [{ path: '/', pathType: 'Prefix', serviceName: '', servicePort: 80 }] }]
        break
      }
      case 'pvc': {
        const spec = obj.spec || {}
        formData.value = { storageClassName: spec.storageClassName || '', storage: spec.resources?.requests?.storage || '1Gi', accessModes: spec.accessModes || ['ReadWriteOnce'] }
        break
      }
    }
  } catch (e) { console.error('解析 YAML 失败', e) }
}

const buildYamlFromForm = (): string => {
  try {
    const obj: any = jsYaml.load(yaml.value); if (!obj) return yaml.value
    switch (props.resourceType) {
      case 'deployment': case 'statefulset': case 'daemonset': {
        const fd = formData.value
        if (props.resourceType !== 'daemonset') obj.spec.replicas = fd.replicas
        const strategyKey = props.resourceType === 'deployment' ? 'strategy' : 'updateStrategy'
        obj.spec[strategyKey] = obj.spec[strategyKey] || {}; obj.spec[strategyKey].type = fd.strategy
        if (fd.strategy === 'RollingUpdate') { obj.spec[strategyKey].rollingUpdate = { maxUnavailable: fd.maxUnavailable }; if (props.resourceType === 'deployment') obj.spec[strategyKey].rollingUpdate.maxSurge = fd.maxSurge }
        else { delete obj.spec[strategyKey].rollingUpdate }
        obj.spec.template.metadata = obj.spec.template.metadata || {}
        const podLabels = kvArrayToObj(fd.podLabels); if (Object.keys(podLabels).length) obj.spec.template.metadata.labels = { ...obj.spec.template.metadata.labels, ...podLabels }
        const podAnnos = kvArrayToObj(fd.podAnnotations); if (Object.keys(podAnnos).length) obj.spec.template.metadata.annotations = podAnnos
        const podSpec = obj.spec.template.spec
        const nodeSelector = kvArrayToObj(fd.nodeSelector); podSpec.nodeSelector = Object.keys(nodeSelector).length ? nodeSelector : undefined
        if (fd.serviceAccountName) podSpec.serviceAccountName = fd.serviceAccountName
        if (fd.restartPolicy && fd.restartPolicy !== 'Always') podSpec.restartPolicy = fd.restartPolicy
        fd.containers.forEach((fc: any, idx: number) => {
          if (!podSpec.containers[idx]) return; const c = podSpec.containers[idx]
          c.image = fc.image; c.imagePullPolicy = fc.imagePullPolicy
          if (fc.command) { try { c.command = JSON.parse(fc.command) } catch {} } else { delete c.command }
          if (fc.args) { try { c.args = JSON.parse(fc.args) } catch {} } else { delete c.args }
          if (fc.cpuRequest || fc.cpuLimit || fc.memoryRequest || fc.memoryLimit) {
            c.resources = c.resources || {}
            if (fc.cpuRequest || fc.memoryRequest) { c.resources.requests = {}; if (fc.cpuRequest) c.resources.requests.cpu = fc.cpuRequest; if (fc.memoryRequest) c.resources.requests.memory = fc.memoryRequest }
            if (fc.cpuLimit || fc.memoryLimit) { c.resources.limits = {}; if (fc.cpuLimit) c.resources.limits.cpu = fc.cpuLimit; if (fc.memoryLimit) c.resources.limits.memory = fc.memoryLimit }
          }
          if (fc.ports?.length) c.ports = fc.ports.filter((p: any) => p.containerPort).map((p: any) => ({ name: p.name || undefined, containerPort: p.containerPort, protocol: p.protocol }))
          if (fc.env?.length) c.env = fc.env.filter((e: any) => e.name).map((e: any) => ({ name: e.name, value: e.value }))
          if (fc.volumeMounts?.length) c.volumeMounts = fc.volumeMounts.filter((m: any) => m.name && m.mountPath).map((m: any) => ({ name: m.name, mountPath: m.mountPath, subPath: m.subPath || undefined, readOnly: m.readOnly || undefined }))
          c.livenessProbe = buildProbe(fc.livenessProbe)
          c.readinessProbe = buildProbe(fc.readinessProbe)
          c.startupProbe = buildProbe(fc.startupProbe)
        })
        if (fd.volumes?.length) { podSpec.volumes = fd.volumes.filter((v: any) => v.name).map((v: any) => {
          const vol: any = { name: v.name }
          if (v.type === 'emptyDir') vol.emptyDir = {}; else if (v.type === 'configMap') vol.configMap = { name: v.configMapName }
          else if (v.type === 'secret') vol.secret = { secretName: v.secretName }; else if (v.type === 'pvc') vol.persistentVolumeClaim = { claimName: v.pvcName }
          else if (v.type === 'hostPath') vol.hostPath = { path: v.hostPath }; return vol
        }) }
        break
      }
      case 'service': {
        const fd = formData.value; obj.spec.type = fd.type; obj.spec.sessionAffinity = fd.sessionAffinity
        if (fd.type === 'NodePort' || fd.type === 'LoadBalancer') obj.spec.externalTrafficPolicy = fd.externalTrafficPolicy
        if (fd.type === 'ExternalName') obj.spec.externalName = fd.externalName
        obj.spec.ports = fd.ports.map((p: any) => ({ name: p.name || undefined, port: p.port, targetPort: isNaN(Number(p.targetPort)) ? p.targetPort : Number(p.targetPort), protocol: p.protocol, nodePort: p.nodePort || undefined }))
        const selector = kvArrayToObj(fd.selector); if (Object.keys(selector).length) obj.spec.selector = selector
        break
      }
      case 'configmap': case 'secret': {
        const fd = formData.value; const isSecret = props.resourceType === 'secret'
        const labels = kvArrayToObj(fd.labels); if (Object.keys(labels).length) obj.metadata.labels = labels
        const annos = kvArrayToObj(fd.annotations); if (Object.keys(annos).length) obj.metadata.annotations = annos
        obj.data = {}; fd.data.forEach((item: any) => { if (item.key) obj.data[item.key] = isSecret ? btoa(item.value) : item.value })
        break
      }
      case 'ingress': {
        const fd = formData.value; obj.spec.ingressClassName = fd.ingressClassName || undefined
        const annos = kvArrayToObj(fd.annotations); if (Object.keys(annos).length) obj.metadata.annotations = annos
        if (fd.tls?.length) obj.spec.tls = fd.tls.filter((t: any) => t.secretName).map((t: any) => ({ secretName: t.secretName, hosts: t.hosts ? t.hosts.split(',').map((h: string) => h.trim()) : undefined }))
        obj.spec.rules = fd.rules.map((r: any) => ({ host: r.host || undefined, http: { paths: r.paths.map((p: any) => ({ path: p.path, pathType: p.pathType, backend: { service: { name: p.serviceName, port: { number: p.servicePort } } } })) } }))
        break
      }
      case 'pvc': { const fd = formData.value; obj.spec.storageClassName = fd.storageClassName || undefined; obj.spec.resources = { requests: { storage: fd.storage } }; obj.spec.accessModes = fd.accessModes; break }
    }
    return jsYaml.dump(obj, { lineWidth: -1 })
  } catch (e) { console.error('构建 YAML 失败', e); return yaml.value }
}

const handleEdit = () => { editYaml.value = yaml.value; parseYamlToForm(); editMode.value = canEditByForm.value ? 'form' : 'yaml'; formActiveKeys.value = ['basic', 'containers']; activeContainerIdx.value = 0; editModalVisible.value = true }
const handleSave = async () => {
  let yamlToApply = editMode.value === 'form' && canEditByForm.value ? buildYamlFromForm() : editYaml.value
  if (!yamlToApply.trim()) { message.error('YAML 内容不能为空'); return }
  if (!yamlToApply.includes('apiVersion:') || !yamlToApply.includes('kind:')) { message.error('YAML 必须包含 apiVersion 和 kind 字段'); return }
  saving.value = true
  try { const res = await k8sResourceApi.applyResource(props.clusterId, yamlToApply); if (res.code === 0) { message.success('保存成功'); editModalVisible.value = false; await fetchDetail(); emit('refresh') } else { message.error(res.message || '保存失败') } }
  catch (e: any) { message.error(e.message || '保存失败') } finally { saving.value = false }
}
</script>

<style scoped>
.detail-card { margin-bottom: 12px; }
.detail-card :deep(.ant-card-head) { min-height: 40px; padding: 0 12px; }
.detail-card :deep(.ant-card-head-title) { padding: 8px 0; font-size: 14px; }
.detail-card :deep(.ant-card-body) { padding: 12px; }
.tags-container { display: flex; flex-wrap: wrap; gap: 4px; }
.annotation-item { margin-bottom: 6px; font-size: 12px; line-height: 1.5; }
.anno-key { color: #1890ff; font-weight: 500; }
.related-section { margin-bottom: 12px; }
.section-title { font-weight: 500; margin-bottom: 8px; color: #333; }
.link { color: #1890ff; cursor: pointer; }
.link:hover { text-decoration: underline; }
.ellipsis-text { max-width: 280px; display: inline-block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.yaml-content { background: #1e1e1e; color: #d4d4d4; padding: 12px; border-radius: 4px; max-height: 350px; overflow: auto; font-size: 12px; font-family: 'Consolas', 'Monaco', monospace; white-space: pre-wrap; word-break: break-all; margin-top: 8px; }
.edit-form { max-height: 520px; overflow-y: auto; }
.edit-form :deep(.ant-collapse-header) { padding: 8px 12px !important; font-weight: 500; }
.edit-form :deep(.ant-collapse-content-box) { padding: 12px !important; }
.edit-form :deep(.ant-form-item) { margin-bottom: 12px; }
.edit-form :deep(.ant-form-item-label) { padding-bottom: 2px; }
.form-section-title { font-size: 13px; font-weight: 500; color: #666; margin: 12px 0 8px; padding-bottom: 4px; border-bottom: 1px dashed #e8e8e8; }
.kv-row { display: flex; align-items: center; gap: 8px; margin-bottom: 6px; }
.kv-input { flex: 1; }
.inline-row { display: flex; align-items: center; gap: 6px; margin-bottom: 6px; flex-wrap: wrap; }
.volume-row { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; flex-wrap: wrap; }
.data-item { background: #fafafa; padding: 10px; margin-bottom: 8px; border-radius: 4px; }
.data-item-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; }
.rule-section { background: #fafafa; padding: 10px; margin-bottom: 8px; border-radius: 4px; }
.rule-header { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; }
.probe-section { background: #f8f8f8; padding: 10px; margin-bottom: 8px; border-radius: 4px; border: 1px solid #e8e8e8; }
.probe-header { display: flex; align-items: center; justify-content: space-between; font-weight: 500; color: #333; }
.probe-config { margin-top: 10px; }
.probe-row { display: flex; align-items: center; flex-wrap: wrap; gap: 8px; margin-bottom: 8px; }
.probe-label { color: #666; font-size: 13px; white-space: nowrap; }
.probe-unit { color: #999; font-size: 12px; margin-right: 8px; }
.yaml-editor { font-family: 'Consolas', 'Monaco', monospace; font-size: 12px; }
.modal-footer { margin-top: 16px; text-align: right; display: flex; justify-content: flex-end; gap: 8px; }
.selector-preview { margin-bottom: 8px; }
.hpa-actions { margin-top: 12px; display: flex; gap: 8px; justify-content: flex-end; }
.cron-schedule-item { margin-bottom: 12px; }
.cron-schedule-item :deep(.ant-card-head) { min-height: 36px; padding: 0 12px; }
.cron-schedule-item :deep(.ant-card-head-title) { padding: 6px 0; font-size: 13px; }
.cron-schedule-item :deep(.ant-card-body) { padding: 12px; }
</style>