import { createApp } from 'vue'
import { createPinia } from 'pinia'
import Antd from 'ant-design-vue'
import ElementPlus from 'element-plus'
import App from './App.vue'
import router from './router'
import i18n from './locales'
import { setupErrorHandler } from './utils/errorHandler'
import 'ant-design-vue/dist/reset.css'
import 'element-plus/dist/index.css'

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)
app.use(router)
app.use(i18n)
app.use(Antd)
app.use(ElementPlus)

// Setup global error handler
setupErrorHandler(app)

app.mount('#app')
