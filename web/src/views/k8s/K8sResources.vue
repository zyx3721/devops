<template>
  <div class="k8s-resources">
    <!-- 顶部导航 -->
    <div class="top-header">
      <a-button @click="goBack"><ArrowLeftOutlined /></a-button>
      <span class="cluster-name">{{ clusterName }}</span>
      <a-select v-model:value="selectedNamespace" style="width: 200px; margin-left: 16px" :loading="loadingNs" v-if="showNamespaceSelect">
        <a-select-option value="">全部命名空间</a-select-option>
        <a-select-option v-for="ns in namespaces" :key="ns.name" :value="ns.name">{{ ns.name }}</a-select-option>
      </a-select>
      <a-input-search v-model:value="searchKeyword" placeholder="搜索名称..." style="width: 200px; margin-left: 12px" allow-clear @search="onSearch" @change="onSearchChange" />
      <a-button style="margin-left: 12px" @click="refreshData" :loading="loading"><ReloadOutlined /></a-button>
      <a-button type="primary" style="margin-left: 12px" @click="showCreateModal" v-if="selectedNamespace && canCreateResource"><PlusOutlined /> 创建{{ currentResourceLabel }}</a-button>
      <a-button type="primary" style="margin-left: 12px" @click="showJoinNodeModal" v-if="selectedMenu[0] === 'nodes'"><PlusOutlined /> 添加节点</a-button>
      <a-button type="primary" style="margin-left: 12px" @click="showCreateNamespaceModal" v-if="selectedMenu[0] === 'namespaces'"><PlusOutlined /> 创建命名空间</a-button>
    </div>

    <div class="main-content">
      <!-- 左侧菜单 -->
      <div class="left-menu">
        <a-menu v-model:selectedKeys="selectedMenu" mode="inline">
          <a-menu-item-group title="集群">
            <a-menu-item key="namespaces"><BlockOutlined /> 命名空间</a-menu-item>
            <a-menu-item key="nodes"><CloudServerOutlined /> 节点</a-menu-item>
            <a-menu-item key="events"><BellOutlined /> 事件</a-menu-item>
          </a-menu-item-group>
          <a-menu-item-group title="工作负载">
            <a-menu-item key="deployments"><DeploymentUnitOutlined /> Deployments</a-menu-item>
            <a-menu-item key="statefulsets"><DatabaseOutlined /> StatefulSets</a-menu-item>
            <a-menu-item key="daemonsets"><ClusterOutlined /> DaemonSets</a-menu-item>
            <a-menu-item key="jobs"><ThunderboltOutlined /> Jobs</a-menu-item>
            <a-menu-item key="cronjobs"><FieldTimeOutlined /> CronJobs</a-menu-item>
            <a-menu-item key="pods"><AppstoreOutlined /> Pods</a-menu-item>
            <a-menu-item key="hpa"><DashboardOutlined /> HPA</a-menu-item>
          </a-menu-item-group>
          <a-menu-item-group title="服务与路由">
            <a-menu-item key="services"><ApiOutlined /> Services</a-menu-item>
            <a-menu-item key="ingresses"><GlobalOutlined /> Ingresses</a-menu-item>
          </a-menu-item-group>
          <a-menu-item-group title="配置管理">
            <a-menu-item key="configmaps"><FileTextOutlined /> ConfigMaps</a-menu-item>
            <a-menu-item key="secrets"><LockOutlined /> Secrets</a-menu-item>
          </a-menu-item-group>
          <a-menu-item-group title="存储">
            <a-menu-item key="pvcs"><HddOutlined /> PVCs</a-menu-item>
            <a-menu-item key="pvs"><InboxOutlined /> PVs</a-menu-item>
            <a-menu-item key="storageclasses"><FolderOutlined /> StorageClasses</a-menu-item>
          </a-menu-item-group>
        </a-menu>
      </div>

      <!-- 右侧内容 -->
      <div class="right-content">
        <!-- Namespaces -->
        <div v-if="selectedMenu[0] === 'namespaces'">
          <div class="content-header"><h3>命名空间</h3></div>
          <a-table :columns="namespaceColumns" :data-source="filteredNamespaces" :loading="loading" row-key="name" size="small">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'status'">
                <a-tag :color="record.status === 'Active' ? 'green' : 'orange'">{{ record.status }}</a-tag>
              </template>
              <template v-if="column.key === 'action'">
                <a-space>
                  <a @click="showYAMLModal(record, 'namespace')">YAML</a>
                  <a-popconfirm title="确定删除该命名空间？删除后其中的所有资源都将被删除！" @confirm="deleteNamespace(record)" :disabled="isSystemNamespace(record.name)">
                    <a :style="{ color: isSystemNamespace(record.name) ? '#ccc' : '#ff4d4f', cursor: isSystemNamespace(record.name) ? 'not-allowed' : 'pointer' }">删除</a>
                  </a-popconfirm>
                </a-space>
              </template>
            </template>
          </a-table>
        </div>

        <!-- Deployments -->
        <div v-if="selectedMenu[0] === 'deployments'">
          <div class="content-header"><h3>Deployments</h3></div>
          <a-table :columns="deploymentColumns" :data-source="filteredDeployments" :loading="loading" row-key="name" size="small">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'name'">
                <a class="resource-link" @click="showResourceDetail('deployment', record.namespace, record.name)">{{ record.name }}</a>
              </template>
              <template v-if="column.key === 'replicas'">
                <a-tag :color="record.ready === record.replicas ? 'green' : 'orange'">{{ record.ready }}/{{ record.replicas }}</a-tag>
              </template>
              <template v-if="column.key === 'images'">
                <a-tooltip v-for="(img, i) in record.images" :key="i" :title="img">
                  <a-tag style="max-width: 200px; overflow: hidden; text-overflow: ellipsis">{{ img.split('/').pop() }}</a-tag>
                </a-tooltip>
              </template>
              <template v-if="column.key === 'action'">
                <a-space :size="4">
                  <a @click="showYAMLModal(record, 'deployment')">YAML</a>
                  <a-dropdown>
                    <a>更多 <DownOutlined /></a>
                    <template #overlay>
                      <a-menu>
                        <a-menu-item @click="showScaleModal(record)"><ExpandOutlined /> 伸缩</a-menu-item>
                        <a-menu-item @click="showImageUpdateModal(record)"><EditOutlined /> 更新镜像</a-menu-item>
                        <a-menu-item @click="showRollbackModal(record)"><RollbackOutlined /> 回滚</a-menu-item>
                        <a-menu-item @click="restartDeployment(record)"><SyncOutlined /> 重启</a-menu-item>
                        <a-menu-divider />
                        <a-menu-item @click="confirmDeleteDeployment(record)" danger><DeleteOutlined /> 删除</a-menu-item>
                      </a-menu>
                    </template>
                  </a-dropdown>
                </a-space>
              </template>
            </template>
          </a-table>
        </div>

        <!-- StatefulSets -->
        <div v-if="selectedMenu[0] === 'statefulsets'">
          <div class="content-header"><h3>StatefulSets</h3></div>
          <a-table :columns="statefulSetColumns" :data-source="filteredStatefulSets" :loading="loading" row-key="name" size="small">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'name'">
                <a class="resource-link" @click="showResourceDetail('statefulset', record.namespace, record.name)">{{ record.name }}</a>
              </template>
              <template v-if="column.key === 'replicas'">
                <a-tag :color="record.ready === record.replicas ? 'green' : 'orange'">{{ record.ready }}/{{ record.replicas }}</a-tag>
              </template>
              <template v-if="column.key === 'action'">
                <a-space>
                  <a @click="showYAMLModal(record, 'statefulset')">YAML</a>
                  <a-popconfirm title="确定删除？" @confirm="deleteResource(record, 'statefulset')"><a style="color: #ff4d4f">删除</a></a-popconfirm>
                </a-space>
              </template>
            </template>
          </a-table>
        </div>

        <!-- DaemonSets -->
        <div v-if="selectedMenu[0] === 'daemonsets'">
          <div class="content-header"><h3>DaemonSets</h3></div>
          <a-table :columns="daemonSetColumns" :data-source="filteredDaemonSets" :loading="loading" row-key="name" size="small">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'name'">
                <a class="resource-link" @click="showResourceDetail('daemonset', record.namespace, record.name)">{{ record.name }}</a>
              </template>
              <template v-if="column.key === 'status'">
                <a-tag :color="record.ready === record.desired ? 'green' : 'orange'">{{ record.ready }}/{{ record.desired }}</a-tag>
              </template>
              <template v-if="column.key === 'action'">
                <a-space>
                  <a @click="showYAMLModal(record, 'daemonset')">YAML</a>
                  <a-popconfirm title="确定删除？" @confirm="deleteResource(record, 'daemonset')"><a style="color: #ff4d4f">删除</a></a-popconfirm>
                </a-space>
              </template>
            </template>
          </a-table>
        </div>

        <!-- Jobs -->
        <div v-if="selectedMenu[0] === 'jobs'">
          <div class="content-header"><h3>Jobs</h3></div>
          <a-table :columns="jobColumns" :data-source="filteredJobs" :loading="loading" row-key="name" size="small">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'name'">
                <a class="resource-link" @click="showResourceDetail('job', record.namespace, record.name)">{{ record.name }}</a>
              </template>
              <template v-if="column.key === 'status'">
                <a-tag :color="record.succeeded > 0 ? 'green' : record.failed > 0 ? 'red' : 'blue'">
                  {{ record.succeeded > 0 ? '完成' : record.failed > 0 ? '失败' : '运行中' }}
                </a-tag>
              </template>
              <template v-if="column.key === 'action'">
                <a-space>
                  <a @click="showYAMLModal(record, 'job')">YAML</a>
                  <a-popconfirm title="确定删除？" @confirm="deleteResource(record, 'job')"><a style="color: #ff4d4f">删除</a></a-popconfirm>
                </a-space>
              </template>
            </template>
          </a-table>
        </div>

        <!-- CronJobs -->
        <div v-if="selectedMenu[0] === 'cronjobs'">
          <div class="content-header"><h3>CronJobs</h3></div>
          <a-table :columns="cronJobColumns" :data-source="filteredCronJobs" :loading="loading" row-key="name" size="small">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'name'">
                <a class="resource-link" @click="showResourceDetail('cronjob', record.namespace, record.name)">{{ record.name }}</a>
              </template>
              <template v-if="column.key === 'suspend'">
                <a-tag :color="record.suspend ? 'red' : 'green'">{{ record.suspend ? '已暂停' : '运行中' }}</a-tag>
              </template>
              <template v-if="column.key === 'action'">
                <a-space>
                  <a @click="showYAMLModal(record, 'cronjob')">YAML</a>
                  <a-popconfirm title="确定删除？" @confirm="deleteResource(record, 'cronjob')"><a style="color: #ff4d4f">删除</a></a-popconfirm>
                </a-space>
              </template>
            </template>
          </a-table>
        </div>

        <!-- Pods -->
        <div v-if="selectedMenu[0] === 'pods'">
          <div class="content-header"><h3>Pods</h3></div>
          <a-table :columns="podColumns" :data-source="filteredPods" :loading="loading" row-key="name" size="small">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'name'">
                <a class="resource-link" @click="showResourceDetail('pod', record.namespace, record.name)">{{ record.name }}</a>
              </template>
              <template v-if="column.key === 'node'">
                <a class="resource-link" @click="showResourceDetail('node', '', record.node)">{{ record.node }}</a>
              </template>
              <template v-if="column.key === 'status'">
                <a-tag :color="getPodStatusColor(record.status)">{{ record.status }}</a-tag>
              </template>
              <template v-if="column.key === 'action'">
                <a-space>
                  <a @click="showLogs(record)">日志</a>
                  <a @click="openTerminal(record)">终端</a>
                  <a-popconfirm title="确定删除？" @confirm="deletePod(record)"><a style="color: #ff4d4f">删除</a></a-popconfirm>
                </a-space>
              </template>
            </template>
          </a-table>
        </div>

        <!-- Services -->
        <div v-if="selectedMenu[0] === 'services'">
          <div class="content-header"><h3>Services</h3></div>
          <a-table :columns="serviceColumns" :data-source="filteredServices" :loading="loading" row-key="name" size="small">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'name'">
                <a class="resource-link" @click="showResourceDetail('service', record.namespace, record.name)">{{ record.name }}</a>
              </template>
              <template v-if="column.key === 'type'"><a-tag>{{ record.type }}</a-tag></template>
              <template v-if="column.key === 'ports'">
                <span v-for="(p, i) in record.ports" :key="i">
                  {{ p.port }}:{{ p.target_port }}/{{ p.protocol }}{{ p.node_port ? ` (${p.node_port})` : '' }}
                  <br v-if="i < record.ports.length - 1" />
                </span>
              </template>
              <template v-if="column.key === 'action'">
                <a-space>
                  <a @click="showYAMLModal(record, 'service')">YAML</a>
                  <a-popconfirm title="确定删除？" @confirm="deleteResource(record, 'service')"><a style="color: #ff4d4f">删除</a></a-popconfirm>
                </a-space>
              </template>
            </template>
          </a-table>
        </div>

        <!-- Ingresses -->
        <div v-if="selectedMenu[0] === 'ingresses'">
          <div class="content-header"><h3>Ingresses</h3></div>
          <a-table :columns="ingressColumns" :data-source="filteredIngresses" :loading="loading" row-key="name" size="small">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'name'">
                <a class="resource-link" @click="showResourceDetail('ingress', record.namespace, record.name)">{{ record.name }}</a>
              </template>
              <template v-if="column.key === 'hosts'">
                <a-tag v-for="h in record.hosts" :key="h">{{ h }}</a-tag>
              </template>
              <template v-if="column.key === 'action'">
                <a-space>
                  <a @click="showYAMLModal(record, 'ingress')">YAML</a>
                  <a-popconfirm title="确定删除？" @confirm="deleteResource(record, 'ingress')"><a style="color: #ff4d4f">删除</a></a-popconfirm>
                </a-space>
              </template>
            </template>
          </a-table>
        </div>

        <!-- ConfigMaps -->
        <div v-if="selectedMenu[0] === 'configmaps'">
          <div class="content-header"><h3>ConfigMaps</h3></div>
          <a-table :columns="configMapColumns" :data-source="filteredConfigMaps" :loading="loading" row-key="name" size="small">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'name'">
                <a class="resource-link" @click="showResourceDetail('configmap', record.namespace, record.name)">{{ record.name }}</a>
              </template>
              <template v-if="column.key === 'keys'">
                <a-tag v-for="k in record.keys.slice(0, 3)" :key="k" style="margin: 2px">{{ k }}</a-tag>
                <span v-if="record.keys.length > 3" class="more-tag">+{{ record.keys.length - 3 }}</span>
              </template>
              <template v-if="column.key === 'action'">
                <a-space>
                  <a @click="showYAMLModal(record, 'configmap')">YAML</a>
                  <a-popconfirm title="确定删除？" @confirm="deleteResource(record, 'configmap')"><a style="color: #ff4d4f">删除</a></a-popconfirm>
                </a-space>
              </template>
            </template>
          </a-table>
        </div>

        <!-- Secrets -->
        <div v-if="selectedMenu[0] === 'secrets'">
          <div class="content-header"><h3>Secrets</h3></div>
          <a-table :columns="secretColumns" :data-source="filteredSecrets" :loading="loading" row-key="name" size="small">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'name'">
                <a class="resource-link" @click="showResourceDetail('secret', record.namespace, record.name)">{{ record.name }}</a>
              </template>
              <template v-if="column.key === 'keys'">
                <a-tag v-for="k in record.keys.slice(0, 3)" :key="k" style="margin: 2px">{{ k }}</a-tag>
                <span v-if="record.keys.length > 3" class="more-tag">+{{ record.keys.length - 3 }}</span>
              </template>
              <template v-if="column.key === 'action'">
                <a-space>
                  <a @click="showYAMLModal(record, 'secret')">YAML</a>
                  <a-popconfirm title="确定删除？" @confirm="deleteResource(record, 'secret')"><a style="color: #ff4d4f">删除</a></a-popconfirm>
                </a-space>
              </template>
            </template>
          </a-table>
        </div>

        <!-- PVCs -->
        <div v-if="selectedMenu[0] === 'pvcs'">
          <div class="content-header"><h3>PersistentVolumeClaims</h3></div>
          <a-table :columns="pvcColumns" :data-source="filteredPvcs" :loading="loading" row-key="name" size="small">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'name'">
                <a class="resource-link" @click="showResourceDetail('pvc', record.namespace, record.name)">{{ record.name }}</a>
              </template>
              <template v-if="column.key === 'status'">
                <a-tag :color="record.status === 'Bound' ? 'green' : 'orange'">{{ record.status }}</a-tag>
              </template>
              <template v-if="column.key === 'action'">
                <a-space>
                  <a @click="showYAMLModal(record, 'pvc')">YAML</a>
                  <a-popconfirm title="确定删除？" @confirm="deleteResource(record, 'pvc')"><a style="color: #ff4d4f">删除</a></a-popconfirm>
                </a-space>
              </template>
            </template>
          </a-table>
        </div>

        <!-- PVs -->
        <div v-if="selectedMenu[0] === 'pvs'">
          <div class="content-header"><h3>PersistentVolumes</h3></div>
          <a-table :columns="pvColumns" :data-source="filteredPvs" :loading="loading" row-key="name" size="small">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'name'">
                <a class="resource-link" @click="showResourceDetail('pv', '', record.name)">{{ record.name }}</a>
              </template>
              <template v-if="column.key === 'status'">
                <a-tag :color="record.status === 'Bound' ? 'green' : record.status === 'Available' ? 'blue' : 'orange'">{{ record.status }}</a-tag>
              </template>
              <template v-if="column.key === 'action'">
                <a @click="showYAMLModal(record, 'pv')">YAML</a>
              </template>
            </template>
          </a-table>
        </div>

        <!-- StorageClasses -->
        <div v-if="selectedMenu[0] === 'storageclasses'">
          <div class="content-header"><h3>StorageClasses</h3></div>
          <a-table :columns="storageClassColumns" :data-source="filteredStorageClasses" :loading="loading" row-key="name" size="small">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'name'">
                <a class="resource-link" @click="showResourceDetail('storageclass', '', record.name)">{{ record.name }}</a>
              </template>
              <template v-if="column.key === 'default'">
                <a-tag v-if="record.is_default" color="blue">默认</a-tag>
              </template>
              <template v-if="column.key === 'expansion'">
                <a-tag :color="record.allow_expansion ? 'green' : 'default'">{{ record.allow_expansion ? '支持' : '不支持' }}</a-tag>
              </template>
            </template>
          </a-table>
        </div>

        <!-- Nodes -->
        <div v-if="selectedMenu[0] === 'nodes'">
          <div class="content-header"><h3>节点管理</h3></div>
          <a-table :columns="nodeColumns" :data-source="filteredNodes" :loading="loading" row-key="name" size="small">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'name'">
                <a class="resource-link" @click="showResourceDetail('node', '', record.name)">{{ record.name }}</a>
              </template>
              <template v-if="column.key === 'status'">
                <a-tag :color="record.status === 'Ready' ? 'green' : 'red'">{{ record.status }}</a-tag>
                <a-tag v-if="!record.schedulable" color="orange">不可调度</a-tag>
              </template>
              <template v-if="column.key === 'roles'">
                <a-tag v-for="role in record.roles" :key="role" color="blue">{{ role }}</a-tag>
              </template>
              <template v-if="column.key === 'resources'">
                <div>CPU: {{ record.cpu_allocatable }} / {{ record.cpu_capacity }}</div>
                <div>内存: {{ record.memory_allocatable }}</div>
              </template>
              <template v-if="column.key === 'taints'">
                <a-tag v-for="(t, i) in (record.taints || []).slice(0, 2)" :key="i" style="margin: 2px">{{ t.key }}={{ t.effect }}</a-tag>
                <span v-if="(record.taints || []).length > 2" class="more-tag">+{{ record.taints.length - 2 }}</span>
              </template>
              <template v-if="column.key === 'action'">
                <a-space>
                  <a @click="showNodeDetail(record)">详情</a>
                  <a-popconfirm v-if="record.schedulable" title="确定设为不可调度？" @confirm="cordonNode(record)"><a>停止调度</a></a-popconfirm>
                  <a-popconfirm v-else title="确定恢复调度？" @confirm="uncordonNode(record)"><a>恢复调度</a></a-popconfirm>
                  <a @click="showTaintModal(record)">污点</a>
                </a-space>
              </template>
            </template>
          </a-table>
        </div>

        <!-- Events -->
        <div v-if="selectedMenu[0] === 'events'">
          <div class="content-header"><h3>事件</h3></div>
          <a-table :columns="eventColumns" :data-source="filteredEvents" :loading="loading" row-key="name" size="small">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'type'">
                <a-tag :color="record.type === 'Normal' ? 'green' : 'orange'">{{ record.type }}</a-tag>
              </template>
              <template v-if="column.key === 'message'">
                <a-tooltip :title="record.message">
                  <span style="max-width: 300px; display: inline-block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap">{{ record.message }}</span>
                </a-tooltip>
              </template>
            </template>
          </a-table>
        </div>

        <!-- HPA -->
        <div v-if="selectedMenu[0] === 'hpa'">
          <div class="content-header">
            <h3>HPA (水平自动伸缩)</h3>
            <a-button type="primary" @click="showCreateHPAModal" v-if="selectedNamespace"><PlusOutlined /> 创建 HPA</a-button>
          </div>
          <a-table :columns="hpaColumns" :data-source="filteredHPAs" :loading="loading" row-key="name" size="small">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'target'">
                <a-tag color="blue">{{ record.target_kind }}</a-tag>
                <span>{{ record.target_name }}</span>
              </template>
              <template v-if="column.key === 'replicas'">
                <span>{{ record.current_replicas }} → {{ record.desired_replicas }}</span>
                <span class="replica-range">({{ record.min_replicas }}-{{ record.max_replicas }})</span>
              </template>
              <template v-if="column.key === 'metrics'">
                <a-tag v-for="m in record.metrics" :key="m" color="green">{{ m }}</a-tag>
              </template>
              <template v-if="column.key === 'action'">
                <a-space>
                  <a @click="editHPA(record)">编辑</a>
                  <a-popconfirm title="确定删除此 HPA?" @confirm="deleteHPA(record)">
                    <a style="color: #ff4d4f">删除</a>
                  </a-popconfirm>
                </a-space>
              </template>
            </template>
          </a-table>
        </div>
      </div>
    </div>

    <!-- Pod 日志查看器 -->
    <PodLogViewer
      v-model:visible="logViewerVisible"
      :cluster-id="clusterId"
      :pod="currentPod"
      :pods="pods"
      @update:pod="onLogPodChange"
    />

    <!-- 节点详情弹窗 -->
    <a-modal v-model:open="nodeDetailVisible" :title="`节点详情 - ${currentNode?.name}`" width="900px" :footer="null">
      <a-descriptions :column="2" bordered size="small" v-if="nodeDetail">
        <a-descriptions-item label="名称">{{ nodeDetail.name }}</a-descriptions-item>
        <a-descriptions-item label="可调度">{{ nodeDetail.schedulable ? '是' : '否' }}</a-descriptions-item>
        <a-descriptions-item label="CPU">{{ nodeDetail.cpu_allocatable }} / {{ nodeDetail.cpu_capacity }}</a-descriptions-item>
        <a-descriptions-item label="内存">{{ nodeDetail.memory_allocatable }} / {{ nodeDetail.memory_capacity }}</a-descriptions-item>
        <a-descriptions-item label="Pod 数量">{{ nodeDetail.pod_count }}</a-descriptions-item>
        <a-descriptions-item label="创建时间">{{ nodeDetail.created_at }}</a-descriptions-item>
      </a-descriptions>
      <a-divider>污点</a-divider>
      <a-table :columns="[{title:'Key',dataIndex:'key'},{title:'Value',dataIndex:'value'},{title:'Effect',dataIndex:'effect'}]" :data-source="nodeDetail?.taints || []" :pagination="false" size="small" />
      <a-divider>条件</a-divider>
      <a-table :columns="[{title:'类型',dataIndex:'type'},{title:'状态',dataIndex:'status'},{title:'原因',dataIndex:'reason'}]" :data-source="nodeDetail?.conditions || []" :pagination="false" size="small" />
      <a-divider>Pod 列表</a-divider>
      <a-table :columns="[{title:'名称',dataIndex:'name'},{title:'命名空间',dataIndex:'namespace'},{title:'状态',dataIndex:'status'},{title:'IP',dataIndex:'ip'}]" :data-source="nodeDetail?.pods || []" :pagination="{ pageSize: 10 }" size="small" />
    </a-modal>

    <!-- 污点管理弹窗 -->
    <a-modal v-model:open="taintModalVisible" :title="`污点管理 - ${currentNode?.name}`" width="700px" :footer="null">
      <a-table :columns="taintColumns" :data-source="currentNode?.taints || []" :pagination="false" size="small">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'action'">
            <a-popconfirm title="确定移除？" @confirm="removeTaint(record)"><a style="color: #ff4d4f">移除</a></a-popconfirm>
          </template>
        </template>
      </a-table>
      <a-divider>添加污点</a-divider>
      <a-form layout="inline" style="margin-top: 12px">
        <a-form-item label="Key"><a-input v-model:value="newTaint.key" placeholder="key" style="width: 150px" /></a-form-item>
        <a-form-item label="Value"><a-input v-model:value="newTaint.value" placeholder="value" style="width: 150px" /></a-form-item>
        <a-form-item label="Effect">
          <a-select v-model:value="newTaint.effect" style="width: 150px">
            <a-select-option value="NoSchedule">NoSchedule</a-select-option>
            <a-select-option value="PreferNoSchedule">PreferNoSchedule</a-select-option>
            <a-select-option value="NoExecute">NoExecute</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item><a-button type="primary" @click="addTaint" :loading="addingTaint">添加</a-button></a-form-item>
      </a-form>
    </a-modal>

    <!-- 节点加入弹窗 -->
    <a-modal v-model:open="joinNodeVisible" title="添加节点到集群" width="800px" :footer="null">
      <a-alert message="请在新节点上执行以下命令将其加入集群" type="info" show-icon style="margin-bottom: 16px" />
      <a-spin :spinning="loadingJoinCommand">
        <div v-if="joinCommand">
          <p><strong>1. 确保新节点已安装 Docker 和 kubeadm</strong></p>
          <p><strong>2. 在新节点上执行以下命令：</strong></p>
          <div class="join-command-box">
            <pre>{{ joinCommand }}</pre>
            <a-button type="link" @click="copyJoinCommand"><CopyOutlined /> 复制</a-button>
          </div>
          <a-alert message="注意：此 token 有效期为 24 小时，请尽快使用" type="warning" show-icon style="margin-top: 16px" />
        </div>
        <div v-else>
          <a-button type="primary" @click="generateJoinCommand" :loading="loadingJoinCommand">生成加入命令</a-button>
        </div>
      </a-spin>
    </a-modal>

    <!-- 创建命名空间弹窗 -->
    <a-modal v-model:open="createNamespaceVisible" title="创建命名空间" @ok="handleCreateNamespace" :confirm-loading="creatingNamespace" ok-text="创建" cancel-text="取消">
      <a-form :label-col="{ span: 6 }">
        <a-form-item label="名称" required>
          <a-input v-model:value="newNamespaceName" placeholder="请输入命名空间名称" />
        </a-form-item>
        <a-form-item label="标签">
          <div v-for="(label, index) in newNamespaceLabels" :key="index" style="display: flex; gap: 8px; margin-bottom: 8px">
            <a-input v-model:value="label.key" placeholder="Key" style="width: 40%" />
            <a-input v-model:value="label.value" placeholder="Value" style="width: 40%" />
            <a-button @click="removeNamespaceLabel(index)" danger><MinusOutlined /></a-button>
          </div>
          <a-button @click="addNamespaceLabel" type="dashed" block><PlusOutlined /> 添加标签</a-button>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 伸缩弹窗 -->
    <a-modal v-model:open="scaleModalVisible" title="调整副本数" @ok="handleScale" :confirm-loading="scaling">
      <a-form :label-col="{ span: 6 }">
        <a-form-item label="Deployment">{{ currentDeployment?.name }}</a-form-item>
        <a-form-item label="当前副本数">{{ currentDeployment?.replicas }}</a-form-item>
        <a-form-item label="目标副本数">
          <a-input-number v-model:value="targetReplicas" :min="0" :max="100" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- YAML 编辑弹窗 -->
    <a-modal v-model:open="yamlModalVisible" :title="yamlModalTitle" width="900px" @ok="handleApplyYAML" :confirm-loading="applyingYAML" ok-text="应用" cancel-text="取消">
      <a-spin :spinning="loadingYAML">
        <a-textarea v-model:value="yamlContent" :rows="20" style="font-family: 'Consolas', 'Monaco', monospace; font-size: 12px" />
      </a-spin>
    </a-modal>

    <!-- 创建资源弹窗 -->
    <a-modal v-model:open="createModalVisible" :title="`创建 ${currentResourceLabel}`" width="900px" @ok="handleCreateResource" :confirm-loading="creatingResource" ok-text="创建" cancel-text="取消">
      <a-tabs v-model:activeKey="createMode">
        <a-tab-pane key="form" tab="表单创建">
          <ResourceFormDeployment v-if="selectedMenu[0] === 'deployments'" :form="formData" />
          <ResourceFormService v-else-if="selectedMenu[0] === 'services'" :form="formData" />
          <ResourceFormConfigMap v-else-if="selectedMenu[0] === 'configmaps'" :form="formData" />
          <ResourceFormSecret v-else-if="selectedMenu[0] === 'secrets'" :form="formData" />
          <ResourceFormIngress v-else-if="selectedMenu[0] === 'ingresses'" :form="formData" />
          <ResourceFormSimple v-else :form="formData" :resource-type="selectedMenu[0]" />
        </a-tab-pane>
        <a-tab-pane key="yaml" tab="YAML 创建">
          <a-alert v-if="yamlError" :message="yamlError" type="error" show-icon style="margin-bottom: 12px" />
          <a-textarea v-model:value="createYAMLContent" :rows="20" style="font-family: 'Consolas', 'Monaco', monospace; font-size: 12px" placeholder="输入 YAML 内容" @change="onYAMLChange" />
        </a-tab-pane>
      </a-tabs>
    </a-modal>

    <!-- 资源详情抽屉 -->
    <ResourceDetail
      :visible="detailVisible"
      :cluster-id="clusterId"
      :resource-type="detailResourceType"
      :namespace="detailNamespace"
      :name="detailName"
      @close="detailVisible = false"
      @navigate="handleDetailNavigate"
      @refresh="fetchData"
    />

    <!-- 镜像更新弹窗 -->
    <a-modal v-model:open="imageUpdateModalVisible" title="更新镜像" @ok="handleImageUpdate" :confirm-loading="updatingImage" ok-text="更新" cancel-text="取消">
      <a-form :label-col="{ span: 6 }">
        <a-form-item label="Deployment">{{ currentDeployment?.name }}</a-form-item>
        <a-form-item label="命名空间">{{ currentDeployment?.namespace }}</a-form-item>
        <a-divider>容器镜像</a-divider>
        <div v-for="(container, index) in imageContainers" :key="index" style="margin-bottom: 16px">
          <a-form-item :label="`容器 ${index + 1}`">
            <a-input v-model:value="container.newImage" placeholder="输入新镜像地址" />
            <div style="color: #999; font-size: 12px; margin-top: 4px">当前: {{ container.image }}</div>
          </a-form-item>
        </div>
      </a-form>
    </a-modal>

    <!-- 回滚弹窗 -->
    <a-modal v-model:open="rollbackModalVisible" title="回滚 Deployment" @ok="handleRollback" :confirm-loading="rollingBack" ok-text="回滚" cancel-text="取消">
      <a-form :label-col="{ span: 6 }">
        <a-form-item label="Deployment">{{ currentDeployment?.name }}</a-form-item>
        <a-form-item label="命名空间">{{ currentDeployment?.namespace }}</a-form-item>
      </a-form>
      <a-divider>版本历史</a-divider>
      <a-spin :spinning="loadingRevisions">
        <a-radio-group v-model:value="selectedRevision" style="width: 100%">
          <a-table :columns="revisionColumns" :data-source="revisionHistory" :pagination="false" size="small" row-key="revision">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'select'">
                <a-radio :value="record.revision" />
              </template>
              <template v-if="column.key === 'image'">
                <a-tooltip :title="record.image">
                  <span style="max-width: 200px; display: inline-block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap">{{ record.image }}</span>
                </a-tooltip>
              </template>
            </template>
          </a-table>
        </a-radio-group>
        <a-empty v-if="!loadingRevisions && revisionHistory.length === 0" description="暂无版本历史" />
      </a-spin>
    </a-modal>

    <!-- Pod 终端抽屉 -->
    <a-drawer
      v-model:open="terminalDrawerVisible"
      :title="`终端 - ${currentTerminalPod?.name || ''}`"
      width="80%"
      placement="right"
      :destroy-on-close="true"
      @close="closeTerminalDrawer"
    >
      <PodTerminal
        v-if="terminalDrawerVisible && currentTerminalPod"
        :cluster-id="clusterId"
        :namespace="currentTerminalPod.namespace"
        :pod-name="currentTerminalPod.name"
      />
    </a-drawer>

    <!-- HPA 创建/编辑弹窗 -->
    <a-modal v-model:open="hpaModalVisible" :title="editingHPA ? '编辑 HPA' : '创建 HPA'" @ok="saveHPA" :confirm-loading="savingHPA">
      <a-form :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }">
        <a-form-item label="名称" v-if="!editingHPA">
          <a-input v-model:value="hpaFormData.name" placeholder="HPA 名称" />
        </a-form-item>
        <a-form-item label="命名空间" v-if="!editingHPA">
          <a-select v-model:value="hpaFormData.namespace" placeholder="选择命名空间">
            <a-select-option v-for="ns in namespaces" :key="ns.name" :value="ns.name">{{ ns.name }}</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="目标类型" v-if="!editingHPA">
          <a-select v-model:value="hpaFormData.target_kind">
            <a-select-option value="Deployment">Deployment</a-select-option>
            <a-select-option value="StatefulSet">StatefulSet</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="目标名称" v-if="!editingHPA">
          <a-input v-model:value="hpaFormData.target_name" placeholder="Deployment/StatefulSet 名称" />
        </a-form-item>
        <a-form-item label="最小副本数">
          <a-input-number v-model:value="hpaFormData.min_replicas" :min="1" style="width: 100%" />
        </a-form-item>
        <a-form-item label="最大副本数">
          <a-input-number v-model:value="hpaFormData.max_replicas" :min="1" style="width: 100%" />
        </a-form-item>
        <a-form-item label="CPU 目标 (%)">
          <a-input-number v-model:value="hpaFormData.cpu_target_percent" :min="1" :max="100" placeholder="如 80" style="width: 100%" />
        </a-form-item>
        <a-form-item label="内存目标 (%)">
          <a-input-number v-model:value="hpaFormData.mem_target_percent" :min="1" :max="100" placeholder="可选" style="width: 100%" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>


<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message, Modal } from 'ant-design-vue'
import {
  ArrowLeftOutlined, ReloadOutlined, DeploymentUnitOutlined, DatabaseOutlined,
  ClusterOutlined, ThunderboltOutlined, FieldTimeOutlined, AppstoreOutlined,
  ApiOutlined, GlobalOutlined, FileTextOutlined, LockOutlined, HddOutlined, PlusOutlined, MinusOutlined,
  CloudServerOutlined, BellOutlined, InboxOutlined, FolderOutlined, CopyOutlined, BlockOutlined,
  DownOutlined, ExpandOutlined, EditOutlined, RollbackOutlined, SyncOutlined, DeleteOutlined, DashboardOutlined
} from '@ant-design/icons-vue'
import { k8sClusterApi, k8sResourceApi, k8sNodeApi, k8sStorageApi, k8sEventApi, k8sDeploymentApi, k8sPodApi, k8sHPAApi } from '@/services/k8s'
import type { K8sNamespace, K8sDeployment, K8sPod, K8sService, K8sConfigMap, K8sSecret, K8sNode, K8sNodeDetail, K8sPV, K8sStorageClass, K8sEvent, K8sRevisionInfo, K8sContainer, K8sHPA } from '@/services/k8s'
import { 
  ResourceFormDeployment, ResourceFormService, ResourceFormConfigMap, 
  ResourceFormSecret, ResourceFormIngress, ResourceFormSimple, ResourceDetail,
  createDefaultFormData, buildYAMLFromForm, validateForm, validateYAML as validateYAMLContent
} from './components'
import PodTerminal from './components/PodTerminal.vue'
import PodLogViewer from '@/components/k8s/PodLogViewer.vue'
import type { ResourceFormData } from './components'

const route = useRoute()
const router = useRouter()
const clusterId = Number(route.params.id)

const clusterName = ref('')
const loading = ref(false)
const loadingNs = ref(false)
const scaling = ref(false)
const selectedMenu = ref(['deployments'])
const selectedNamespace = ref('')

// 资源详情抽屉
const detailVisible = ref(false)
const detailResourceType = ref('')
const detailNamespace = ref('')
const detailName = ref('')

// 数据
const namespaces = ref<K8sNamespace[]>([])
const deployments = ref<K8sDeployment[]>([])
const statefulSets = ref<any[]>([])
const daemonSets = ref<any[]>([])
const jobs = ref<any[]>([])
const cronJobs = ref<any[]>([])
const pods = ref<K8sPod[]>([])
const services = ref<K8sService[]>([])
const ingresses = ref<any[]>([])
const configMaps = ref<K8sConfigMap[]>([])
const secrets = ref<K8sSecret[]>([])
const pvcs = ref<any[]>([])
const pvs = ref<K8sPV[]>([])
const storageClasses = ref<K8sStorageClass[]>([])
const nodes = ref<K8sNode[]>([])
const events = ref<K8sEvent[]>([])

// HPA 相关
const hpas = ref<any[]>([])
const hpaModalVisible = ref(false)
const editingHPA = ref<any>(null)
const hpaFormData = ref({
  name: '',
  namespace: '',
  target_kind: 'Deployment' as 'Deployment' | 'StatefulSet',
  target_name: '',
  min_replicas: 1,
  max_replicas: 10,
  cpu_target_percent: 80,
  mem_target_percent: undefined as number | undefined
})
const savingHPA = ref(false)
const nodeDetailVisible = ref(false)
const currentNode = ref<K8sNode | null>(null)
const nodeDetail = ref<K8sNodeDetail | null>(null)
const taintModalVisible = ref(false)
const newTaint = ref({ key: '', value: '', effect: 'NoSchedule' })
const addingTaint = ref(false)
const joinNodeVisible = ref(false)
const joinCommand = ref('')
const loadingJoinCommand = ref(false)

// 命名空间相关
const createNamespaceVisible = ref(false)
const creatingNamespace = ref(false)
const newNamespaceName = ref('')
const newNamespaceLabels = ref<{ key: string; value: string }[]>([])

// 搜索相关
const searchKeyword = ref('')

// 日志相关
const logViewerVisible = ref(false)
const currentPod = ref<K8sPod | null>(null)

// 伸缩相关
const scaleModalVisible = ref(false)
const currentDeployment = ref<K8sDeployment | null>(null)
const targetReplicas = ref(1)

// 镜像更新相关
const imageUpdateModalVisible = ref(false)
const imageContainers = ref<{ name: string; image: string; newImage: string }[]>([])
const updatingImage = ref(false)

// 回滚相关
const rollbackModalVisible = ref(false)
const revisionHistory = ref<K8sRevisionInfo[]>([])
const selectedRevision = ref<number | null>(null)
const loadingRevisions = ref(false)
const rollingBack = ref(false)

// Pod 终端相关
const terminalDrawerVisible = ref(false)
const currentTerminalPod = ref<K8sPod | null>(null)
const podContainers = ref<K8sContainer[]>([])

// YAML 编辑相关
const yamlModalVisible = ref(false)
const yamlModalTitle = ref('')
const yamlContent = ref('')
const loadingYAML = ref(false)
const applyingYAML = ref(false)
const currentResource = ref<{ name: string; namespace: string; type: string } | null>(null)

// 创建资源相关
const createModalVisible = ref(false)
const createYAMLContent = ref('')
const creatingResource = ref(false)
const createMode = ref('form')
const yamlError = ref('')

// 表单数据
const formData = ref<ResourceFormData>(createDefaultFormData())
const resetFormData = () => { formData.value = createDefaultFormData() }

// 资源类型映射
const resourceTypeMap: Record<string, { type: string; label: string; canCreate: boolean }> = {
  deployments: { type: 'deployment', label: 'Deployment', canCreate: true },
  statefulsets: { type: 'statefulset', label: 'StatefulSet', canCreate: true },
  daemonsets: { type: 'daemonset', label: 'DaemonSet', canCreate: true },
  jobs: { type: 'job', label: 'Job', canCreate: true },
  cronjobs: { type: 'cronjob', label: 'CronJob', canCreate: true },
  pods: { type: 'pod', label: 'Pod', canCreate: false },
  services: { type: 'service', label: 'Service', canCreate: true },
  ingresses: { type: 'ingress', label: 'Ingress', canCreate: true },
  configmaps: { type: 'configmap', label: 'ConfigMap', canCreate: true },
  secrets: { type: 'secret', label: 'Secret', canCreate: true },
  pvcs: { type: 'persistentvolumeclaim', label: 'PVC', canCreate: true },
  pvs: { type: 'pv', label: 'PV', canCreate: false },
  storageclasses: { type: 'storageclass', label: 'StorageClass', canCreate: false },
  nodes: { type: 'node', label: '节点', canCreate: false },
  events: { type: 'event', label: '事件', canCreate: false },
  hpa: { type: 'hpa', label: 'HPA', canCreate: false }
}

const currentResourceLabel = computed(() => {
  const menu = selectedMenu.value[0]
  return resourceTypeMap[menu]?.label || ''
})

const canCreateResource = computed(() => {
  const menu = selectedMenu.value[0]
  return resourceTypeMap[menu]?.canCreate ?? false
})

// 是否显示命名空间选择器（节点、PV、StorageClass、命名空间等集群级资源不需要）
const showNamespaceSelect = computed(() => {
  const clusterLevelResources = ['nodes', 'pvs', 'storageclasses', 'namespaces']
  return !clusterLevelResources.includes(selectedMenu.value[0])
})

// 过滤后的数据
const filteredNamespaces = computed(() => filterByName(namespaces.value))
const filteredDeployments = computed(() => filterByName(deployments.value))
const filteredStatefulSets = computed(() => filterByName(statefulSets.value))
const filteredDaemonSets = computed(() => filterByName(daemonSets.value))
const filteredJobs = computed(() => filterByName(jobs.value))
const filteredCronJobs = computed(() => filterByName(cronJobs.value))
const filteredPods = computed(() => filterByName(pods.value))
const filteredServices = computed(() => filterByName(services.value))
const filteredIngresses = computed(() => filterByName(ingresses.value))
const filteredConfigMaps = computed(() => filterByName(configMaps.value))
const filteredSecrets = computed(() => filterByName(secrets.value))
const filteredPvcs = computed(() => filterByName(pvcs.value))
const filteredPvs = computed(() => filterByName(pvs.value))
const filteredStorageClasses = computed(() => filterByName(storageClasses.value))
const filteredNodes = computed(() => filterByName(nodes.value))
const filteredEvents = computed(() => {
  if (!searchKeyword.value) return events.value
  const kw = searchKeyword.value.toLowerCase()
  return events.value.filter(e => e.object?.toLowerCase().includes(kw) || e.reason?.toLowerCase().includes(kw) || e.message?.toLowerCase().includes(kw))
})

const filteredHPAs = computed(() => filterByName(hpas.value))

const filterByName = (list: any[]) => {
  if (!searchKeyword.value) return list
  const kw = searchKeyword.value.toLowerCase()
  return list.filter(item => item.name?.toLowerCase().includes(kw))
}

// 表格列定义
const namespaceColumns = [
  { title: '名称', dataIndex: 'name', key: 'name', width: 250 },
  { title: '状态', key: 'status', width: 100 },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 170 },
  { title: '操作', key: 'action', width: 120 }
]

const deploymentColumns = [
  { title: '名称', dataIndex: 'name', key: 'name', width: 250 },
  { title: '命名空间', dataIndex: 'namespace', key: 'namespace', width: 150 },
  { title: '副本', key: 'replicas', width: 80 },
  { title: '镜像', key: 'images' },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 170 },
  { title: '操作', key: 'action', width: 200 }
]

const statefulSetColumns = [
  { title: '名称', dataIndex: 'name', key: 'name' },
  { title: '命名空间', dataIndex: 'namespace', key: 'namespace' },
  { title: '副本', key: 'replicas', width: 80 },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 170 },
  { title: '操作', key: 'action', width: 150 }
]

const daemonSetColumns = [
  { title: '名称', dataIndex: 'name', key: 'name' },
  { title: '命名空间', dataIndex: 'namespace', key: 'namespace' },
  { title: '状态', key: 'status', width: 80 },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 170 },
  { title: '操作', key: 'action', width: 150 }
]

const jobColumns = [
  { title: '名称', dataIndex: 'name', key: 'name' },
  { title: '命名空间', dataIndex: 'namespace', key: 'namespace' },
  { title: '状态', key: 'status', width: 100 },
  { title: '完成数', dataIndex: 'completions', key: 'completions', width: 80 },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 170 },
  { title: '操作', key: 'action', width: 150 }
]

const cronJobColumns = [
  { title: '名称', dataIndex: 'name', key: 'name' },
  { title: '命名空间', dataIndex: 'namespace', key: 'namespace' },
  { title: '调度', dataIndex: 'schedule', key: 'schedule' },
  { title: '状态', key: 'suspend', width: 100 },
  { title: '上次调度', dataIndex: 'last_schedule', key: 'last_schedule', width: 170 },
  { title: '操作', key: 'action', width: 150 }
]

const podColumns = [
  { title: '名称', dataIndex: 'name', key: 'name', width: 300 },
  { title: '命名空间', dataIndex: 'namespace', key: 'namespace', width: 150 },
  { title: '状态', key: 'status', width: 100 },
  { title: 'IP', dataIndex: 'ip', key: 'ip', width: 130 },
  { title: '节点', dataIndex: 'node', key: 'node' },
  { title: '重启', dataIndex: 'restarts', key: 'restarts', width: 60 },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 170 },
  { title: '操作', key: 'action', width: 120 }
]

const serviceColumns = [
  { title: '名称', dataIndex: 'name', key: 'name' },
  { title: '命名空间', dataIndex: 'namespace', key: 'namespace' },
  { title: '类型', key: 'type', width: 100 },
  { title: 'ClusterIP', dataIndex: 'cluster_ip', key: 'cluster_ip', width: 130 },
  { title: '端口', key: 'ports' },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 170 },
  { title: '操作', key: 'action', width: 150 }
]

const ingressColumns = [
  { title: '名称', dataIndex: 'name', key: 'name' },
  { title: '命名空间', dataIndex: 'namespace', key: 'namespace' },
  { title: 'Class', dataIndex: 'ingress_class', key: 'ingress_class' },
  { title: 'Hosts', key: 'hosts' },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 170 },
  { title: '操作', key: 'action', width: 150 }
]

const configMapColumns = [
  { title: '名称', dataIndex: 'name', key: 'name' },
  { title: '命名空间', dataIndex: 'namespace', key: 'namespace' },
  { title: 'Keys', key: 'keys' },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 170 },
  { title: '操作', key: 'action', width: 150 }
]

const secretColumns = [
  { title: '名称', dataIndex: 'name', key: 'name' },
  { title: '命名空间', dataIndex: 'namespace', key: 'namespace' },
  { title: '类型', dataIndex: 'type', key: 'type' },
  { title: 'Keys', key: 'keys' },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 170 },
  { title: '操作', key: 'action', width: 150 }
]

const pvcColumns = [
  { title: '名称', dataIndex: 'name', key: 'name' },
  { title: '命名空间', dataIndex: 'namespace', key: 'namespace' },
  { title: '状态', key: 'status', width: 100 },
  { title: '容量', dataIndex: 'capacity', key: 'capacity', width: 100 },
  { title: 'StorageClass', dataIndex: 'storage_class', key: 'storage_class' },
  { title: '访问模式', dataIndex: 'access_modes', key: 'access_modes' },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 170 },
  { title: '操作', key: 'action', width: 150 }
]

const pvColumns = [
  { title: '名称', dataIndex: 'name', key: 'name' },
  { title: '状态', key: 'status', width: 100 },
  { title: '容量', dataIndex: 'capacity', key: 'capacity', width: 100 },
  { title: '访问模式', dataIndex: 'access_modes', key: 'access_modes' },
  { title: '回收策略', dataIndex: 'reclaim_policy', key: 'reclaim_policy' },
  { title: 'StorageClass', dataIndex: 'storage_class', key: 'storage_class' },
  { title: '绑定', dataIndex: 'claim_ref', key: 'claim_ref' },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 170 },
  { title: '操作', key: 'action', width: 100 }
]

const storageClassColumns = [
  { title: '名称', dataIndex: 'name', key: 'name' },
  { title: '默认', key: 'default', width: 80 },
  { title: 'Provisioner', dataIndex: 'provisioner', key: 'provisioner' },
  { title: '回收策略', dataIndex: 'reclaim_policy', key: 'reclaim_policy' },
  { title: '绑定模式', dataIndex: 'volume_binding_mode', key: 'volume_binding_mode' },
  { title: '扩容', key: 'expansion', width: 80 },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 170 }
]

const nodeColumns = [
  { title: '名称', dataIndex: 'name', key: 'name', width: 200 },
  { title: '状态', key: 'status', width: 150 },
  { title: '角色', key: 'roles', width: 120 },
  { title: 'IP', dataIndex: 'internal_ip', key: 'internal_ip', width: 130 },
  { title: '资源', key: 'resources', width: 200 },
  { title: '污点', key: 'taints' },
  { title: '版本', dataIndex: 'kubelet_version', key: 'kubelet_version', width: 100 },
  { title: '操作', key: 'action', width: 220 }
]

const eventColumns = [
  { title: '类型', key: 'type', width: 80 },
  { title: '原因', dataIndex: 'reason', key: 'reason', width: 120 },
  { title: '对象', dataIndex: 'object', key: 'object', width: 200 },
  { title: '消息', key: 'message' },
  { title: '次数', dataIndex: 'count', key: 'count', width: 60 },
  { title: '时间', dataIndex: 'last_timestamp', key: 'last_timestamp', width: 170 }
]

const hpaColumns = [
  { title: '名称', dataIndex: 'name', key: 'name', width: 200 },
  { title: '命名空间', dataIndex: 'namespace', key: 'namespace', width: 120 },
  { title: '目标', key: 'target', width: 200 },
  { title: '副本数', key: 'replicas', width: 150 },
  { title: '指标', key: 'metrics', width: 200 },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 170 },
  { title: '操作', key: 'action', width: 120 }
]

const taintColumns = [
  { title: 'Key', dataIndex: 'key', key: 'key' },
  { title: 'Value', dataIndex: 'value', key: 'value' },
  { title: 'Effect', dataIndex: 'effect', key: 'effect' },
  { title: '操作', key: 'action', width: 80 }
]

const revisionColumns = [
  { title: '选择', key: 'select', width: 60 },
  { title: '版本', dataIndex: 'revision', key: 'revision', width: 80 },
  { title: '镜像', key: 'image' },
  { title: '变更原因', dataIndex: 'change_cause', key: 'change_cause', width: 150 },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 170 }
]


const goBack = () => router.push('/k8s/clusters')

const getPodStatusColor = (status: string) => {
  const colors: Record<string, string> = { Running: 'green', Pending: 'orange', Succeeded: 'blue', Failed: 'red', Unknown: 'default' }
  return colors[status] || 'default'
}

const fetchClusterInfo = async () => {
  try {
    const res = await k8sClusterApi.getCluster(clusterId)
    if (res.code === 0 && res.data) clusterName.value = res.data.name
  } catch (e: any) {
    message.error(e.message || '获取集群信息失败')
  }
}

const fetchNamespaces = async () => {
  loadingNs.value = true
  try {
    const res = await k8sResourceApi.getNamespaces(clusterId)
    if (res.code === 0 && res.data) namespaces.value = res.data
  } catch (e: any) {
    message.error(e.message || '获取命名空间失败')
  } finally {
    loadingNs.value = false
  }
}

const fetchData = async () => {
  loading.value = true
  const ns = selectedNamespace.value
  try {
    switch (selectedMenu.value[0]) {
      case 'namespaces':
        const nsRes = await k8sResourceApi.getNamespaces(clusterId)
        if (nsRes.code === 0) namespaces.value = nsRes.data || []
        break
      case 'deployments':
        const depRes = await k8sResourceApi.getDeployments(clusterId, ns)
        if (depRes.code === 0) deployments.value = depRes.data || []
        break
      case 'statefulsets':
        const stsRes = await k8sResourceApi.getStatefulSets(clusterId, ns)
        if (stsRes.code === 0) statefulSets.value = stsRes.data || []
        break
      case 'daemonsets':
        const dsRes = await k8sResourceApi.getDaemonSets(clusterId, ns)
        if (dsRes.code === 0) daemonSets.value = dsRes.data || []
        break
      case 'jobs':
        const jobRes = await k8sResourceApi.getJobs(clusterId, ns)
        if (jobRes.code === 0) jobs.value = jobRes.data || []
        break
      case 'cronjobs':
        const cjRes = await k8sResourceApi.getCronJobs(clusterId, ns)
        if (cjRes.code === 0) cronJobs.value = cjRes.data || []
        break
      case 'pods':
        const podRes = await k8sResourceApi.getPods(clusterId, ns)
        if (podRes.code === 0) pods.value = podRes.data || []
        break
      case 'services':
        const svcRes = await k8sResourceApi.getServices(clusterId, ns)
        if (svcRes.code === 0) services.value = svcRes.data || []
        break
      case 'ingresses':
        const ingRes = await k8sResourceApi.getIngresses(clusterId, ns)
        if (ingRes.code === 0) ingresses.value = ingRes.data || []
        break
      case 'configmaps':
        const cmRes = await k8sResourceApi.getConfigMaps(clusterId, ns)
        if (cmRes.code === 0) configMaps.value = cmRes.data || []
        break
      case 'secrets':
        const secRes = await k8sResourceApi.getSecrets(clusterId, ns)
        if (secRes.code === 0) secrets.value = secRes.data || []
        break
      case 'pvcs':
        const pvcRes = await k8sResourceApi.getPVCs(clusterId, ns)
        if (pvcRes.code === 0) pvcs.value = pvcRes.data || []
        break
      case 'pvs':
        const pvRes = await k8sStorageApi.getPVs(clusterId)
        if (pvRes.code === 0) pvs.value = pvRes.data || []
        break
      case 'storageclasses':
        const scRes = await k8sStorageApi.getStorageClasses(clusterId)
        if (scRes.code === 0) storageClasses.value = scRes.data || []
        break
      case 'nodes':
        const nodeRes = await k8sNodeApi.getNodes(clusterId)
        if (nodeRes.code === 0) nodes.value = nodeRes.data || []
        break
      case 'events':
        const eventRes = await k8sEventApi.getEvents(clusterId, ns)
        if (eventRes.code === 0) events.value = eventRes.data || []
        break
      case 'hpa':
        const hpaRes = await k8sHPAApi.list(clusterId, ns)
        if (hpaRes.code === 0) hpas.value = hpaRes.data || []
        break
    }
  } catch (e: any) {
    message.error(e.message || '获取数据失败')
  } finally {
    loading.value = false
  }
}

const refreshData = () => fetchData()

// 监听菜单变化，自动加载数据（只在菜单真正变化时触发）
let lastMenu = 'deployments'
watch(selectedMenu, (newVal) => {
  if (newVal[0] !== lastMenu) {
    lastMenu = newVal[0]
    fetchData()
  }
})

// 监听命名空间变化，自动加载数据（只在命名空间真正变化时触发）
let lastNamespace = ''
watch(() => selectedNamespace.value, (newVal) => {
  if (newVal !== lastNamespace) {
    lastNamespace = newVal
    fetchData()
  }
})

// 搜索相关方法
const onSearch = () => {
  // 搜索时不需要重新请求数据，因为是前端过滤
}
const onSearchChange = () => {
  // 实时过滤，不需要额外操作
}

// 节点加入相关方法
const showJoinNodeModal = () => {
  joinCommand.value = ''
  joinNodeVisible.value = true
}

const generateJoinCommand = async () => {
  loadingJoinCommand.value = true
  try {
    const res = await k8sNodeApi.getJoinCommand(clusterId)
    if (res.code === 0) {
      joinCommand.value = res.data || ''
    } else {
      message.error(res.message || '生成加入命令失败')
    }
  } catch (e: any) {
    message.error(e.message || '生成加入命令失败')
  } finally {
    loadingJoinCommand.value = false
  }
}

const copyJoinCommand = () => {
  if (joinCommand.value) {
    navigator.clipboard.writeText(joinCommand.value)
    message.success('已复制到剪贴板')
  }
}

// 命名空间管理方法
const showCreateNamespaceModal = () => {
  newNamespaceName.value = ''
  newNamespaceLabels.value = []
  createNamespaceVisible.value = true
}

const addNamespaceLabel = () => {
  newNamespaceLabels.value.push({ key: '', value: '' })
}

const removeNamespaceLabel = (index: number) => {
  newNamespaceLabels.value.splice(index, 1)
}

const handleCreateNamespace = async () => {
  if (!newNamespaceName.value.trim()) {
    message.warning('请输入命名空间名称')
    return
  }
  // 验证名称格式
  const nameRegex = /^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/
  if (!nameRegex.test(newNamespaceName.value)) {
    message.warning('命名空间名称只能包含小写字母、数字和连字符，且必须以字母或数字开头和结尾')
    return
  }
  
  creatingNamespace.value = true
  try {
    const labels: Record<string, string> = {}
    newNamespaceLabels.value.forEach(l => {
      if (l.key.trim()) labels[l.key.trim()] = l.value.trim()
    })
    
    const res = await k8sResourceApi.createNamespace(clusterId, newNamespaceName.value, labels)
    if (res.code === 0) {
      message.success('创建成功')
      createNamespaceVisible.value = false
      fetchData()
      fetchNamespaces() // 刷新命名空间下拉列表
    } else {
      message.error(res.message || '创建失败')
    }
  } catch (e: any) {
    message.error(e.message || '创建失败')
  } finally {
    creatingNamespace.value = false
  }
}

const deleteNamespace = async (ns: K8sNamespace) => {
  try {
    const res = await k8sResourceApi.deleteNamespace(clusterId, ns.name)
    if (res.code === 0) {
      message.success('删除成功')
      fetchData()
      fetchNamespaces() // 刷新命名空间下拉列表
    } else {
      message.error(res.message || '删除失败')
    }
  } catch (e: any) {
    message.error(e.message || '删除失败')
  }
}

const isSystemNamespace = (name: string) => {
  const systemNamespaces = ['default', 'kube-system', 'kube-public', 'kube-node-lease']
  return systemNamespaces.includes(name)
}

const showLogs = (pod: K8sPod) => {
  currentPod.value = pod
  logViewerVisible.value = true
}

const onLogPodChange = (pod: K8sPod) => {
  currentPod.value = pod
}

const deletePod = async (pod: K8sPod) => {
  try {
    await k8sResourceApi.deletePod(clusterId, pod.namespace, pod.name)
    message.success('删除成功')
    fetchData()
  } catch (e: any) {
    message.error(e.message || '删除失败')
  }
}

const showScaleModal = (deploy: K8sDeployment) => {
  currentDeployment.value = deploy
  targetReplicas.value = deploy.replicas
  scaleModalVisible.value = true
}

const handleScale = async () => {
  if (!currentDeployment.value) return
  scaling.value = true
  try {
    await k8sResourceApi.scaleDeployment(clusterId, currentDeployment.value.namespace, currentDeployment.value.name, targetReplicas.value)
    message.success('调整成功')
    scaleModalVisible.value = false
    fetchData()
  } catch (e: any) {
    message.error(e.message || '调整失败')
  } finally {
    scaling.value = false
  }
}

const restartDeployment = async (deploy: K8sDeployment) => {
  try {
    await k8sResourceApi.restartDeployment(clusterId, deploy.namespace, deploy.name)
    message.success('重启成功')
    fetchData()
  } catch (e: any) {
    message.error(e.message || '重启失败')
  }
}

// 打开 Pod 终端
const openTerminal = async (pod: K8sPod) => {
  currentTerminalPod.value = pod
  // 获取容器列表
  try {
    const res = await k8sPodApi.getContainers(clusterId, pod.namespace, pod.name)
    podContainers.value = res.data || []
  } catch (e) {
    // 如果获取失败，使用 pod 自带的容器信息
    podContainers.value = pod.containers || []
  }
  terminalDrawerVisible.value = true
}

// 关闭终端抽屉
const closeTerminalDrawer = () => {
  terminalDrawerVisible.value = false
  currentTerminalPod.value = null
}

// 显示镜像更新弹窗
const showImageUpdateModal = async (deploy: K8sDeployment) => {
  currentDeployment.value = deploy
  // 解析镜像列表
  imageContainers.value = (deploy.images || []).map((img, index) => ({
    name: `container-${index}`,
    image: img,
    newImage: img
  }))
  
  // 尝试获取更详细的容器信息
  try {
    const res = await k8sDeploymentApi.get(clusterId, deploy.namespace, deploy.name)
    if (res.data && res.data.containers) {
      // 从详情中获取容器名称
      imageContainers.value = res.data.containers.map((c: any) => ({
        name: c.name,
        image: c.image,
        newImage: c.image
      }))
    }
  } catch (e) {
    // 忽略错误，使用默认值
  }
  
  imageUpdateModalVisible.value = true
}

// 执行镜像更新
const handleImageUpdate = async () => {
  if (!currentDeployment.value) return
  
  const changedContainers = imageContainers.value.filter(c => c.image !== c.newImage)
  if (changedContainers.length === 0) {
    message.warning('没有修改任何镜像')
    return
  }
  
  updatingImage.value = true
  try {
    // 更新每个修改的容器镜像
    for (const container of changedContainers) {
      await k8sDeploymentApi.updateImage(
        clusterId,
        currentDeployment.value.namespace,
        currentDeployment.value.name,
        container.name,
        container.newImage
      )
    }
    message.success('镜像更新成功')
    imageUpdateModalVisible.value = false
    fetchData()
  } catch (e: any) {
    message.error(e.message || '镜像更新失败')
  } finally {
    updatingImage.value = false
  }
}

// 显示回滚弹窗
const showRollbackModal = async (deploy: K8sDeployment) => {
  currentDeployment.value = deploy
  selectedRevision.value = null
  revisionHistory.value = []
  rollbackModalVisible.value = true
  
  // 加载版本历史
  loadingRevisions.value = true
  try {
    const res = await k8sDeploymentApi.getRevisions(clusterId, deploy.namespace, deploy.name)
    revisionHistory.value = res.data || []
  } catch (e: any) {
    message.error(e.message || '获取版本历史失败')
  } finally {
    loadingRevisions.value = false
  }
}

// 执行回滚
const handleRollback = async () => {
  if (!currentDeployment.value || !selectedRevision.value) {
    message.warning('请选择要回滚的版本')
    return
  }
  
  rollingBack.value = true
  try {
    await k8sDeploymentApi.rollback(
      clusterId,
      currentDeployment.value.namespace,
      currentDeployment.value.name,
      selectedRevision.value
    )
    message.success('回滚成功')
    rollbackModalVisible.value = false
    fetchData()
  } catch (e: any) {
    message.error(e.message || '回滚失败')
  } finally {
    rollingBack.value = false
  }
}

// YAML 相关方法
const showYAMLModal = async (record: any, resourceType: string) => {
  currentResource.value = { name: record.name, namespace: record.namespace, type: resourceType }
  yamlModalTitle.value = `编辑 ${resourceType} - ${record.name}`
  yamlModalVisible.value = true
  loadingYAML.value = true
  try {
    const res = await k8sResourceApi.getResourceYAML(clusterId, resourceType, record.namespace, record.name)
    if (res.code === 0) {
      yamlContent.value = res.data || ''
    } else {
      message.error(res.message || '获取YAML失败')
    }
  } catch (e: any) {
    message.error(e.message || '获取YAML失败')
  } finally {
    loadingYAML.value = false
  }
}

const handleApplyYAML = async () => {
  if (!yamlContent.value.trim()) {
    message.warning('YAML内容不能为空')
    return
  }
  applyingYAML.value = true
  try {
    const res = await k8sResourceApi.applyResource(clusterId, yamlContent.value)
    if (res.code === 0) {
      message.success('应用成功')
      yamlModalVisible.value = false
      fetchData()
    } else {
      message.error(res.message || '应用失败')
    }
  } catch (e: any) {
    message.error(e.message || '应用失败')
  } finally {
    applyingYAML.value = false
  }
}

const deleteResource = async (record: any, resourceType: string) => {
  try {
    await k8sResourceApi.deleteResource(clusterId, resourceType, record.namespace, record.name)
    message.success('删除成功')
    fetchData()
  } catch (e: any) {
    message.error(e.message || '删除失败')
  }
}

// 确认删除 Deployment
const confirmDeleteDeployment = (record: any) => {
  Modal.confirm({
    title: '确认删除',
    content: `确定要删除 Deployment "${record.name}" 吗？此操作不可恢复。`,
    okText: '删除',
    okType: 'danger',
    cancelText: '取消',
    onOk: async () => {
      await deleteResource(record, 'deployment')
    }
  })
}

// 创建资源相关方法
const showCreateModal = () => {
  const menu = selectedMenu.value[0]
  const resourceType = resourceTypeMap[menu]?.type || ''
  resetFormData()
  createMode.value = 'form'
  yamlError.value = ''
  if (resourceType && selectedNamespace.value) {
    createYAMLContent.value = getResourceTemplate(resourceType, selectedNamespace.value)
  } else {
    createYAMLContent.value = ''
  }
  createModalVisible.value = true
}

// YAML 变更处理
const onYAMLChange = () => {
  yamlError.value = validateYAMLContent(createYAMLContent.value)
}

const getResourceTemplate = (type: string, namespace: string): string => {
  const templates: Record<string, string> = {
    deployment: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-deployment
  namespace: ${namespace}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: my-app
  template:
    metadata:
      labels:
        app: my-app
    spec:
      containers:
      - name: my-container
        image: nginx:latest
        ports:
        - containerPort: 80`,
    statefulset: `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: my-statefulset
  namespace: ${namespace}
spec:
  serviceName: my-statefulset
  replicas: 1
  selector:
    matchLabels:
      app: my-app
  template:
    metadata:
      labels:
        app: my-app
    spec:
      containers:
      - name: my-container
        image: nginx:latest
        ports:
        - containerPort: 80`,
    daemonset: `apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: my-daemonset
  namespace: ${namespace}
spec:
  selector:
    matchLabels:
      app: my-app
  template:
    metadata:
      labels:
        app: my-app
    spec:
      containers:
      - name: my-container
        image: nginx:latest`,
    service: `apiVersion: v1
kind: Service
metadata:
  name: my-service
  namespace: ${namespace}
spec:
  selector:
    app: my-app
  ports:
  - port: 80
    targetPort: 80
  type: ClusterIP`,
    configmap: `apiVersion: v1
kind: ConfigMap
metadata:
  name: my-configmap
  namespace: ${namespace}
data:
  key1: value1
  key2: value2`,
    secret: `apiVersion: v1
kind: Secret
metadata:
  name: my-secret
  namespace: ${namespace}
type: Opaque
stringData:
  username: admin
  password: secret`,
    ingress: `apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: my-ingress
  namespace: ${namespace}
spec:
  rules:
  - host: example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: my-service
            port:
              number: 80`,
    job: `apiVersion: batch/v1
kind: Job
metadata:
  name: my-job
  namespace: ${namespace}
spec:
  template:
    spec:
      containers:
      - name: my-job
        image: busybox
        command: ["echo", "Hello World"]
      restartPolicy: Never
  backoffLimit: 4`,
    cronjob: `apiVersion: batch/v1
kind: CronJob
metadata:
  name: my-cronjob
  namespace: ${namespace}
spec:
  schedule: "*/5 * * * *"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: my-cronjob
            image: busybox
            command: ["echo", "Hello World"]
          restartPolicy: OnFailure`,
    persistentvolumeclaim: `apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: my-pvc
  namespace: ${namespace}
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi`
  }
  return templates[type] || ''
}

const handleCreateResource = async () => {
  let yamlToApply = ''
  const menu = selectedMenu.value[0]
  
  if (createMode.value === 'form') {
    const error = validateForm(formData.value, menu)
    if (error) {
      message.warning(error)
      return
    }
    yamlToApply = buildYAMLFromForm(formData.value, selectedNamespace.value, menu)
  } else {
    if (!createYAMLContent.value.trim()) {
      message.warning('YAML内容不能为空')
      return
    }
    const validationError = validateYAMLContent(createYAMLContent.value)
    if (validationError) {
      message.warning(validationError)
      return
    }
    yamlToApply = createYAMLContent.value
  }
  
  creatingResource.value = true
  try {
    const res = await k8sResourceApi.applyResource(clusterId, yamlToApply)
    if (res.code === 0) {
      message.success('创建成功')
      createModalVisible.value = false
      fetchData()
    } else {
      message.error(res.message || '创建失败')
    }
  } catch (e: any) {
    message.error(e.message || '创建失败')
  } finally {
    creatingResource.value = false
  }
}

// 节点管理方法
const showNodeDetail = async (node: K8sNode) => {
  currentNode.value = node
  nodeDetailVisible.value = true
  try {
    const res = await k8sNodeApi.getNodeDetail(clusterId, node.name)
    if (res.code === 0) nodeDetail.value = res.data
  } catch (e: any) {
    message.error(e.message || '获取节点详情失败')
  }
}

const cordonNode = async (node: K8sNode) => {
  try {
    await k8sNodeApi.cordonNode(clusterId, node.name)
    message.success('设置成功')
    fetchData()
  } catch (e: any) {
    message.error(e.message || '设置失败')
  }
}

const uncordonNode = async (node: K8sNode) => {
  try {
    await k8sNodeApi.uncordonNode(clusterId, node.name)
    message.success('设置成功')
    fetchData()
  } catch (e: any) {
    message.error(e.message || '设置失败')
  }
}

const showTaintModal = (node: K8sNode) => {
  currentNode.value = node
  newTaint.value = { key: '', value: '', effect: 'NoSchedule' }
  taintModalVisible.value = true
}

const addTaint = async () => {
  if (!newTaint.value.key) {
    message.warning('请输入 Key')
    return
  }
  addingTaint.value = true
  try {
    await k8sNodeApi.addNodeTaint(clusterId, currentNode.value!.name, newTaint.value)
    message.success('添加成功')
    newTaint.value = { key: '', value: '', effect: 'NoSchedule' }
    fetchData()
    // 刷新当前节点数据
    const res = await k8sNodeApi.getNodes(clusterId)
    if (res.code === 0) {
      nodes.value = res.data || []
      currentNode.value = nodes.value.find(n => n.name === currentNode.value?.name) || null
    }
  } catch (e: any) {
    message.error(e.message || '添加失败')
  } finally {
    addingTaint.value = false
  }
}

const removeTaint = async (taint: { key: string; effect: string }) => {
  try {
    await k8sNodeApi.removeNodeTaint(clusterId, currentNode.value!.name, taint.key, taint.effect)
    message.success('移除成功')
    fetchData()
    // 刷新当前节点数据
    const res = await k8sNodeApi.getNodes(clusterId)
    if (res.code === 0) {
      nodes.value = res.data || []
      currentNode.value = nodes.value.find(n => n.name === currentNode.value?.name) || null
    }
  } catch (e: any) {
    message.error(e.message || '移除失败')
  }
}

// 资源详情相关方法
const showResourceDetail = (type: string, namespace: string, name: string) => {
  detailResourceType.value = type
  detailNamespace.value = namespace
  detailName.value = name
  detailVisible.value = true
}

const handleDetailNavigate = (nav: { type: string; namespace: string; name: string }) => {
  if (nav.type === 'namespace-filter') {
    // 切换命名空间过滤
    selectedNamespace.value = nav.namespace
    fetchData()
    detailVisible.value = false
  } else {
    // 跳转到其他资源详情
    const menuMap: Record<string, string> = {
      deployment: 'deployments',
      statefulset: 'statefulsets',
      daemonset: 'daemonsets',
      job: 'jobs',
      cronjob: 'cronjobs',
      pod: 'pods',
      service: 'services',
      ingress: 'ingresses',
      configmap: 'configmaps',
      secret: 'secrets',
      pvc: 'pvcs',
      pv: 'pvs',
      storageclass: 'storageclasses',
      node: 'nodes',
      namespace: 'namespaces'
    }
    const menu = menuMap[nav.type]
    if (menu) {
      selectedMenu.value = [menu]
      if (nav.namespace) {
        selectedNamespace.value = nav.namespace
      }
      fetchData()
    }
    // 打开新资源的详情
    showResourceDetail(nav.type, nav.namespace, nav.name)
  }
}

// ==================== HPA 管理 ====================
const showCreateHPAModal = () => {
  editingHPA.value = null
  hpaFormData.value = {
    name: '',
    namespace: selectedNamespace.value,
    target_kind: 'Deployment',
    target_name: '',
    min_replicas: 1,
    max_replicas: 10,
    cpu_target_percent: 80,
    mem_target_percent: undefined
  }
  hpaModalVisible.value = true
}

const editHPA = (record: any) => {
  editingHPA.value = record
  hpaFormData.value = {
    name: record.name,
    namespace: record.namespace,
    target_kind: record.target_kind,
    target_name: record.target_name,
    min_replicas: record.min_replicas,
    max_replicas: record.max_replicas,
    cpu_target_percent: 80,
    mem_target_percent: undefined
  }
  hpaModalVisible.value = true
}

const saveHPA = async () => {
  if (!hpaFormData.value.name || !hpaFormData.value.target_name) {
    message.error('请填写必要信息')
    return
  }
  savingHPA.value = true
  try {
    if (editingHPA.value) {
      await k8sHPAApi.update(clusterId, editingHPA.value.namespace, editingHPA.value.name, {
        min_replicas: hpaFormData.value.min_replicas,
        max_replicas: hpaFormData.value.max_replicas,
        cpu_target_percent: hpaFormData.value.cpu_target_percent,
        mem_target_percent: hpaFormData.value.mem_target_percent
      })
      message.success('更新成功')
    } else {
      await k8sHPAApi.create(clusterId, hpaFormData.value)
      message.success('创建成功')
    }
    hpaModalVisible.value = false
    fetchData()
  } catch (e: any) {
    message.error(e.message || '操作失败')
  } finally {
    savingHPA.value = false
  }
}

const deleteHPA = async (record: any) => {
  try {
    await k8sHPAApi.delete(clusterId, record.namespace, record.name)
    message.success('删除成功')
    fetchData()
  } catch (e: any) {
    message.error(e.message || '删除失败')
  }
}

onMounted(async () => {
  fetchClusterInfo()
  // 先获取命名空间
  await fetchNamespaces()
  // 初始化 lastNamespace，避免触发 watch
  lastNamespace = ''
  selectedNamespace.value = ''
  // 再加载资源数据
  fetchData()
})
</script>


<style scoped>
.k8s-resources {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.top-header {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  background: #fff;
  border-bottom: 1px solid #f0f0f0;
}

.cluster-name {
  font-size: 16px;
  font-weight: 500;
  margin-left: 12px;
}

.main-content {
  display: flex;
  flex: 1;
  overflow: hidden;
}

.left-menu {
  width: 200px;
  background: #fff;
  border-right: 1px solid #f0f0f0;
  overflow-y: auto;
}

.left-menu :deep(.ant-menu) {
  border-right: none;
}

.left-menu :deep(.ant-menu-item-group-title) {
  font-size: 12px;
  color: #999;
  padding: 8px 16px 4px;
}

.left-menu :deep(.ant-menu-item) {
  height: 36px;
  line-height: 36px;
  margin: 0 !important;
  padding-left: 24px !important;
}

.right-content {
  flex: 1;
  padding: 16px;
  overflow-y: auto;
  background: #f5f5f5;
}

.content-header {
  margin-bottom: 16px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.content-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 500;
}

.replica-range {
  color: #999;
  margin-left: 8px;
  font-size: 12px;
}

.more-tag {
  color: #999;
  font-size: 12px;
}

.log-header {
  display: flex;
  gap: 12px;
  margin-bottom: 12px;
}

.log-content {
  background: #1e1e1e;
  color: #d4d4d4;
  padding: 16px;
  border-radius: 4px;
  max-height: 500px;
  overflow: auto;
  font-size: 12px;
  font-family: 'Consolas', 'Monaco', monospace;
  white-space: pre-wrap;
  word-break: break-all;
}

.join-command-box {
  background: #1e1e1e;
  border-radius: 4px;
  padding: 16px;
  position: relative;
}

.join-command-box pre {
  color: #d4d4d4;
  margin: 0;
  font-size: 13px;
  font-family: 'Consolas', 'Monaco', monospace;
  white-space: pre-wrap;
  word-break: break-all;
}

.join-command-box .ant-btn-link {
  position: absolute;
  top: 8px;
  right: 8px;
  color: #fff;
}

.resource-link {
  color: #1890ff;
  cursor: pointer;
}

.resource-link:hover {
  text-decoration: underline;
}
</style>
