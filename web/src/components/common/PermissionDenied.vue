<template>
  <div class="permission-denied">
    <div class="permission-denied__icon">
      <LockOutlined :style="{ fontSize: '64px', color: '#faad14' }" />
    </div>
    <div class="permission-denied__content">
      <h3 class="permission-denied__title">{{ title }}</h3>
      <p class="permission-denied__description">{{ description }}</p>
      <div v-if="requiredPermissions.length > 0" class="permission-denied__permissions">
        <p>需要以下权限：</p>
        <ul>
          <li v-for="permission in requiredPermissions" :key="permission">
            {{ permission }}
          </li>
        </ul>
      </div>
      <slot name="extra"></slot>
    </div>
    <div class="permission-denied__action">
      <a-space>
        <a-button v-if="showContact" type="primary" @click="handleContact">
          <template #icon><MailOutlined /></template>
          联系管理员
        </a-button>
        <a-button v-if="showBack" @click="handleBack">
          <template #icon><RollbackOutlined /></template>
          返回
        </a-button>
        <a-button v-if="showHome" @click="handleHome">
          <template #icon><HomeOutlined /></template>
          返回首页
        </a-button>
        <slot name="action"></slot>
      </a-space>
    </div>
  </div>
</template>

<script setup lang="ts">
import {
  LockOutlined,
  MailOutlined,
  RollbackOutlined,
  HomeOutlined,
} from '@ant-design/icons-vue'
import { useRouter } from 'vue-router'

interface Props {
  title?: string
  description?: string
  requiredPermissions?: string[]
  showContact?: boolean
  showBack?: boolean
  showHome?: boolean
  contactEmail?: string
}

const props = withDefaults(defineProps<Props>(), {
  title: '权限不足',
  description: '抱歉，您没有权限访问此页面或执行此操作',
  requiredPermissions: () => [],
  showContact: true,
  showBack: true,
  showHome: false,
  contactEmail: 'admin@example.com',
})

const emit = defineEmits<{
  contact: []
  back: []
  home: []
}>()

const router = useRouter()

const handleContact = () => {
  if (emit) {
    emit('contact')
  } else if (props.contactEmail) {
    window.location.href = `mailto:${props.contactEmail}`
  }
}

const handleBack = () => {
  if (emit) {
    emit('back')
  } else {
    router.back()
  }
}

const handleHome = () => {
  if (emit) {
    emit('home')
  } else {
    router.push('/')
  }
}
</script>

<style scoped lang="scss">
.permission-denied {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 48px 24px;
  text-align: center;
  min-height: 400px;

  &__icon {
    margin-bottom: 24px;
  }

  &__content {
    margin-bottom: 24px;
    max-width: 600px;
  }

  &__title {
    font-size: 20px;
    font-weight: 500;
    color: rgba(0, 0, 0, 0.85);
    margin: 0 0 12px;
  }

  &__description {
    font-size: 14px;
    color: rgba(0, 0, 0, 0.65);
    margin: 0 0 16px;
    line-height: 1.5;
  }

  &__permissions {
    background: #fffbe6;
    border: 1px solid #ffe58f;
    border-radius: 4px;
    padding: 16px;
    text-align: left;
    margin-top: 16px;

    p {
      margin: 0 0 8px;
      font-weight: 500;
      color: rgba(0, 0, 0, 0.85);
    }

    ul {
      margin: 0;
      padding-left: 20px;
      list-style: disc;

      li {
        color: rgba(0, 0, 0, 0.65);
        line-height: 1.8;
      }
    }
  }

  &__action {
    margin-top: 8px;
  }
}
</style>
