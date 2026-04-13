import type { ResourceFormData } from './types'

// 标签转 YAML
const labelsToYAML = (items: { key: string; value: string }[], indent: string, defaultName: string): string => {
  const validItems = items.filter(i => i.key && i.value)
  if (validItems.length === 0) return `${indent}app: ${defaultName}`
  return validItems.map(i => `${indent}${i.key}: ${i.value}`).join('\n')
}

// 构建 Deployment YAML
export const buildDeploymentYAML = (f: ResourceFormData, ns: string): string => {
  const containers = f.containers.map((c, idx) => {
    const containerName = c.name || `container-${idx + 1}`
    let yaml = `      - name: ${containerName}
        image: ${c.image}
        imagePullPolicy: ${c.imagePullPolicy}`
    
    // 端口
    const validPorts = c.ports.filter(p => p.containerPort)
    if (validPorts.length > 0) {
      yaml += `\n        ports:`
      validPorts.forEach(p => {
        yaml += `\n        - containerPort: ${p.containerPort}`
        if (p.name) yaml += `\n          name: ${p.name}`
        yaml += `\n          protocol: ${p.protocol}`
      })
    }
    
    // 环境变量
    const validEnvs = c.envs.filter(e => e.name)
    if (validEnvs.length > 0) {
      yaml += `\n        env:`
      validEnvs.forEach(e => {
        yaml += `\n        - name: ${e.name}\n          value: "${e.value}"`
      })
    }
    
    // 资源限制
    const hasResources = c.resources.cpuRequest || c.resources.cpuLimit || c.resources.memoryRequest || c.resources.memoryLimit
    if (hasResources) {
      yaml += `\n        resources:`
      if (c.resources.cpuRequest || c.resources.memoryRequest) {
        yaml += `\n          requests:`
        if (c.resources.cpuRequest) yaml += `\n            cpu: "${c.resources.cpuRequest}"`
        if (c.resources.memoryRequest) yaml += `\n            memory: "${c.resources.memoryRequest}"`
      }
      if (c.resources.cpuLimit || c.resources.memoryLimit) {
        yaml += `\n          limits:`
        if (c.resources.cpuLimit) yaml += `\n            cpu: "${c.resources.cpuLimit}"`
        if (c.resources.memoryLimit) yaml += `\n            memory: "${c.resources.memoryLimit}"`
      }
    }
    
    // 命令和参数
    if (c.command) {
      const cmds = c.command.split(',').map(s => s.trim())
      yaml += `\n        command: [${cmds.map(s => `"${s}"`).join(', ')}]`
    }
    if (c.args) {
      const args = c.args.split(',').map(s => s.trim())
      yaml += `\n        args: [${args.map(s => `"${s}"`).join(', ')}]`
    }
    
    // 卷挂载
    const validMounts = c.volumeMounts.filter(m => m.name && m.mountPath)
    if (validMounts.length > 0) {
      yaml += `\n        volumeMounts:`
      validMounts.forEach(m => {
        yaml += `\n        - name: ${m.name}\n          mountPath: ${m.mountPath}`
        if (m.readOnly) yaml += `\n          readOnly: true`
      })
    }
    
    return yaml
  }).join('\n')
  
  let yaml = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: ${f.name}
  namespace: ${ns}
spec:
  replicas: ${f.replicas}
  selector:
    matchLabels:
${labelsToYAML(f.labelItems, '      ', f.name)}
  template:
    metadata:
      labels:
${labelsToYAML(f.labelItems, '        ', f.name)}
    spec:
      containers:
${containers}`
  
  // 卷
  const validVolumes = f.volumes.filter(v => v.name)
  if (validVolumes.length > 0) {
    yaml += `\n      volumes:`
    validVolumes.forEach(v => {
      yaml += `\n      - name: ${v.name}`
      switch (v.type) {
        case 'emptyDir': yaml += `\n        emptyDir: {}`; break
        case 'configMap': yaml += `\n        configMap:\n          name: ${v.source}`; break
        case 'secret': yaml += `\n        secret:\n          secretName: ${v.source}`; break
        case 'pvc': yaml += `\n        persistentVolumeClaim:\n          claimName: ${v.source}`; break
        case 'hostPath': yaml += `\n        hostPath:\n          path: ${v.source}`; break
      }
    })
  }
  
  if (f.imagePullSecrets) {
    yaml += `\n      imagePullSecrets:`
    f.imagePullSecrets.split(',').forEach(s => { yaml += `\n      - name: ${s.trim()}` })
  }
  
  if (f.serviceAccount) yaml += `\n      serviceAccountName: ${f.serviceAccount}`
  
  const validNodeSelectors = f.nodeSelectorItems.filter(i => i.key && i.value)
  if (validNodeSelectors.length > 0) {
    yaml += `\n      nodeSelector:`
    validNodeSelectors.forEach(i => { yaml += `\n        ${i.key}: ${i.value}` })
  }
  
  return yaml
}

// 构建 Service YAML
export const buildServiceYAML = (f: ResourceFormData, ns: string): string => {
  if (f.serviceType === 'ExternalName') {
    return `apiVersion: v1
kind: Service
metadata:
  name: ${f.name}
  namespace: ${ns}
spec:
  type: ExternalName
  externalName: ${f.externalName}`
  }
  
  let yaml = `apiVersion: v1
kind: Service
metadata:
  name: ${f.name}
  namespace: ${ns}
spec:
  type: ${f.serviceType}
  selector:
${labelsToYAML(f.selectorItems, '    ', f.name)}
  ports:`
  
  f.servicePorts.forEach(p => {
    yaml += `\n  - port: ${p.port}`
    if (p.name) yaml += `\n    name: ${p.name}`
    yaml += `\n    targetPort: ${p.targetPort || p.port}`
    yaml += `\n    protocol: ${p.protocol}`
    if (f.serviceType === 'NodePort' && p.nodePort) yaml += `\n    nodePort: ${p.nodePort}`
  })
  
  if (f.sessionAffinity !== 'None') yaml += `\n  sessionAffinity: ${f.sessionAffinity}`
  if (f.serviceType === 'LoadBalancer' && f.loadBalancerIP) yaml += `\n  loadBalancerIP: ${f.loadBalancerIP}`
  
  return yaml
}

// 构建 ConfigMap YAML
export const buildConfigMapYAML = (f: ResourceFormData, ns: string): string => {
  let yaml = `apiVersion: v1
kind: ConfigMap
metadata:
  name: ${f.name}
  namespace: ${ns}`
  
  const validLabels = f.labelItems.filter(i => i.key && i.value)
  if (validLabels.length > 0) {
    yaml += `\n  labels:`
    validLabels.forEach(i => { yaml += `\n    ${i.key}: ${i.value}` })
  }
  
  const validData = f.dataItems.filter(i => i.key)
  if (validData.length > 0) {
    yaml += `\ndata:`
    validData.forEach(i => {
      if (i.value.includes('\n')) {
        yaml += `\n  ${i.key}: |\n${i.value.split('\n').map(l => '    ' + l).join('\n')}`
      } else {
        yaml += `\n  ${i.key}: "${i.value}"`
      }
    })
  }
  
  return yaml
}

// 构建 Secret YAML
export const buildSecretYAML = (f: ResourceFormData, ns: string): string => {
  let yaml = `apiVersion: v1
kind: Secret
metadata:
  name: ${f.name}
  namespace: ${ns}
type: ${f.secretType}`
  
  switch (f.secretType) {
    case 'kubernetes.io/tls':
      yaml += `\nstringData:\n  tls.crt: |\n${f.tlsCert.split('\n').map(l => '    ' + l).join('\n')}\n  tls.key: |\n${f.tlsKey.split('\n').map(l => '    ' + l).join('\n')}`
      break
    case 'kubernetes.io/dockerconfigjson':
      const auth = btoa(`${f.dockerUsername}:${f.dockerPassword}`)
      const dockerConfig = { auths: { [f.dockerServer]: { username: f.dockerUsername, password: f.dockerPassword, email: f.dockerEmail, auth } } }
      yaml += `\nstringData:\n  .dockerconfigjson: '${JSON.stringify(dockerConfig)}'`
      break
    case 'kubernetes.io/basic-auth':
      yaml += `\nstringData:\n  username: ${f.basicUsername}\n  password: ${f.basicPassword}`
      break
    case 'kubernetes.io/ssh-auth':
      yaml += `\nstringData:\n  ssh-privatekey: |\n${f.sshPrivateKey.split('\n').map(l => '    ' + l).join('\n')}`
      break
    default:
      const validData = f.dataItems.filter(i => i.key)
      if (validData.length > 0) {
        yaml += `\nstringData:`
        validData.forEach(i => { yaml += `\n  ${i.key}: "${i.value}"` })
      }
  }
  
  return yaml
}

// 构建 Ingress YAML
export const buildIngressYAML = (f: ResourceFormData, ns: string): string => {
  let yaml = `apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ${f.name}
  namespace: ${ns}`
  
  const validAnnotations = f.annotationItems.filter(i => i.key && i.value)
  if (validAnnotations.length > 0) {
    yaml += `\n  annotations:`
    validAnnotations.forEach(i => { yaml += `\n    ${i.key}: "${i.value}"` })
  }
  
  yaml += `\nspec:`
  if (f.ingressClass) yaml += `\n  ingressClassName: ${f.ingressClass}`
  
  const validTls = f.tlsConfigs.filter(t => t.hosts && t.secretName)
  if (validTls.length > 0) {
    yaml += `\n  tls:`
    validTls.forEach(t => {
      yaml += `\n  - hosts:`
      t.hosts.split(',').forEach(h => { yaml += `\n    - ${h.trim()}` })
      yaml += `\n    secretName: ${t.secretName}`
    })
  }
  
  yaml += `\n  rules:`
  f.ingressRules.forEach(rule => {
    yaml += `\n  - host: ${rule.host}\n    http:\n      paths:`
    rule.paths.forEach(p => {
      yaml += `\n      - path: ${p.path}\n        pathType: ${p.pathType}\n        backend:\n          service:\n            name: ${p.serviceName}\n            port:\n              number: ${p.servicePort}`
    })
  })
  
  return yaml
}

// 构建简单资源 YAML
export const buildSimpleYAML = (f: ResourceFormData, ns: string, resourceType: string): string => {
  switch (resourceType) {
    case 'statefulsets':
      return `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: ${f.name}
  namespace: ${ns}
spec:
  serviceName: ${f.serviceName || f.name}
  replicas: ${f.replicas}
  selector:
    matchLabels:
      app: ${f.name}
  template:
    metadata:
      labels:
        app: ${f.name}
    spec:
      containers:
      - name: ${f.name}
        image: ${f.image}
        ports:
        - containerPort: ${f.containerPort}`
    
    case 'daemonsets':
      return `apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: ${f.name}
  namespace: ${ns}
spec:
  selector:
    matchLabels:
      app: ${f.name}
  template:
    metadata:
      labels:
        app: ${f.name}
    spec:
      containers:
      - name: ${f.name}
        image: ${f.image}`
    
    case 'jobs': {
      const cmdLine = f.command ? `        command: [${f.command.split(',').map(s => `"${s.trim()}"`).join(', ')}]` : ''
      return `apiVersion: batch/v1
kind: Job
metadata:
  name: ${f.name}
  namespace: ${ns}
spec:
  backoffLimit: ${f.backoffLimit}
  template:
    spec:
      containers:
      - name: ${f.name}
        image: ${f.image}
${cmdLine}
      restartPolicy: Never`
    }
    
    case 'cronjobs': {
      const cmdLine = f.command ? `          command: [${f.command.split(',').map(s => `"${s.trim()}"`).join(', ')}]` : ''
      return `apiVersion: batch/v1
kind: CronJob
metadata:
  name: ${f.name}
  namespace: ${ns}
spec:
  schedule: "${f.schedule}"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: ${f.name}
            image: ${f.image}
${cmdLine}
          restartPolicy: OnFailure`
    }
    
    case 'pvcs': {
      const scLine = f.storageClass ? `  storageClassName: ${f.storageClass}` : ''
      return `apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: ${f.name}
  namespace: ${ns}
spec:
${scLine}
  accessModes:
  - ${f.accessMode}
  resources:
    requests:
      storage: ${f.storage}`
    }
    
    default:
      return ''
  }
}

// 主构建函数
export const buildYAMLFromForm = (f: ResourceFormData, ns: string, resourceType: string): string => {
  switch (resourceType) {
    case 'deployments': return buildDeploymentYAML(f, ns)
    case 'services': return buildServiceYAML(f, ns)
    case 'configmaps': return buildConfigMapYAML(f, ns)
    case 'secrets': return buildSecretYAML(f, ns)
    case 'ingresses': return buildIngressYAML(f, ns)
    default: return buildSimpleYAML(f, ns, resourceType)
  }
}

// 表单验证
export const validateForm = (f: ResourceFormData, resourceType: string): string | null => {
  if (!f.name) return '请输入资源名称'
  if (!/^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/.test(f.name)) {
    return '名称只能包含小写字母、数字和连字符，且必须以字母或数字开头和结尾'
  }
  
  switch (resourceType) {
    case 'deployments':
      if (f.containers.length === 0) return '至少需要一个容器'
      for (let i = 0; i < f.containers.length; i++) {
        if (!f.containers[i].image) return `容器 ${i + 1} 的镜像不能为空`
      }
      break
    case 'statefulsets':
    case 'daemonsets':
    case 'jobs':
    case 'cronjobs':
      if (!f.image) return '请输入镜像地址'
      if (resourceType === 'cronjobs' && !f.schedule) return '请输入调度表达式'
      break
    case 'services':
      if (f.serviceType === 'ExternalName') {
        if (!f.externalName) return '请输入外部名称'
      } else {
        const validSelectors = f.selectorItems.filter(i => i.key && i.value)
        if (validSelectors.length === 0) return '请至少添加一个选择器'
        if (f.servicePorts.length === 0) return '请至少添加一个端口'
        for (let i = 0; i < f.servicePorts.length; i++) {
          if (!f.servicePorts[i].port) return `端口 ${i + 1} 的端口号不能为空`
        }
      }
      break
    case 'secrets':
      switch (f.secretType) {
        case 'kubernetes.io/tls':
          if (!f.tlsCert) return '请输入 TLS 证书'
          if (!f.tlsKey) return '请输入 TLS 私钥'
          break
        case 'kubernetes.io/dockerconfigjson':
          if (!f.dockerServer) return '请输入 Registry 地址'
          if (!f.dockerUsername) return '请输入用户名'
          if (!f.dockerPassword) return '请输入密码'
          break
        case 'kubernetes.io/basic-auth':
          if (!f.basicUsername) return '请输入用户名'
          if (!f.basicPassword) return '请输入密码'
          break
        case 'kubernetes.io/ssh-auth':
          if (!f.sshPrivateKey) return '请输入 SSH 私钥'
          break
      }
      break
    case 'ingresses':
      if (f.ingressRules.length === 0) return '请至少添加一条规则'
      for (let i = 0; i < f.ingressRules.length; i++) {
        const rule = f.ingressRules[i]
        if (!rule.host) return `规则 ${i + 1} 的域名不能为空`
        for (let j = 0; j < rule.paths.length; j++) {
          const path = rule.paths[j]
          if (!path.serviceName) return `规则 ${i + 1} 路径 ${j + 1} 的服务名称不能为空`
          if (!path.servicePort) return `规则 ${i + 1} 路径 ${j + 1} 的服务端口不能为空`
        }
      }
      break
    case 'pvcs':
      if (!f.storage) return '请输入存储大小'
      break
  }
  
  return null
}

// YAML 校验
export const validateYAML = (content: string): string => {
  if (!content.trim()) return ''
  
  const lines = content.split('\n')
  let hasApiVersion = false, hasKind = false, hasMetadata = false, hasName = false
  
  for (const line of lines) {
    const trimmed = line.trim()
    if (trimmed.startsWith('apiVersion:')) hasApiVersion = true
    if (trimmed.startsWith('kind:')) hasKind = true
    if (trimmed.startsWith('metadata:')) hasMetadata = true
    if (trimmed.startsWith('name:') && hasMetadata) hasName = true
  }
  
  if (!hasApiVersion) return '缺少 apiVersion 字段'
  if (!hasKind) return '缺少 kind 字段'
  if (!hasMetadata) return '缺少 metadata 字段'
  if (!hasName) return '缺少 metadata.name 字段'
  
  return ''
}
