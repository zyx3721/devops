<template>
  <div class="profile">
    <div class="page-header">
      <h1>个人中心</h1>
    </div>

    <a-row :gutter="24">
      <a-col :xs="24" :md="12">
        <a-card title="个人信息">
          <a-descriptions :column="1">
            <a-descriptions-item label="用户名">{{ userInfo?.username || '-' }}</a-descriptions-item>
            <a-descriptions-item label="邮箱">{{ userInfo?.email || '-' }}</a-descriptions-item>
            <a-descriptions-item label="手机号">{{ userInfo?.phone || '-' }}</a-descriptions-item>
            <a-descriptions-item label="角色">
              <a-tag :color="getRoleColor(userInfo?.role)">{{ getRoleText(userInfo?.role) }}</a-tag>
            </a-descriptions-item>
            <a-descriptions-item label="状态">
              <a-tag :color="userInfo?.status === 'active' ? 'green' : 'red'">
                {{ userInfo?.status === 'active' ? '启用' : '禁用' }}
              </a-tag>
            </a-descriptions-item>
            <a-descriptions-item label="最后登录">{{ userInfo?.last_login_at || '-' }}</a-descriptions-item>
          </a-descriptions>
        </a-card>
      </a-col>

      <a-col :xs="24" :md="12">
        <a-card title="修改密码">
          <a-form :model="passwordForm" :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }" @finish="handleChangePassword">
            <a-form-item label="原密码" name="old_password" :rules="[{ required: true, message: '请输入原密码' }]">
              <a-input-password v-model:value="passwordForm.old_password" placeholder="请输入原密码" />
            </a-form-item>
            <a-form-item label="新密码" name="new_password" :rules="[{ required: true, message: '请输入新密码' }, { min: 6, message: '密码至少6位' }]">
              <a-input-password v-model:value="passwordForm.new_password" placeholder="请输入新密码" />
            </a-form-item>
            <a-form-item label="确认密码" name="confirm_password" :rules="[{ required: true, message: '请确认新密码' }]">
              <a-input-password v-model:value="passwordForm.confirm_password" placeholder="请确认新密码" />
            </a-form-item>
            <a-form-item :wrapper-col="{ offset: 6, span: 16 }">
              <a-button type="primary" html-type="submit" :loading="changing">修改密码</a-button>
            </a-form-item>
          </a-form>
        </a-card>
      </a-col>
    </a-row>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { authApi } from '@/services/auth'
import type { User } from '@/types'

const userInfo = ref<User | null>(null)
const changing = ref(false)

const passwordForm = reactive({
  old_password: '',
  new_password: '',
  confirm_password: ''
})

const getRoleColor = (role?: string) => {
  const colors: Record<string, string> = {
    admin: 'red',
    user: 'blue',
    viewer: 'default'
  }
  return colors[role || ''] || 'default'
}

const getRoleText = (role?: string) => {
  const texts: Record<string, string> = {
    admin: '管理员',
    user: '普通用户',
    viewer: '只读用户'
  }
  return texts[role || ''] || role || '-'
}

const fetchProfile = async () => {
  try {
    const response = await authApi.getProfile()
    if (response.code === 0 && response.data) {
      userInfo.value = response.data
    }
  } catch (error: any) {
    message.error(error.message || '获取个人信息失败')
  }
}

const handleChangePassword = async () => {
  if (passwordForm.new_password !== passwordForm.confirm_password) {
    message.error('两次输入的密码不一致')
    return
  }

  changing.value = true
  try {
    const response = await authApi.changePassword({
      old_password: passwordForm.old_password,
      new_password: passwordForm.new_password
    })
    if (response.code === 0) {
      message.success('密码修改成功')
      Object.assign(passwordForm, {
        old_password: '',
        new_password: '',
        confirm_password: ''
      })
    } else {
      message.error(response.message || '修改失败')
    }
  } catch (error: any) {
    message.error(error.message || '修改失败')
  } finally {
    changing.value = false
  }
}

onMounted(() => {
  fetchProfile()
})
</script>

<style scoped>
.profile {
  padding: 0;
}

.page-header {
  margin-bottom: 24px;
}

.page-header h1 {
  font-size: 20px;
  font-weight: 500;
  margin: 0;
}
</style>
