/**
 * 验证性能优化配置
 * 检查所有性能优化相关的配置是否正确
 */

import fs from 'fs'
import path from 'path'
import { fileURLToPath } from 'url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const projectRoot = path.resolve(__dirname, '..')

console.log('🔍 验证性能优化配置...\n')

let allPassed = true

// 1. 检查 vite.config.ts 是否包含代码分割配置
console.log('1. 检查 Vite 配置...')
const viteConfigPath = path.join(projectRoot, 'vite.config.ts')
const viteConfig = fs.readFileSync(viteConfigPath, 'utf-8')

const checks = [
  { name: 'manualChunks 配置', pattern: /manualChunks\s*:/ },
  { name: 'vue-vendor chunk', pattern: /'vue-vendor'\s*:/ },
  { name: 'antd-vendor chunk', pattern: /'antd-vendor'\s*:/ },
  { name: 'app-pages chunk', pattern: /'app-pages'\s*:/ },
  { name: 'pipeline-pages chunk', pattern: /'pipeline-pages'\s*:/ },
  { name: 'k8s-pages chunk', pattern: /'k8s-pages'\s*:/ },
  { name: 'CSS 代码分割', pattern: /cssCodeSplit\s*:\s*true/ },
  { name: 'chunk 大小警告', pattern: /chunkSizeWarningLimit/ }
]

checks.forEach(check => {
  if (check.pattern.test(viteConfig)) {
    console.log(`   ✅ ${check.name}`)
  } else {
    console.log(`   ❌ ${check.name}`)
    allPassed = false
  }
})

// 2. 检查路由配置是否使用动态 import
console.log('\n2. 检查路由懒加载...')
const routerPath = path.join(projectRoot, 'src/router/index.ts')
const routerConfig = fs.readFileSync(routerPath, 'utf-8')

const routeChecks = [
  { name: '弹性工程路由', pattern: /path:\s*['"]\/resilience['"][\s\S]*?import\(['"]@\/views\/resilience/ },
  { name: '流水线设计器路由', pattern: /path:\s*['"]\/pipeline\/designer['"][\s\S]*?import\(['"]@\/views\/pipeline/ },
  { name: '构建缓存路由', pattern: /path:\s*['"]\/pipeline\/cache['"][\s\S]*?import\(['"]@\/views\/pipeline/ },
  { name: '系统监控路由', pattern: /path:\s*['"]\/system\/monitor['"][\s\S]*?import\(['"]@\/views\/system/ }
]

routeChecks.forEach(check => {
  if (check.pattern.test(routerConfig)) {
    console.log(`   ✅ ${check.name}`)
  } else {
    console.log(`   ⚠️  ${check.name} (可能未找到或格式不同)`)
  }
})

// 3. 检查 MainLayout 优化
console.log('\n3. 检查 MainLayout 优化...')
const layoutPath = path.join(projectRoot, 'src/layouts/MainLayout.vue')
const layoutContent = fs.readFileSync(layoutPath, 'utf-8')

const layoutChecks = [
  { name: 'v-show 优化 (logo)', pattern: /v-show\s*=\s*["']!collapsed["']/ },
  { name: '性能监控组件', pattern: /PerformanceMonitor/ },
  { name: '菜单 key 设置', pattern: /key\s*=\s*["']\/dashboard["']/ }
]

layoutChecks.forEach(check => {
  if (check.pattern.test(layoutContent)) {
    console.log(`   ✅ ${check.name}`)
  } else {
    console.log(`   ❌ ${check.name}`)
    allPassed = false
  }
})

// 4. 检查性能工具文件是否存在
console.log('\n4. 检查性能工具文件...')
const files = [
  { name: '性能测量工具', path: 'src/utils/performance.ts' },
  { name: '性能监控组件', path: 'src/components/PerformanceMonitor.vue' },
  { name: '性能测试指南', path: '../../.kiro/specs/frontend-menu-integration/TASK_9_PERFORMANCE_TESTING_GUIDE.md' },
  { name: '完成总结', path: '../../.kiro/specs/frontend-menu-integration/TASK_9_COMPLETION_SUMMARY.md' }
]

files.forEach(file => {
  const filePath = path.join(projectRoot, file.path)
  if (fs.existsSync(filePath)) {
    console.log(`   ✅ ${file.name}`)
  } else {
    console.log(`   ❌ ${file.name}`)
    allPassed = false
  }
})

// 5. 检查构建产物
console.log('\n5. 检查构建产物...')
const distPath = path.join(projectRoot, 'dist')
if (fs.existsSync(distPath)) {
  const jsPath = path.join(distPath, 'js')
  if (fs.existsSync(jsPath)) {
    const jsFiles = fs.readdirSync(jsPath)
    const chunks = {
      'vue-vendor': jsFiles.some(f => f.includes('vue-vendor')),
      'antd-vendor': jsFiles.some(f => f.includes('antd-vendor')),
      'app-pages': jsFiles.some(f => f.includes('app-pages')),
      'pipeline-pages': jsFiles.some(f => f.includes('pipeline-pages')),
      'k8s-pages': jsFiles.some(f => f.includes('k8s-pages'))
    }
    
    Object.entries(chunks).forEach(([name, exists]) => {
      if (exists) {
        console.log(`   ✅ ${name} chunk`)
      } else {
        console.log(`   ⚠️  ${name} chunk (未找到，可能需要重新构建)`)
      }
    })
    
    console.log(`\n   总共生成了 ${jsFiles.length} 个 JS 文件`)
  } else {
    console.log('   ⚠️  dist/js 目录不存在，请运行 npm run build')
  }
} else {
  console.log('   ⚠️  dist 目录不存在，请运行 npm run build')
}

// 总结
console.log('\n' + '='.repeat(50))
if (allPassed) {
  console.log('✅ 所有关键配置检查通过！')
  console.log('\n下一步:')
  console.log('1. 运行 npm run dev 启动开发服务器')
  console.log('2. 打开浏览器查看右下角的性能监控面板')
  console.log('3. 按照 TASK_9_PERFORMANCE_TESTING_GUIDE.md 进行性能测试')
} else {
  console.log('⚠️  部分配置检查未通过，请检查上述标记为 ❌ 的项目')
}
console.log('='.repeat(50))
