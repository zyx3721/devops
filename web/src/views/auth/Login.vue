<template>
  <div class="login-container">
    <!-- 动态科技背景 -->
    <div class="tech-background">
      <div class="code-rain"></div>
      <div class="data-flow"></div>
      <div class="grid-lines"></div>
    </div>

    <!-- 左侧插画区域 -->
    <div class="login-left">
      <div class="illustration-wrapper">
        <div class="brand-title">
          <h1 class="glowing-title">DevOps 管理系统</h1>
          <p class="subtitle">统一运维管理平台</p>
        </div>
        <div class="illustration-content">
          <svg class="devops-illustration" viewBox="0 0 600 500" xmlns="http://www.w3.org/2000/svg">
            <!-- 定义渐变和滤镜 -->
            <defs>
              <linearGradient id="blueGlow" x1="0%" y1="0%" x2="100%" y2="100%">
                <stop offset="0%" style="stop-color:#3b82f6;stop-opacity:1" />
                <stop offset="100%" style="stop-color:#06b6d4;stop-opacity:1" />
              </linearGradient>
              <linearGradient id="purpleGlow" x1="0%" y1="0%" x2="100%" y2="100%">
                <stop offset="0%" style="stop-color:#8b5cf6;stop-opacity:1" />
                <stop offset="100%" style="stop-color:#a78bfa;stop-opacity:1" />
              </linearGradient>
              <filter id="glow">
                <feGaussianBlur stdDeviation="4" result="coloredBlur"/>
                <feMerge>
                  <feMergeNode in="coloredBlur"/>
                  <feMergeNode in="SourceGraphic"/>
                </feMerge>
              </filter>
              <filter id="shadow">
                <feDropShadow dx="0" dy="4" stdDeviation="6" flood-color="#3b82f6" flood-opacity="0.4"/>
              </filter>
            </defs>

            <!-- 连接线网络 -->
            <g class="connection-lines" opacity="0.6">
              <line x1="110" y1="250" x2="250" y2="120" stroke="url(#blueGlow)" stroke-width="2" stroke-dasharray="5,5">
                <animate attributeName="stroke-dashoffset" from="0" to="10" dur="1.5s" repeatCount="indefinite"/>
              </line>
              <line x1="250" y1="250" x2="330" y2="120" stroke="url(#blueGlow)" stroke-width="2" stroke-dasharray="5,5">
                <animate attributeName="stroke-dashoffset" from="0" to="10" dur="1.5s" repeatCount="indefinite"/>
              </line>
              <line x1="330" y1="120" x2="490" y2="230" stroke="url(#purpleGlow)" stroke-width="2" stroke-dasharray="5,5">
                <animate attributeName="stroke-dashoffset" from="0" to="10" dur="1.5s" repeatCount="indefinite"/>
              </line>
            </g>

            <!-- 服务器机柜 - 增强3D效果 -->
            <g class="server-rack" filter="url(#shadow)">
              <rect x="50" y="150" width="120" height="200" rx="8" fill="#1e3a8a" opacity="0.9"/>
              <rect x="55" y="155" width="110" height="190" rx="6" fill="#1e293b"/>

              <!-- 服务器层 -->
              <g class="server-layer">
                <rect x="60" y="165" width="100" height="30" rx="4" fill="url(#blueGlow)"/>
                <circle cx="70" cy="180" r="3" fill="#10b981">
                  <animate attributeName="opacity" values="1;0.3;1" dur="2s" repeatCount="indefinite"/>
                </circle>
                <circle cx="80" cy="180" r="3" fill="#10b981">
                  <animate attributeName="opacity" values="1;0.3;1" dur="2s" begin="0.3s" repeatCount="indefinite"/>
                </circle>
                <rect x="90" y="175" width="60" height="2" rx="1" fill="#60a5fa" opacity="0.6"/>
                <rect x="90" y="180" width="45" height="2" rx="1" fill="#60a5fa" opacity="0.6"/>
                <rect x="90" y="185" width="50" height="2" rx="1" fill="#60a5fa" opacity="0.6"/>
              </g>

              <g class="server-layer">
                <rect x="60" y="205" width="100" height="30" rx="4" fill="url(#blueGlow)"/>
                <circle cx="70" cy="220" r="3" fill="#10b981">
                  <animate attributeName="opacity" values="1;0.3;1" dur="2s" begin="0.5s" repeatCount="indefinite"/>
                </circle>
                <circle cx="80" cy="220" r="3" fill="#10b981">
                  <animate attributeName="opacity" values="1;0.3;1" dur="2s" begin="0.8s" repeatCount="indefinite"/>
                </circle>
                <rect x="90" y="215" width="60" height="2" rx="1" fill="#60a5fa" opacity="0.6"/>
                <rect x="90" y="220" width="50" height="2" rx="1" fill="#60a5fa" opacity="0.6"/>
                <rect x="90" y="225" width="55" height="2" rx="1" fill="#60a5fa" opacity="0.6"/>
              </g>

              <g class="server-layer">
                <rect x="60" y="245" width="100" height="30" rx="4" fill="url(#blueGlow)"/>
                <circle cx="70" cy="260" r="3" fill="#10b981">
                  <animate attributeName="opacity" values="1;0.3;1" dur="2s" begin="1s" repeatCount="indefinite"/>
                </circle>
                <circle cx="80" cy="260" r="3" fill="#10b981">
                  <animate attributeName="opacity" values="1;0.3;1" dur="2s" begin="1.3s" repeatCount="indefinite"/>
                </circle>
                <rect x="90" y="255" width="60" height="2" rx="1" fill="#60a5fa" opacity="0.6"/>
                <rect x="90" y="260" width="48" height="2" rx="1" fill="#60a5fa" opacity="0.6"/>
                <rect x="90" y="265" width="52" height="2" rx="1" fill="#60a5fa" opacity="0.6"/>
              </g>

              <g class="server-layer">
                <rect x="60" y="285" width="100" height="30" rx="4" fill="url(#blueGlow)"/>
                <circle cx="70" cy="300" r="3" fill="#10b981">
                  <animate attributeName="opacity" values="1;0.3;1" dur="2s" begin="1.5s" repeatCount="indefinite"/>
                </circle>
                <circle cx="80" cy="300" r="3" fill="#10b981">
                  <animate attributeName="opacity" values="1;0.3;1" dur="2s" begin="1.8s" repeatCount="indefinite"/>
                </circle>
                <rect x="90" y="295" width="60" height="2" rx="1" fill="#60a5fa" opacity="0.6"/>
                <rect x="90" y="300" width="55" height="2" rx="1" fill="#60a5fa" opacity="0.6"/>
                <rect x="90" y="305" width="50" height="2" rx="1" fill="#60a5fa" opacity="0.6"/>
              </g>

              <!-- 光晕效果 -->
              <circle cx="110" cy="250" r="60" fill="#3b82f6" opacity="0.1">
                <animate attributeName="r" values="60;70;60" dur="3s" repeatCount="indefinite"/>
                <animate attributeName="opacity" values="0.1;0.2;0.1" dur="3s" repeatCount="indefinite"/>
              </circle>
            </g>

            <!-- 流水线构建 - 增强科技感 -->
            <g class="pipeline" filter="url(#shadow)">
              <rect x="220" y="95" width="60" height="60" rx="10" fill="url(#blueGlow)" opacity="0.3"/>
              <rect x="225" y="100" width="50" height="50" rx="8" fill="url(#blueGlow)"/>

              <!-- 代码图标 -->
              <circle cx="235" cy="112" r="3" fill="white" opacity="0.9"/>
              <circle cx="245" cy="112" r="3" fill="white" opacity="0.9"/>
              <circle cx="255" cy="112" r="3" fill="white" opacity="0.9"/>
              <rect x="230" y="120" width="35" height="3" rx="1.5" fill="white" opacity="0.8"/>
              <rect x="230" y="127" width="25" height="3" rx="1.5" fill="white" opacity="0.8"/>
              <rect x="230" y="134" width="30" height="3" rx="1.5" fill="white" opacity="0.8"/>

              <!-- 数据流动线 -->
              <line x1="280" y1="125" x2="320" y2="125" stroke="#60a5fa" stroke-width="3" stroke-dasharray="8,4" filter="url(#glow)">
                <animate attributeName="stroke-dashoffset" from="0" to="12" dur="0.8s" repeatCount="indefinite"/>
              </line>
              <circle cx="300" cy="125" r="4" fill="#60a5fa">
                <animate attributeName="cx" values="280;320;280" dur="2s" repeatCount="indefinite"/>
              </circle>

              <rect x="320" y="95" width="60" height="60" rx="10" fill="url(#purpleGlow)" opacity="0.3"/>
              <rect x="325" y="100" width="50" height="50" rx="8" fill="url(#purpleGlow)"/>

              <!-- 部署图标 -->
              <polygon points="350,112 343,118 350,124 357,118" fill="white" opacity="0.9"/>
              <circle cx="350" cy="133" r="5" fill="white" opacity="0.8"/>
              <path d="M 345 133 L 350 138 L 355 133" stroke="white" stroke-width="2" fill="none" opacity="0.8"/>

              <!-- 光晕 -->
              <circle cx="250" cy="125" r="40" fill="#3b82f6" opacity="0.15">
                <animate attributeName="r" values="40;50;40" dur="2.5s" repeatCount="indefinite"/>
              </circle>
              <circle cx="350" cy="125" r="40" fill="#8b5cf6" opacity="0.15">
                <animate attributeName="r" values="40;50;40" dur="2.5s" begin="0.5s" repeatCount="indefinite"/>
              </circle>
            </g>

            <!-- K8s容器编排 - 3D立体效果 -->
            <g class="containers" filter="url(#shadow)">
              <rect x="210" y="210" width="70" height="70" rx="8" fill="#06b6d4" opacity="0.2"/>
              <rect x="215" y="215" width="60" height="60" rx="6" fill="url(#blueGlow)"/>
              <text x="245" y="250" text-anchor="middle" fill="white" font-size="16" font-weight="bold">K8s</text>

              <!-- 六边形装饰 -->
              <polygon points="245,225 250,228 250,234 245,237 240,234 240,228" fill="white" opacity="0.8"/>
              <polygon points="235,255 240,258 240,264 235,267 230,264 230,258" fill="white" opacity="0.6"/>
              <polygon points="255,255 260,258 260,264 255,267 250,264 250,258" fill="white" opacity="0.6"/>

              <rect x="300" y="210" width="70" height="70" rx="8" fill="#06b6d4" opacity="0.2"/>
              <rect x="305" y="215" width="60" height="60" rx="6" fill="url(#blueGlow)"/>
              <circle cx="335" cy="245" r="18" fill="#2563eb" opacity="0.9"/>
              <circle cx="335" cy="245" r="12" fill="#1e40af" opacity="0.9"/>
              <circle cx="335" cy="245" r="6" fill="#1e3a8a" opacity="0.9">
                <animate attributeName="r" values="6;8;6" dur="2s" repeatCount="indefinite"/>
              </circle>

              <!-- 光晕 -->
              <circle cx="245" cy="245" r="50" fill="#06b6d4" opacity="0.1">
                <animate attributeName="r" values="50;60;50" dur="3s" repeatCount="indefinite"/>
              </circle>
            </g>

            <!-- 实时监控大屏 - 动态图表 -->
            <g class="monitor" filter="url(#shadow)">
              <rect x="410" y="170" width="160" height="120" rx="10" fill="#1e293b" stroke="url(#blueGlow)" stroke-width="3"/>
              <rect x="415" y="175" width="150" height="110" rx="8" fill="#0f172a"/>

              <!-- 动态折线图 -->
              <path d="M 425 220 L 445 200 L 465 210 L 485 190 L 505 205 L 525 185 L 545 195 L 555 180"
                    stroke="#10b981" stroke-width="3" fill="none" filter="url(#glow)">
                <animate attributeName="d"
                         values="M 425 220 L 445 200 L 465 210 L 485 190 L 505 205 L 525 185 L 545 195 L 555 180;
                                 M 425 220 L 445 205 L 465 195 L 485 200 L 505 190 L 525 200 L 545 185 L 555 190;
                                 M 425 220 L 445 200 L 465 210 L 485 190 L 505 205 L 525 185 L 545 195 L 555 180"
                         dur="4s" repeatCount="indefinite"/>
              </path>

              <!-- 数据点 -->
              <circle cx="425" cy="220" r="3" fill="#10b981"/>
              <circle cx="445" cy="200" r="3" fill="#10b981">
                <animate attributeName="cy" values="200;205;200" dur="4s" repeatCount="indefinite"/>
              </circle>
              <circle cx="465" cy="210" r="3" fill="#10b981">
                <animate attributeName="cy" values="210;195;210" dur="4s" repeatCount="indefinite"/>
              </circle>
              <circle cx="485" cy="190" r="3" fill="#10b981">
                <animate attributeName="cy" values="190;200;190" dur="4s" repeatCount="indefinite"/>
              </circle>
              <circle cx="505" cy="205" r="3" fill="#10b981">
                <animate attributeName="cy" values="205;190;205" dur="4s" repeatCount="indefinite"/>
              </circle>
              <circle cx="525" cy="185" r="3" fill="#10b981">
                <animate attributeName="cy" values="185;200;185" dur="4s" repeatCount="indefinite"/>
              </circle>
              <circle cx="545" cy="195" r="3" fill="#10b981">
                <animate attributeName="cy" values="195;185;195" dur="4s" repeatCount="indefinite"/>
              </circle>
              <circle cx="555" cy="180" r="3" fill="#10b981">
                <animate attributeName="cy" values="180;190;180" dur="4s" repeatCount="indefinite"/>
              </circle>

              <!-- 网格线 -->
              <line x1="420" y1="200" x2="560" y2="200" stroke="#334155" stroke-width="1" opacity="0.3"/>
              <line x1="420" y1="220" x2="560" y2="220" stroke="#334155" stroke-width="1" opacity="0.3"/>
              <line x1="420" y1="240" x2="560" y2="240" stroke="#334155" stroke-width="1" opacity="0.3"/>

              <text x="490" y="270" text-anchor="middle" fill="#64748b" font-size="13" font-weight="500">实时监控</text>

              <!-- 光晕 -->
              <circle cx="490" cy="230" r="70" fill="#10b981" opacity="0.08">
                <animate attributeName="r" values="70;80;70" dur="3s" repeatCount="indefinite"/>
              </circle>
            </g>

            <!-- 自动化部署箭头 - 增强动画 -->
            <g class="deploy-arrow">
              <path d="M 170 250 Q 300 200 410 220" stroke="url(#purpleGlow)" stroke-width="4" fill="none" opacity="0.8" filter="url(#glow)">
                <animate attributeName="stroke-dasharray" from="0,1000" to="1000,0" dur="3s" repeatCount="indefinite"/>
              </path>
              <polygon points="410,220 400,215 400,225" fill="#a78bfa" opacity="0.8">
                <animate attributeName="opacity" values="0.8;1;0.8" dur="1.5s" repeatCount="indefinite"/>
              </polygon>

              <!-- 流动粒子 -->
              <circle r="4" fill="#a78bfa" filter="url(#glow)">
                <animateMotion dur="3s" repeatCount="indefinite" path="M 170 250 Q 300 200 410 220"/>
              </circle>
            </g>
          </svg>
        </div>
        <div class="feature-tags">
          <span class="tag">流水线构建</span>
          <span class="tag">自动化部署</span>
          <span class="tag">容器编排</span>
          <span class="tag">实时监控</span>
        </div>
      </div>
    </div>

    <!-- 右侧登录表单区域 -->
    <div class="login-right">
      <div class="login-card">
        <div class="card-decoration"></div>
        <div class="login-header">
          <h2>欢迎登录</h2>
          <p>DevOps 统一运维管理平台</p>
        </div>
        <a-form :model="form" :rules="rules" @finish="handleSubmit" layout="vertical" class="login-form">
          <a-form-item label="用户名" name="username">
            <a-input
              v-model:value="form.username"
              placeholder="请输入用户名"
              size="large"
              class="custom-input"
            >
              <template #prefix><UserOutlined class="input-icon" /></template>
            </a-input>
          </a-form-item>
          <a-form-item label="密码" name="password">
            <a-input-password
              v-model:value="form.password"
              placeholder="请输入密码"
              size="large"
              class="custom-input"
            >
              <template #prefix><LockOutlined class="input-icon" /></template>
            </a-input-password>
          </a-form-item>
          <a-form-item>
            <a-button
              type="primary"
              html-type="submit"
              size="large"
              block
              :loading="loading"
              class="login-button"
            >
              <span class="button-text">登录系统</span>
            </a-button>
          </a-form-item>
        </a-form>
        <div class="login-tips">
          <a-typography-text type="secondary">
            首次使用？请联系管理员创建账号
          </a-typography-text>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { UserOutlined, LockOutlined } from '@ant-design/icons-vue'
import { authApi } from '@/services/auth'

const router = useRouter()
const loading = ref(false)

const form = reactive({
  username: '',
  password: ''
})

const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }]
}

const handleSubmit = async () => {
  loading.value = true
  try {
    const response = await authApi.login(form.username, form.password)
    if (response.code === 0 && response.data?.token) {
      localStorage.setItem('token', response.data.token)
      localStorage.setItem('userInfo', JSON.stringify(response.data.user))
      message.success('登录成功')
      router.push('/')
    } else {
      message.error(response.message || '登录失败')
    }
  } catch (error: any) {
    message.error(error.response?.data?.message || '登录失败')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
/* 基础容器 */
.login-container {
  display: flex;
  height: 100vh;
  max-height: 100vh;
  background: linear-gradient(135deg, #0a0e1a 0%, #0f172a 30%, #1e293b 70%, #0f172a 100%);
  position: relative;
  overflow: hidden;
}

/* 动态科技背景 */
.tech-background {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  pointer-events: none;
  z-index: 0;
}

.code-rain {
  position: absolute;
  width: 100%;
  height: 100%;
  background-image:
    linear-gradient(90deg, rgba(59, 130, 246, 0.03) 1px, transparent 1px),
    linear-gradient(rgba(59, 130, 246, 0.03) 1px, transparent 1px);
  background-size: 50px 50px;
  animation: codeRainMove 20s linear infinite;
}

@keyframes codeRainMove {
  0% { transform: translateY(0); }
  100% { transform: translateY(50px); }
}

.data-flow {
  position: absolute;
  width: 100%;
  height: 100%;
  background:
    radial-gradient(circle at 20% 30%, rgba(59, 130, 246, 0.15) 0%, transparent 40%),
    radial-gradient(circle at 80% 70%, rgba(139, 92, 246, 0.15) 0%, transparent 40%),
    radial-gradient(circle at 50% 50%, rgba(6, 182, 212, 0.1) 0%, transparent 50%);
  animation: dataFlowPulse 8s ease-in-out infinite;
}

@keyframes dataFlowPulse {
  0%, 100% { opacity: 0.5; }
  50% { opacity: 1; }
}

.grid-lines {
  position: absolute;
  width: 100%;
  height: 100%;
  background-image:
    linear-gradient(90deg, rgba(59, 130, 246, 0.05) 1px, transparent 1px),
    linear-gradient(rgba(59, 130, 246, 0.05) 1px, transparent 1px);
  background-size: 100px 100px;
  opacity: 0.3;
}

/* 左侧插画区域 */
.login-left {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px 50px;
  position: relative;
  z-index: 1;
  overflow: hidden;
}

.illustration-wrapper {
  max-width: 650px;
  width: 100%;
  display: flex;
  flex-direction: column;
  justify-content: center;
  height: 100%;
}

/* 品牌标题 - 增强发光效果 */
.brand-title {
  text-align: center;
  margin-bottom: 40px;
  flex-shrink: 0;
}

.glowing-title {
  font-size: 42px;
  font-weight: 800;
  color: #ffffff;
  margin-bottom: 12px;
  letter-spacing: 3px;
  text-shadow:
    0 0 10px rgba(59, 130, 246, 0.8),
    0 0 20px rgba(59, 130, 246, 0.6),
    0 0 30px rgba(59, 130, 246, 0.4),
    0 2px 15px rgba(0, 0, 0, 0.5);
  animation: titleGlow 3s ease-in-out infinite;
  background: linear-gradient(135deg, #ffffff 0%, #60a5fa 50%, #a78bfa 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

@keyframes titleGlow {
  0%, 100% {
    text-shadow:
      0 0 10px rgba(59, 130, 246, 0.8),
      0 0 20px rgba(59, 130, 246, 0.6),
      0 0 30px rgba(59, 130, 246, 0.4);
  }
  50% {
    text-shadow:
      0 0 15px rgba(59, 130, 246, 1),
      0 0 30px rgba(59, 130, 246, 0.8),
      0 0 45px rgba(59, 130, 246, 0.6);
  }
}

.subtitle {
  font-size: 17px;
  color: #94a3b8;
  margin: 0;
  letter-spacing: 2px;
  font-weight: 500;
  text-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
}

.illustration-content {
  margin-bottom: 35px;
  flex-shrink: 0;
}

.devops-illustration {
  width: 100%;
  height: auto;
  max-height: 450px;
  filter: drop-shadow(0 15px 40px rgba(59, 130, 246, 0.3));
}

/* 磨砂玻璃质感标签 */
.feature-tags {
  display: flex;
  justify-content: center;
  gap: 16px;
  flex-wrap: wrap;
  flex-shrink: 0;
}

.tag {
  padding: 10px 24px;
  background: rgba(59, 130, 246, 0.08);
  border: 1.5px solid rgba(59, 130, 246, 0.3);
  border-radius: 24px;
  color: #60a5fa;
  font-size: 14px;
  font-weight: 600;
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  box-shadow:
    0 4px 12px rgba(0, 0, 0, 0.2),
    inset 0 1px 0 rgba(255, 255, 255, 0.1);
  transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
  cursor: pointer;
  position: relative;
  overflow: hidden;
}

.tag::before {
  content: '';
  position: absolute;
  top: 0;
  left: -100%;
  width: 100%;
  height: 100%;
  background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.2), transparent);
  transition: left 0.5s ease;
}

.tag:hover {
  background: rgba(59, 130, 246, 0.15);
  border-color: rgba(59, 130, 246, 0.6);
  transform: translateY(-3px) scale(1.05);
  box-shadow:
    0 8px 20px rgba(59, 130, 246, 0.3),
    inset 0 1px 0 rgba(255, 255, 255, 0.2);
  color: #93c5fd;
}

.tag:hover::before {
  left: 100%;
}

.tag:active {
  transform: translateY(-1px) scale(1.02);
}

/* 右侧登录表单区域 */
.login-right {
  width: 40%;
  min-width: 500px;
  max-width: 650px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px 50px;
  background: rgba(30, 41, 59, 0.35);
  backdrop-filter: blur(25px);
  -webkit-backdrop-filter: blur(25px);
  border-left: 1px solid rgba(59, 130, 246, 0.2);
  position: relative;
  z-index: 1;
  overflow-y: auto;
  overflow-x: hidden;
}

.login-right::-webkit-scrollbar {
  width: 6px;
}

.login-right::-webkit-scrollbar-track {
  background: rgba(15, 23, 42, 0.3);
}

.login-right::-webkit-scrollbar-thumb {
  background: rgba(59, 130, 246, 0.3);
  border-radius: 3px;
}

.login-right::-webkit-scrollbar-thumb:hover {
  background: rgba(59, 130, 246, 0.5);
}

.login-right::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: linear-gradient(180deg, rgba(59, 130, 246, 0.05) 0%, transparent 100%);
  pointer-events: none;
}

/* 登录卡片 - 精致悬浮效果 */
.login-card {
  width: 100%;
  max-width: 480px;
  padding: 48px 44px;
  background: rgba(255, 255, 255, 0.98);
  border-radius: 24px;
  box-shadow:
    0 30px 80px rgba(0, 0, 0, 0.4),
    0 0 0 1px rgba(59, 130, 246, 0.2),
    inset 0 1px 0 rgba(255, 255, 255, 1),
    inset 0 -1px 0 rgba(0, 0, 0, 0.05);
  backdrop-filter: blur(15px);
  -webkit-backdrop-filter: blur(15px);
  position: relative;
  transition: transform 0.3s ease, box-shadow 0.3s ease;
}

.login-card:hover {
  transform: translateY(-2px);
  box-shadow:
    0 35px 90px rgba(0, 0, 0, 0.45),
    0 0 0 1px rgba(59, 130, 246, 0.3),
    inset 0 1px 0 rgba(255, 255, 255, 1);
}

/* 顶部渐变装饰条 */
.card-decoration {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 5px;
  background: linear-gradient(90deg, #3b82f6 0%, #8b5cf6 50%, #06b6d4 100%);
  border-radius: 24px 24px 0 0;
  box-shadow: 0 2px 10px rgba(59, 130, 246, 0.5);
}

.card-decoration::after {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  width: 30%;
  height: 100%;
  background: linear-gradient(90deg, rgba(255, 255, 255, 0.5), transparent);
  animation: decorationShine 3s ease-in-out infinite;
}

@keyframes decorationShine {
  0%, 100% { transform: translateX(0); opacity: 0; }
  50% { transform: translateX(300%); opacity: 1; }
}

/* 登录表头 */
.login-header {
  text-align: center;
  margin-bottom: 36px;
}

.login-header h2 {
  font-size: 32px;
  font-weight: 800;
  color: #0f172a;
  margin-bottom: 12px;
  letter-spacing: 0.5px;
  background: linear-gradient(135deg, #0f172a 0%, #1e40af 50%, #3b82f6 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  text-shadow: 0 2px 10px rgba(59, 130, 246, 0.1);
}

.login-header p {
  font-size: 14px;
  color: #64748b;
  margin: 0;
  font-weight: 500;
  letter-spacing: 0.5px;
}

/* 表单样式 */
.login-form {
  margin-bottom: 24px;
}

.login-form :deep(.ant-form-item-label > label) {
  font-size: 14px;
  font-weight: 700;
  color: #1e293b;
  letter-spacing: 0.3px;
}

.login-form :deep(.ant-form-item) {
  margin-bottom: 20px;
}

/* 输入框 - 现代简约风格 */
.custom-input {
  border-radius: 12px;
  border: 2px solid #e2e8f0;
  background: #f8fafc;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.custom-input:hover {
  border-color: #3b82f6;
  background: #ffffff;
  box-shadow: 0 2px 6px rgba(59, 130, 246, 0.1);
}

.custom-input:focus,
.custom-input:focus-within {
  border-color: #3b82f6;
  background: #ffffff;
  box-shadow:
    0 0 0 4px rgba(59, 130, 246, 0.15),
    0 4px 12px rgba(59, 130, 246, 0.2);
  transform: translateY(-1px);
}

.input-icon {
  color: #64748b;
  font-size: 16px;
  transition: color 0.3s ease;
}

.custom-input:focus .input-icon,
.custom-input:focus-within .input-icon {
  color: #3b82f6;
}

/* 密码显示/隐藏按钮优化 */
.custom-input :deep(.ant-input-password-icon) {
  color: #94a3b8;
  font-size: 16px;
  transition: all 0.3s ease;
}

.custom-input :deep(.ant-input-password-icon:hover) {
  color: #3b82f6;
  transform: scale(1.1);
}

/* 登录按钮 - 渐变蓝+悬浮动效 */
.login-button {
  height: 48px;
  border-radius: 12px;
  font-size: 16px;
  font-weight: 700;
  background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
  border: none;
  box-shadow:
    0 8px 20px rgba(59, 130, 246, 0.4),
    inset 0 1px 0 rgba(255, 255, 255, 0.2);
  transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
  position: relative;
  overflow: hidden;
}

.login-button::before {
  content: '';
  position: absolute;
  top: 0;
  left: -100%;
  width: 100%;
  height: 100%;
  background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.3), transparent);
  transition: left 0.6s ease;
}

.login-button:hover::before {
  left: 100%;
}

.login-button:hover {
  background: linear-gradient(135deg, #2563eb 0%, #1d4ed8 100%);
  box-shadow:
    0 12px 28px rgba(59, 130, 246, 0.5),
    inset 0 1px 0 rgba(255, 255, 255, 0.3);
  transform: translateY(-3px);
}

.login-button:active {
  transform: translateY(-1px);
  box-shadow:
    0 6px 16px rgba(59, 130, 246, 0.4),
    inset 0 1px 0 rgba(255, 255, 255, 0.2);
}

.button-text {
  position: relative;
  z-index: 1;
  letter-spacing: 1px;
}

/* 登录提示 */
.login-tips {
  text-align: center;
  padding-top: 20px;
  border-top: 1px solid #e2e8f0;
}

.login-tips :deep(.ant-typography) {
  font-size: 13px;
  color: #64748b;
  font-weight: 500;
}

/* 响应式适配 */
@media (max-width: 1200px) {
  .login-left {
    padding: 40px;
  }

  .glowing-title {
    font-size: 40px;
  }

  .subtitle {
    font-size: 18px;
  }

  .login-right {
    width: 45%;
    min-width: 450px;
    padding: 40px;
  }

  .login-card {
    max-width: 450px;
    padding: 48px 40px;
  }
}

@media (max-width: 992px) {
  .login-container {
    flex-direction: column;
  }

  .login-left {
    padding: 40px 20px;
  }

  .brand-title {
    margin-bottom: 50px;
  }

  .glowing-title {
    font-size: 36px;
  }

  .subtitle {
    font-size: 16px;
  }

  .illustration-content {
    margin-bottom: 40px;
  }

  .login-right {
    width: 100%;
    border-left: none;
    border-top: 1px solid rgba(59, 130, 246, 0.2);
  }
}

@media (max-width: 576px) {
  .login-left {
    padding: 30px 20px;
  }

  .glowing-title {
    font-size: 32px;
    letter-spacing: 2px;
  }

  .subtitle {
    font-size: 15px;
  }

  .login-right {
    padding: 30px 20px;
  }

  .login-card {
    padding: 40px 28px;
  }

  .login-header h2 {
    font-size: 30px;
  }

  .feature-tags {
    gap: 12px;
  }

  .tag {
    padding: 10px 20px;
    font-size: 14px;
  }
}

/* 暗色模式友好 */
@media (prefers-color-scheme: dark) {
  .login-card {
    background: rgba(30, 41, 59, 0.95);
  }

  .login-header h2 {
    color: #f1f5f9;
    background: linear-gradient(135deg, #f1f5f9 0%, #60a5fa 50%, #a78bfa 100%);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;
  }

  .login-header p {
    color: #94a3b8;
  }

  .login-form :deep(.ant-form-item-label > label) {
    color: #e2e8f0;
  }

  .custom-input {
    background: rgba(15, 23, 42, 0.6);
    border-color: rgba(59, 130, 246, 0.3);
    color: #f1f5f9;
  }

  .custom-input:hover {
    background: rgba(15, 23, 42, 0.8);
    border-color: rgba(59, 130, 246, 0.5);
  }

  .login-tips {
    border-top-color: rgba(59, 130, 246, 0.2);
  }

  .login-tips :deep(.ant-typography) {
    color: #94a3b8;
  }
}
</style>
