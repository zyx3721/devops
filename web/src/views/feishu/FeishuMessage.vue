<template>
  <div class="feishu-message">
    <div class="page-header">
      <h1>飞书管理</h1>
      <a-space>
        <a-tag color="blue">版本: {{ version || '-' }}</a-tag>
        <a-button size="small" @click="fetchVersion" :loading="loadingVersion">刷新</a-button>
        <a-button v-if="activeTab === 'apps'" type="primary" size="small" @click="showAppModal()">
          <template #icon><PlusOutlined /></template>
          添加应用
        </a-button>
        <a-button v-if="activeTab === 'bots'" type="primary" size="small" @click="showBotModal()">
          <template #icon><PlusOutlined /></template>
          添加机器人
        </a-button>
      </a-space>
    </div>

    <a-tabs v-model:activeKey="activeTab">
      <!-- 应用管理 Tab -->
      <a-tab-pane key="apps" tab="应用管理">
        <a-card :bordered="false">
          <template #extra>
            <a-space>
              <a-tag color="green">回调运行: {{ runningCallbacks.length }} 个</a-tag>
              <a-button size="small" @click="refreshCallbacks" :loading="refreshingCallbacks">
                <template #icon><ReloadOutlined /></template>
                刷新回调
              </a-button>
            </a-space>
          </template>
          <a-table :columns="appColumns" :data-source="appList" :loading="loadingApps" row-key="id" :pagination="{ pageSize: 10 }">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'app_id'">
                <a-typography-text copyable :content="record.app_id">{{ record.app_id }}</a-typography-text>
              </template>
              <template v-if="column.key === 'callback'">
                <a-tag v-if="isCallbackRunning(record.app_id)" color="green">
                  <template #icon><CheckCircleOutlined /></template>
                  已连接
                </a-tag>
                <a-tag v-else color="default">
                  <template #icon><CloseCircleOutlined /></template>
                  未连接
                </a-tag>
              </template>
              <template v-if="column.key === 'bindings'">
                <div v-if="record.bindings">
                  <a-tooltip v-for="jenkins in record.bindings.jenkins_instances" :key="'j'+jenkins.id" :title="jenkins.url">
                    <a-tag color="orange" style="margin: 2px;">Jenkins: {{ jenkins.name }}</a-tag>
                  </a-tooltip>
                  <a-tag v-for="k8s in record.bindings.k8s_clusters" :key="'k'+k8s.id" color="blue" style="margin: 2px;">K8s: {{ k8s.name }}</a-tag>
                  <span v-if="!record.bindings.jenkins_instances?.length && !record.bindings.k8s_clusters?.length" style="color: #999">-</span>
                </div>
                <span v-else style="color: #999">-</span>
              </template>
              <template v-if="column.key === 'webhook'">
                <a-typography-text v-if="record.webhook" copyable :content="record.webhook" ellipsis style="max-width: 200px">{{ record.webhook }}</a-typography-text>
                <span v-else>-</span>
              </template>
              <template v-if="column.key === 'status'">
                <a-badge :status="record.status === 'active' ? 'success' : 'default'" :text="record.status === 'active' ? '启用' : '禁用'" />
              </template>
              <template v-if="column.key === 'is_default'">
                <a-tag v-if="record.is_default" color="blue">默认</a-tag>
                <a-button v-else type="link" size="small" @click="setDefaultApp(record.id)">设为默认</a-button>
              </template>
              <template v-if="column.key === 'action'">
                <a-space>
                  <a-button type="link" size="small" @click="showAppModal(record)">编辑</a-button>
                  <a-popconfirm title="确定删除？" @confirm="() => deleteApp(record.id)" :disabled="!record.id">
                    <a-button type="link" size="small" danger :disabled="record.is_default">删除</a-button>
                  </a-popconfirm>
                </a-space>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>

      <!-- 机器人管理 Tab -->
      <a-tab-pane key="bots" tab="机器人管理">
        <a-card :bordered="false">
          <a-table :columns="botColumns" :data-source="botList" :loading="loadingBots" row-key="id" :pagination="{ pageSize: 10 }">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'webhook_url'">
                <a-typography-text copyable :content="record.webhook_url" ellipsis style="max-width: 300px">{{ record.webhook_url }}</a-typography-text>
              </template>
              <template v-if="column.key === 'status'">
                <a-badge :status="record.status === 'active' ? 'success' : 'default'" :text="record.status === 'active' ? '启用' : '禁用'" />
              </template>
              <template v-if="column.key === 'action'">
                <a-space>
                  <a-button type="link" size="small" @click="showBotModal(record)">编辑</a-button>
                  <a-popconfirm title="确定删除？" @confirm="() => deleteBot(record.id)" :disabled="!record.id">
                    <a-button type="link" size="small" danger>删除</a-button>
                  </a-popconfirm>
                </a-space>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>

      <!-- 发送消息 Tab -->
      <a-tab-pane key="message" tab="发送消息">
        <a-row :gutter="24">
          <a-col :xs="24" :lg="12">
            <a-card title="消息配置" :bordered="false">
              <a-form :model="messageForm" layout="vertical">
                <a-row :gutter="16">
                  <a-col :span="12">
                    <a-form-item label="接收者ID" required><a-input v-model:value="messageForm.receive_id" placeholder="请输入接收者ID" /></a-form-item>
                  </a-col>
                  <a-col :span="12">
                    <a-form-item label="ID类型" required>
                      <a-select v-model:value="messageForm.receive_id_type" style="width: 100%">
                        <a-select-option value="open_id">Open ID</a-select-option>
                        <a-select-option value="user_id">User ID</a-select-option>
                        <a-select-option value="chat_id">Chat ID (群组)</a-select-option>
                      </a-select>
                    </a-form-item>
                  </a-col>
                </a-row>
                <a-form-item label="消息类型" required>
                  <a-radio-group v-model:value="messageForm.msg_type" button-style="solid">
                    <a-radio-button value="text">文本</a-radio-button>
                    <a-radio-button value="post">富文本</a-radio-button>
                    <a-radio-button value="interactive">卡片</a-radio-button>
                  </a-radio-group>
                </a-form-item>
                <a-form-item label="消息内容" required><a-textarea v-model:value="messageForm.content" placeholder="请输入消息内容" :rows="6" /></a-form-item>
                <a-form-item>
                  <a-button type="primary" @click="sendMessage" :loading="sendingMessage" block>
                    <template #icon><SendOutlined /></template>发送消息
                  </a-button>
                </a-form-item>
              </a-form>
            </a-card>
          </a-col>
          <a-col :xs="24" :lg="12">
            <a-card title="使用说明" :bordered="false">
              <a-list size="small" :data-source="idTypeHelp" :split="false">
                <template #renderItem="{ item }">
                  <a-list-item><a-typography-text code>{{ item.type }}</a-typography-text><span style="margin-left: 8px">{{ item.desc }}</span></a-list-item>
                </template>
              </a-list>
            </a-card>
          </a-col>
        </a-row>
      </a-tab-pane>

      <!-- 发布卡片 Tab -->
      <a-tab-pane key="card" tab="发布卡片">
        <a-row :gutter="24">
          <a-col :xs="24" :lg="10">
            <a-card title="基本配置" :bordered="false">
              <a-form :model="cardForm" layout="vertical">
                <a-row :gutter="16">
                  <a-col :span="12"><a-form-item label="接收者ID" required><a-input v-model:value="cardForm.receive_id" placeholder="群组或用户ID" /></a-form-item></a-col>
                  <a-col :span="12">
                    <a-form-item label="ID类型" required>
                      <a-select v-model:value="cardForm.receive_id_type" style="width: 100%">
                        <a-select-option value="chat_id">Chat ID (群组)</a-select-option>
                        <a-select-option value="open_id">Open ID</a-select-option>
                      </a-select>
                    </a-form-item>
                  </a-col>
                </a-row>
                <a-form-item label="卡片标题"><a-input v-model:value="cardForm.title" placeholder="应用发布申请" /></a-form-item>
              </a-form>
            </a-card>
          </a-col>
          <a-col :xs="24" :lg="14">
            <a-card :bordered="false">
              <template #title><span>服务配置</span><a-button type="link" size="small" @click="addService" style="float: right"><template #icon><PlusOutlined /></template>添加服务</a-button></template>
              <a-empty v-if="cardForm.services.length === 0" description="暂无服务配置"><a-button type="primary" @click="addService">添加第一个服务</a-button></a-empty>
              <div v-else class="service-list">
                <div v-for="(service, index) in cardForm.services" :key="index" class="service-item">
                  <div class="service-header">
                    <span class="service-title">服务 {{ index + 1 }}</span>
                    <a-button type="text" danger size="small" @click="removeService(index)" v-if="cardForm.services.length > 1"><template #icon><DeleteOutlined /></template></a-button>
                  </div>
                  <a-row :gutter="12">
                    <a-col :span="8"><a-input v-model:value="service.name" placeholder="服务名称" /></a-col>
                    <a-col :span="8"><a-input v-model:value="service.object_id" placeholder="分支" /></a-col>
                    <a-col :span="8">
                      <a-select v-model:value="service.actions" mode="multiple" placeholder="操作" style="width: 100%">
                        <a-select-option value="gray">灰度</a-select-option>
                        <a-select-option value="official">正式</a-select-option>
                        <a-select-option value="rollback">回滚</a-select-option>
                        <a-select-option value="restart">重启</a-select-option>
                      </a-select>
                    </a-col>
                  </a-row>
                </div>
              </div>
              <a-divider />
              <a-button type="primary" @click="sendCard" :loading="sendingCard" block :disabled="cardForm.services.length === 0">
                <template #icon><SendOutlined /></template>发送发布卡片
              </a-button>
            </a-card>
          </a-col>
        </a-row>
      </a-tab-pane>

      <!-- 发送记录 Tab -->
      <a-tab-pane key="logs" tab="发送记录">
        <a-card :bordered="false">
          <template #extra>
            <a-space>
              <a-select v-model:value="logFilter.source" placeholder="来源" style="width: 120px" allow-clear @change="fetchLogs">
                <a-select-option value="manual">手动发送</a-select-option>
                <a-select-option value="oa_sync">OA同步</a-select-option>
              </a-select>
              <a-select v-model:value="logFilter.msg_type" placeholder="类型" style="width: 120px" allow-clear @change="fetchLogs">
                <a-select-option value="text">文本</a-select-option>
                <a-select-option value="interactive">卡片</a-select-option>
              </a-select>
              <a-button @click="fetchLogs"><template #icon><ReloadOutlined /></template></a-button>
            </a-space>
          </template>
          <a-table :columns="logColumns" :data-source="logList" :loading="loadingLogs" row-key="id" :pagination="logPagination" @change="onLogTableChange">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'msg_type'">
                <a-tag :color="record.msg_type === 'interactive' ? 'blue' : 'default'">{{ record.msg_type === 'interactive' ? '卡片' : record.msg_type }}</a-tag>
              </template>
              <template v-if="column.key === 'source'">
                <a-tag :color="record.source === 'oa_sync' ? 'green' : 'default'">{{ record.source === 'oa_sync' ? 'OA同步' : '手动' }}</a-tag>
              </template>
              <template v-if="column.key === 'status'">
                <a-badge :status="record.status === 'success' ? 'success' : 'error'" :text="record.status === 'success' ? '成功' : '失败'" />
              </template>
              <template v-if="column.key === 'created_at'">{{ formatTime(record.created_at) }}</template>
              <template v-if="column.key === 'action'">
                <a-button type="link" size="small" @click="viewLogDetail(record)">详情</a-button>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>

      <!-- 用户搜索 Tab -->
      <a-tab-pane key="users" tab="用户搜索">
        <a-card :bordered="false">
          <template #extra>
            <a-space>
              <a-tag v-if="userTokenStatus.is_valid" color="green">User Token 正常</a-tag>
              <a-tag v-else-if="userTokenStatus.has_token" color="red">
                <template #icon><ExclamationCircleOutlined /></template>
                Token 已过期
              </a-tag>
              <a-tag v-else color="orange">User Token 未配置</a-tag>
              <span v-if="userTokenStatus.expires_at" style="color: #999; font-size: 12px">过期时间: {{ userTokenStatus.expires_at }}</span>
              <a-button size="small" @click="showUserTokenModal">
                <template #icon><SettingOutlined /></template>
                配置 Token
              </a-button>
              <a-button size="small" type="primary" @click="authorizeFeishu">
                <template #icon><LinkOutlined /></template>
                飞书授权
              </a-button>
            </a-space>
          </template>
          <a-alert v-if="userTokenStatus.has_token && !userTokenStatus.is_valid" type="error" show-icon style="margin-bottom: 16px">
            <template #message>User Token 已过期，请点击"飞书授权"重新授权以恢复姓名搜索功能。</template>
          </a-alert>
          <a-alert v-else-if="!userTokenStatus.has_token" type="warning" show-icon style="margin-bottom: 16px">
            <template #message>未配置 User Token，只能通过邮箱或手机号搜索。点击"飞书授权"或"配置 Token"以启用姓名搜索。</template>
          </a-alert>
          <a-row :gutter="16" style="margin-bottom: 16px">
            <a-col :span="16">
              <a-input-search v-model:value="userSearchQuery" placeholder="输入用户名、邮箱或手机号搜索" enter-button="搜索" @search="searchUsers" :loading="searchingUsers" />
            </a-col>
          </a-row>
          <a-table :columns="userColumns" :data-source="userList" :loading="searchingUsers" row-key="user_id" :pagination="false">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'avatar'">
                <a-avatar :src="record.avatar?.avatar_72" size="small">{{ record.name?.charAt(0) }}</a-avatar>
              </template>
              <template v-if="column.key === 'user_id'">
                <a-typography-text copyable :content="record.user_id">{{ record.user_id }}</a-typography-text>
              </template>
              <template v-if="column.key === 'open_id'">
                <a-typography-text copyable :content="record.open_id">{{ record.open_id }}</a-typography-text>
              </template>
              <template v-if="column.key === 'action'">
                <a-space>
                  <a-button type="link" size="small" @click="selectUserForChat(record)">选择</a-button>
                </a-space>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>

      <!-- 群聊管理 Tab -->
      <a-tab-pane key="chats" tab="群聊管理">
        <a-card :bordered="false">
          <template #extra>
            <a-space>
              <a-button type="primary" size="small" @click="showCreateChatModal">
                <template #icon><PlusOutlined /></template>
                创建群聊
              </a-button>
              <a-button size="small" @click="fetchChats"><template #icon><ReloadOutlined /></template></a-button>
            </a-space>
          </template>
          <a-table :columns="chatColumns" :data-source="chatList" :loading="loadingChats" row-key="chat_id" :pagination="false">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'chat_id'">
                <a-typography-text copyable :content="record.chat_id">{{ record.chat_id }}</a-typography-text>
              </template>
              <template v-if="column.key === 'chat_type'">
                <a-tag>{{ record.chat_type === 'group' ? '群组' : record.chat_type }}</a-tag>
              </template>
              <template v-if="column.key === 'action'">
                <a-space>
                  <a-button type="link" size="small" @click="showAddMembersModal(record)">添加成员</a-button>
                </a-space>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>
    </a-tabs>

    <!-- 应用编辑弹窗 -->
    <a-modal v-model:open="appModalVisible" :title="editingAppId ? '编辑应用' : '添加应用'" @ok="saveApp" :confirm-loading="savingApp" width="600px">
      <a-form :model="editingApp" layout="vertical">
        <a-row :gutter="16">
          <a-col :span="12"><a-form-item label="应用名称" required><a-input v-model:value="editingApp.name" placeholder="我的飞书应用" /></a-form-item></a-col>
          <a-col :span="12"><a-form-item label="项目" required><a-input v-model:value="editingApp.project" placeholder="项目标识" /></a-form-item></a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="12"><a-form-item label="App ID" required><a-input v-model:value="editingApp.app_id" placeholder="cli_xxxxxxxx" /></a-form-item></a-col>
          <a-col :span="12"><a-form-item label="App Secret" required><a-input-password v-model:value="editingApp.app_secret" placeholder="应用密钥" /></a-form-item></a-col>
        </a-row>
        <a-form-item label="Webhook 地址"><a-input v-model:value="editingApp.webhook" placeholder="https://open.feishu.cn/open-apis/bot/v2/hook/xxx" /></a-form-item>
        <a-form-item label="描述"><a-textarea v-model:value="editingApp.description" :rows="2" /></a-form-item>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="状态">
              <a-select v-model:value="editingApp.status" style="width: 100%">
                <a-select-option value="active">启用</a-select-option>
                <a-select-option value="inactive">禁用</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="12"><a-form-item label=" "><a-checkbox v-model:checked="editingApp.is_default">设为默认应用</a-checkbox></a-form-item></a-col>
        </a-row>
      </a-form>
    </a-modal>

    <!-- 机器人编辑弹窗 -->
    <a-modal v-model:open="botModalVisible" :title="editingBotId ? '编辑机器人' : '添加机器人'" @ok="saveBot" :confirm-loading="savingBot" width="600px">
      <a-form :model="editingBot" layout="vertical">
        <a-form-item label="机器人名称" required><a-input v-model:value="editingBot.name" placeholder="我的机器人" /></a-form-item>
        <a-form-item label="Webhook URL" required><a-input v-model:value="editingBot.webhook_url" placeholder="https://open.feishu.cn/open-apis/bot/v2/hook/xxx" /></a-form-item>
        <a-form-item label="关联项目"><a-input v-model:value="editingBot.project" placeholder="项目名称，用于标识告警来源" /></a-form-item>
        <a-form-item label="签名密钥"><a-input-password v-model:value="editingBot.secret" placeholder="可选，用于签名验证" /></a-form-item>
        <a-form-item label="描述"><a-textarea v-model:value="editingBot.description" :rows="2" /></a-form-item>
        <a-form-item label="状态">
          <a-select v-model:value="editingBot.status" style="width: 100%">
            <a-select-option value="active">启用</a-select-option>
            <a-select-option value="inactive">禁用</a-select-option>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>
    <!-- 日志详情抽屉 -->
    <a-drawer v-model:open="logDetailVisible" title="发送详情" placement="right" :width="600">
      <a-descriptions :column="1" bordered size="small">
        <a-descriptions-item label="发送时间">{{ formatTime(currentLog?.created_at) }}</a-descriptions-item>
        <a-descriptions-item label="消息类型">
          <a-tag :color="currentLog?.msg_type === 'interactive' ? 'blue' : 'default'">{{ currentLog?.msg_type === 'interactive' ? '卡片' : currentLog?.msg_type }}</a-tag>
        </a-descriptions-item>
        <a-descriptions-item label="来源">
          <a-tag :color="currentLog?.source === 'oa_sync' ? 'green' : 'default'">{{ currentLog?.source === 'oa_sync' ? 'OA同步' : '手动' }}</a-tag>
        </a-descriptions-item>
        <a-descriptions-item label="接收者ID">{{ currentLog?.receive_id }}</a-descriptions-item>
        <a-descriptions-item label="ID类型">{{ currentLog?.receive_id_type }}</a-descriptions-item>
        <a-descriptions-item label="标题">{{ currentLog?.title || '-' }}</a-descriptions-item>
        <a-descriptions-item label="状态">
          <a-badge :status="currentLog?.status === 'success' ? 'success' : 'error'" :text="currentLog?.status === 'success' ? '成功' : '失败'" />
        </a-descriptions-item>
        <a-descriptions-item v-if="currentLog?.error_msg" label="错误信息">
          <a-typography-text type="danger">{{ currentLog?.error_msg }}</a-typography-text>
        </a-descriptions-item>
      </a-descriptions>
      <a-divider orientation="left">消息内容</a-divider>
      <pre class="json-preview">{{ formatJsonContent(currentLog?.content) }}</pre>
    </a-drawer>

    <!-- 创建群聊弹窗 -->
    <a-modal v-model:open="createChatModalVisible" title="创建群聊" @ok="createChat" :confirm-loading="creatingChat" width="500px">
      <a-form :model="newChatForm" layout="vertical">
        <a-form-item label="群名称" required><a-input v-model:value="newChatForm.name" placeholder="请输入群名称" /></a-form-item>
        <a-form-item label="群描述"><a-textarea v-model:value="newChatForm.description" placeholder="请输入群描述" :rows="2" /></a-form-item>
        <a-form-item label="初始成员 (User ID)">
          <a-select v-model:value="newChatForm.userIds" mode="tags" placeholder="输入 User ID 后回车添加" style="width: 100%" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 添加群成员弹窗 -->
    <a-modal v-model:open="addMembersModalVisible" title="添加群成员" @ok="addChatMembers" :confirm-loading="addingMembers" width="500px">
      <a-form layout="vertical">
        <a-form-item label="群聊">
          <a-input :value="currentChat?.name" disabled />
        </a-form-item>
        <a-form-item label="成员 (User ID)" required>
          <a-select v-model:value="addMembersForm.userIds" mode="tags" placeholder="输入 User ID 后回车添加" style="width: 100%" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- User Token 配置弹窗 -->
    <a-modal v-model:open="userTokenModalVisible" title="配置 User Token" @ok="saveUserToken" :confirm-loading="savingUserToken" width="600px">
      <a-alert type="info" show-icon style="margin-bottom: 16px">
        <template #message>
          User Token 用于支持按姓名搜索用户。配置 refresh_token 后系统会每小时自动刷新。
        </template>
      </a-alert>
      <a-form :model="userTokenForm" layout="vertical">
        <a-form-item label="关联应用" required>
          <a-select v-model:value="userTokenForm.app_id" placeholder="选择要配置 Token 的应用" style="width: 100%">
            <a-select-option v-for="app in appList" :key="app.app_id" :value="app.app_id">
              {{ app.name }} ({{ app.app_id }})
            </a-select-option>
          </a-select>
          <template #extra>不同应用可以配置不同的 User Token</template>
        </a-form-item>
        <a-form-item label="User Access Token" required>
          <a-input v-model:value="userTokenForm.user_token" placeholder="u-xxx" />
          <template #extra>从飞书开放平台 API 调试工具获取</template>
        </a-form-item>
        <a-form-item label="Refresh Token (可选，用于自动刷新)">
          <a-input v-model:value="userTokenForm.refresh_token" placeholder="ur-xxx" />
          <template #extra>配置后系统会每小时自动刷新 token，有效期约 30 天</template>
        </a-form-item>
      </a-form>
      <a-divider />
      <p>当前状态：
        <a-tag v-if="userTokenStatus.has_token" color="green">已配置</a-tag>
        <a-tag v-else color="default">未配置</a-tag>
        <span v-if="userTokenStatus.refresh_token" style="margin-left: 8px; color: #999">{{ userTokenStatus.refresh_token }}</span>
      </p>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { SendOutlined, PlusOutlined, DeleteOutlined, ReloadOutlined, CheckCircleOutlined, CloseCircleOutlined, SettingOutlined, LinkOutlined, ExclamationCircleOutlined } from '@ant-design/icons-vue'
import { feishuApi, feishuAppApi, feishuBotApi, feishuCallbackApi, feishuLogApi, feishuUserApi, feishuChatApi, type FeishuApp, type FeishuBot, type FeishuMessageLog, type FeishuUser, type FeishuChat } from '@/services/feishu'

const activeTab = ref('apps')
const sendingMessage = ref(false)
const sendingCard = ref(false)
const loadingVersion = ref(false)
const loadingApps = ref(false)
const loadingBots = ref(false)
const loadingLogs = ref(false)
const loadingChats = ref(false)
const savingApp = ref(false)
const savingBot = ref(false)
const refreshingCallbacks = ref(false)
const searchingUsers = ref(false)
const creatingChat = ref(false)
const addingMembers = ref(false)
const savingUserToken = ref(false)
const appModalVisible = ref(false)
const botModalVisible = ref(false)
const logDetailVisible = ref(false)
const createChatModalVisible = ref(false)
const addMembersModalVisible = ref(false)
const userTokenModalVisible = ref(false)
const version = ref('')
const appList = ref<FeishuApp[]>([])
const botList = ref<FeishuBot[]>([])
const logList = ref<FeishuMessageLog[]>([])
const userList = ref<FeishuUser[]>([])
const chatList = ref<FeishuChat[]>([])
const currentLog = ref<FeishuMessageLog | null>(null)
const currentChat = ref<FeishuChat | null>(null)
const runningCallbacks = ref<string[]>([])
const editingAppId = ref<number | undefined>(undefined)
const editingBotId = ref<number | undefined>(undefined)
const userSearchQuery = ref('')
const editingApp = reactive<Omit<FeishuApp, 'id'>>({ name: '', app_id: '', app_secret: '', webhook: '', project: '', description: '', status: 'active', is_default: false })
const editingBot = reactive<Omit<FeishuBot, 'id'>>({ name: '', webhook_url: '', project: '', secret: '', description: '', status: 'active' })
const logFilter = reactive({ source: '', msg_type: '' })
const logPagination = reactive({ current: 1, pageSize: 20, total: 0, showSizeChanger: true, showTotal: (total: number) => `共 ${total} 条` })
const newChatForm = reactive({ name: '', description: '', userIds: [] as string[] })
const addMembersForm = reactive({ userIds: [] as string[] })
const userTokenForm = reactive({ app_id: '', user_token: '', refresh_token: '' })
const userTokenStatus = reactive({ has_token: false, is_valid: false, refresh_token: '', expires_at: '', status: '' })

const idTypeHelp = [{ type: 'open_id', desc: '用户在应用内的唯一标识' }, { type: 'chat_id', desc: '群组的唯一标识' }]

const appColumns = [
  { title: '名称', dataIndex: 'name', key: 'name', width: 120 },
  { title: 'App ID', dataIndex: 'app_id', key: 'app_id', width: 180 },
  { title: '回调状态', key: 'callback', width: 100 },
  { title: '绑定资源', key: 'bindings', width: 200 },
  { title: '项目', dataIndex: 'project', key: 'project', width: 100 },
  { title: '状态', dataIndex: 'status', key: 'status', width: 80 },
  { title: '默认', dataIndex: 'is_default', key: 'is_default', width: 100 },
  { title: '操作', key: 'action', width: 120 }
]

const botColumns = [
  { title: '名称', dataIndex: 'name', key: 'name', width: 150 },
  { title: 'Webhook URL', dataIndex: 'webhook_url', key: 'webhook_url', ellipsis: true },
  { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
  { title: '状态', dataIndex: 'status', key: 'status', width: 80 },
  { title: '操作', key: 'action', width: 120 }
]

const logColumns = [
  { title: '时间', dataIndex: 'created_at', key: 'created_at', width: 180 },
  { title: '类型', dataIndex: 'msg_type', key: 'msg_type', width: 80 },
  { title: '来源', dataIndex: 'source', key: 'source', width: 100 },
  { title: '接收者', dataIndex: 'receive_id', key: 'receive_id', width: 180, ellipsis: true },
  { title: '标题', dataIndex: 'title', key: 'title', ellipsis: true },
  { title: '状态', dataIndex: 'status', key: 'status', width: 80 },
  { title: '操作', key: 'action', width: 80 }
]

const userColumns = [
  { title: '', key: 'avatar', width: 50 },
  { title: '姓名', dataIndex: 'name', key: 'name', width: 100 },
  { title: 'User ID', dataIndex: 'user_id', key: 'user_id', width: 180 },
  { title: 'Open ID', dataIndex: 'open_id', key: 'open_id', width: 200 },
  { title: '邮箱', dataIndex: 'email', key: 'email', ellipsis: true },
  { title: '操作', key: 'action', width: 80 }
]

const chatColumns = [
  { title: '群名称', dataIndex: 'name', key: 'name', width: 200 },
  { title: 'Chat ID', dataIndex: 'chat_id', key: 'chat_id', width: 250 },
  { title: '类型', dataIndex: 'chat_type', key: 'chat_type', width: 100 },
  { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
  { title: '操作', key: 'action', width: 120 }
]

const messageForm = reactive({ receive_id: '', receive_id_type: 'chat_id' as const, msg_type: 'text' as const, content: '' })
const cardForm = reactive({ receive_id: '', receive_id_type: 'chat_id' as const, title: '应用发布申请', services: [{ name: '', object_id: '', actions: [] as string[] }] })

const isCallbackRunning = (appId: string) => runningCallbacks.value.includes(appId)

const fetchCallbackStatus = async () => {
  try {
    const response = await feishuCallbackApi.getStatus()
    if (response.code === 0 && response.data) { runningCallbacks.value = response.data.running_apps || [] }
  } catch (error: any) { console.error('获取回调状态失败', error) }
}

const refreshCallbacks = async () => {
  refreshingCallbacks.value = true
  try {
    const response = await feishuCallbackApi.refresh()
    if (response.code === 0) {
      message.success('回调已刷新')
      if (response.data) { runningCallbacks.value = response.data.running_apps || [] }
    } else { message.error(response.message || '刷新失败') }
  } catch (error: any) { message.error(error.message || '刷新失败') }
  finally { refreshingCallbacks.value = false }
}

const fetchApps = async () => {
  loadingApps.value = true
  try {
    const response = await feishuAppApi.list()
    if (response.code === 0 && response.data) {
      const apps = (response.data.list || []).map((item: any) => ({ ...item, id: item.id || item.ID, bindings: null }))
      // 获取每个应用的绑定信息
      for (const app of apps) {
        if (app.id) {
          try {
            const bindingsRes = await feishuAppApi.getBindings(app.id)
            if (bindingsRes.code === 0 && bindingsRes.data) {
              app.bindings = bindingsRes.data
            }
          } catch {}
        }
      }
      appList.value = apps
    }
  } catch (error: any) { console.error('获取应用列表失败', error) }
  finally { loadingApps.value = false }
}

const showAppModal = (app?: FeishuApp) => {
  if (app) {
    editingAppId.value = app.id; editingApp.name = app.name; editingApp.app_id = app.app_id; editingApp.app_secret = app.app_secret
    editingApp.webhook = app.webhook || ''; editingApp.project = app.project || ''; editingApp.description = app.description || ''
    editingApp.status = app.status; editingApp.is_default = app.is_default
  } else {
    editingAppId.value = undefined; editingApp.name = ''; editingApp.app_id = ''; editingApp.app_secret = ''
    editingApp.webhook = ''; editingApp.project = ''; editingApp.description = ''; editingApp.status = 'active'; editingApp.is_default = false
  }
  appModalVisible.value = true
}

const saveApp = async () => {
  if (!editingApp.name || !editingApp.app_id || !editingApp.app_secret || !editingApp.project) { message.error('请填写必填项'); return }
  savingApp.value = true
  try {
    const data = { ...editingApp }
    const response = editingAppId.value ? await feishuAppApi.update(editingAppId.value, data) : await feishuAppApi.create(data)
    if (response.code === 0) { message.success(editingAppId.value ? '更新成功' : '添加成功'); appModalVisible.value = false; fetchApps() }
    else { message.error(response.message || '保存失败') }
  } catch (error: any) { message.error(error.message || '保存失败') }
  finally { savingApp.value = false }
}

const deleteApp = async (id: number | undefined) => {
  if (!id) return
  try {
    const response = await feishuAppApi.delete(id)
    if (response.code === 0) { message.success('删除成功'); fetchApps() }
    else { message.error(response.message || '删除失败') }
  } catch (error: any) { message.error(error.message || '删除失败') }
}

const setDefaultApp = async (id: number) => {
  try {
    const response = await feishuAppApi.setDefault(id)
    if (response.code === 0) { message.success('已设为默认'); fetchApps() }
    else { message.error(response.message || '设置失败') }
  } catch (error: any) { message.error(error.message || '设置失败') }
}

const fetchBots = async () => {
  loadingBots.value = true
  try {
    const response = await feishuBotApi.list()
    if (response.code === 0 && response.data) { botList.value = (response.data.list || []).map((item: any) => ({ ...item, id: item.id || item.ID })) }
  } catch (error: any) { console.error('获取机器人列表失败', error) }
  finally { loadingBots.value = false }
}

const showBotModal = (bot?: FeishuBot) => {
  if (bot) {
    editingBotId.value = bot.id; editingBot.name = bot.name; editingBot.webhook_url = bot.webhook_url
    editingBot.project = bot.project || ''; editingBot.secret = bot.secret || ''; editingBot.description = bot.description || ''; editingBot.status = bot.status
  } else {
    editingBotId.value = undefined; editingBot.name = ''; editingBot.webhook_url = ''; editingBot.project = ''; editingBot.secret = ''; editingBot.description = ''; editingBot.status = 'active'
  }
  botModalVisible.value = true
}

const saveBot = async () => {
  if (!editingBot.name || !editingBot.webhook_url) { message.error('请填写必填项'); return }
  savingBot.value = true
  try {
    const data = { ...editingBot }
    const response = editingBotId.value ? await feishuBotApi.update(editingBotId.value, data) : await feishuBotApi.create(data)
    if (response.code === 0) { message.success(editingBotId.value ? '更新成功' : '添加成功'); botModalVisible.value = false; fetchBots() }
    else { message.error(response.message || '保存失败') }
  } catch (error: any) { message.error(error.message || '保存失败') }
  finally { savingBot.value = false }
}

const deleteBot = async (id: number | undefined) => {
  if (!id) return
  try {
    const response = await feishuBotApi.delete(id)
    if (response.code === 0) { message.success('删除成功'); fetchBots() }
    else { message.error(response.message || '删除失败') }
  } catch (error: any) { message.error(error.message || '删除失败') }
}

const addService = () => { cardForm.services.push({ name: '', object_id: '', actions: [] }) }
const removeService = (index: number) => { cardForm.services.splice(index, 1) }

const sendMessage = async () => {
  if (!messageForm.receive_id || !messageForm.content) { message.error('请填写必填项'); return }
  sendingMessage.value = true
  try {
    const response = await feishuApi.sendMessage(messageForm)
    if (response.code === 0) { message.success('消息发送成功'); messageForm.content = '' }
    else { message.error(response.message || '发送失败') }
  } catch (error: any) { message.error(error.message || '发送失败') }
  finally { sendingMessage.value = false }
}

const sendCard = async () => {
  if (!cardForm.receive_id) { message.error('请填写接收者ID'); return }
  const validServices = cardForm.services.filter(s => s.name && s.object_id)
  if (validServices.length === 0) { message.error('请至少配置一个有效的服务'); return }
  sendingCard.value = true
  try {
    const response = await feishuApi.sendCard({ receive_id: cardForm.receive_id, receive_id_type: cardForm.receive_id_type, card_data: { title: cardForm.title, services: validServices } })
    if (response.code === 0) { message.success('卡片发送成功') }
    else { message.error(response.message || '发送失败') }
  } catch (error: any) { message.error(error.message || '发送失败') }
  finally { sendingCard.value = false }
}

const fetchVersion = async () => {
  loadingVersion.value = true
  try {
    const response = await feishuApi.getVersion()
    if (response.code === 0 && response.data) { version.value = response.data.version }
  } catch (error: any) { console.error('获取版本失败', error) }
  finally { loadingVersion.value = false }
}

const fetchLogs = async () => {
  loadingLogs.value = true
  try {
    const response = await feishuLogApi.list(logPagination.current, logPagination.pageSize, logFilter.msg_type, logFilter.source)
    if (response.code === 0 && response.data) {
      logList.value = response.data.list || []
      logPagination.total = response.data.total
    }
  } catch (error: any) { console.error('获取日志失败', error) }
  finally { loadingLogs.value = false }
}

const onLogTableChange = (pagination: any) => {
  logPagination.current = pagination.current
  logPagination.pageSize = pagination.pageSize
  fetchLogs()
}

const viewLogDetail = (log: FeishuMessageLog) => {
  currentLog.value = log
  logDetailVisible.value = true
}

const formatTime = (time: string | undefined) => time ? time.replace('T', ' ').substring(0, 19) : '-'

const formatJsonContent = (content: string | undefined) => {
  if (!content) return ''
  try { return JSON.stringify(JSON.parse(content), null, 2) }
  catch { return content }
}

// 用户搜索
const searchUsers = async () => {
  if (!userSearchQuery.value) { message.warning('请输入搜索关键词'); return }
  searchingUsers.value = true
  try {
    const response = await feishuUserApi.search(userSearchQuery.value)
    if (response.code === 0) { userList.value = response.data || [] }
    else { message.error(response.message || '搜索失败') }
  } catch (error: any) { message.error(error.message || '搜索失败') }
  finally { searchingUsers.value = false }
}

const selectUserForChat = (user: FeishuUser) => {
  if (user.user_id) {
    newChatForm.userIds.push(user.user_id)
    message.success(`已添加用户: ${user.name}`)
  }
}

// 群聊管理
const fetchChats = async () => {
  loadingChats.value = true
  try {
    const response = await feishuChatApi.list()
    if (response.code === 0 && response.data) { chatList.value = response.data.list || [] }
  } catch (error: any) { console.error('获取群列表失败', error) }
  finally { loadingChats.value = false }
}

const showCreateChatModal = () => {
  newChatForm.name = ''
  newChatForm.description = ''
  newChatForm.userIds = []
  createChatModalVisible.value = true
}

const createChat = async () => {
  if (!newChatForm.name) { message.error('请输入群名称'); return }
  creatingChat.value = true
  try {
    const response = await feishuChatApi.create(newChatForm.name, newChatForm.description, newChatForm.userIds)
    if (response.code === 0) {
      message.success('群聊创建成功')
      createChatModalVisible.value = false
      fetchChats()
    } else { message.error(response.message || '创建失败') }
  } catch (error: any) { message.error(error.message || '创建失败') }
  finally { creatingChat.value = false }
}

const showAddMembersModal = (chat: FeishuChat) => {
  currentChat.value = chat
  addMembersForm.userIds = []
  addMembersModalVisible.value = true
}

const addChatMembers = async () => {
  if (!currentChat.value || addMembersForm.userIds.length === 0) { message.error('请输入成员ID'); return }
  addingMembers.value = true
  try {
    const response = await feishuChatApi.addMembers(currentChat.value.chat_id, addMembersForm.userIds)
    if (response.code === 0) {
      message.success('成员添加成功')
      addMembersModalVisible.value = false
    } else { message.error(response.message || '添加失败') }
  } catch (error: any) { message.error(error.message || '添加失败') }
  finally { addingMembers.value = false }
}

// User Token 管理
const fetchUserTokenStatus = async (appId?: string) => {
  try {
    const response = await feishuUserApi.getTokenStatus(appId)
    if (response.code === 0 && response.data) {
      userTokenStatus.has_token = response.data.has_token
      userTokenStatus.is_valid = response.data.is_valid || false
      userTokenStatus.refresh_token = response.data.refresh_token || ''
      userTokenStatus.expires_at = response.data.expires_at || ''
      userTokenStatus.status = response.data.status || ''
    }
  } catch (error: any) { console.error('获取 Token 状态失败', error) }
}

const showUserTokenModal = () => {
  // 默认选择第一个应用或默认应用
  const defaultApp = appList.value.find(a => a.is_default) || appList.value[0]
  userTokenForm.app_id = defaultApp?.app_id || ''
  userTokenForm.user_token = ''
  userTokenForm.refresh_token = ''
  userTokenModalVisible.value = true
  // 获取该应用的 token 状态
  if (userTokenForm.app_id) {
    fetchUserTokenStatus(userTokenForm.app_id)
  }
}

const saveUserToken = async () => {
  if (!userTokenForm.app_id) { message.error('请选择关联应用'); return }
  if (!userTokenForm.user_token) { message.error('请填写 User Access Token'); return }
  savingUserToken.value = true
  try {
    const response = await feishuUserApi.setToken(userTokenForm.app_id, userTokenForm.user_token, userTokenForm.refresh_token)
    if (response.code === 0) {
      message.success('Token 配置成功')
      userTokenModalVisible.value = false
      fetchUserTokenStatus(userTokenForm.app_id)
    } else { message.error(response.message || '配置失败') }
  } catch (error: any) { message.error(error.message || '配置失败') }
  finally { savingUserToken.value = false }
}

const authorizeFeishu = () => {
  window.open(feishuUserApi.getAuthorizeUrl(), '_blank')
}

onMounted(() => { fetchVersion(); fetchApps().then(() => { 
  const defaultApp = appList.value.find(a => a.is_default) || appList.value[0]
  if (defaultApp) fetchUserTokenStatus(defaultApp.app_id)
}); fetchBots(); fetchCallbackStatus(); fetchLogs(); fetchChats() })
</script>

<style scoped>
.feishu-message { padding: 0; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.page-header h1 { font-size: 20px; font-weight: 500; margin: 0; }
.service-list { max-height: 400px; overflow-y: auto; }
.service-item { background: #fafafa; padding: 12px; margin-bottom: 12px; border-radius: 6px; border: 1px solid #f0f0f0; }
.service-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; }
.service-title { font-weight: 500; color: #666; }
.json-preview { background: #1e1e1e; color: #d4d4d4; padding: 16px; border-radius: 6px; max-height: 400px; overflow: auto; font-size: 13px; font-family: monospace; white-space: pre-wrap; word-break: break-all; }
</style>