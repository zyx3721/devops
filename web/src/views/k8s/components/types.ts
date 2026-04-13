// 容器配置
export interface ContainerConfig {
  name: string
  image: string
  imagePullPolicy: string
  ports: { name: string; containerPort: number; protocol: string }[]
  envs: { name: string; value: string }[]
  resources: { cpuRequest: string; cpuLimit: string; memoryRequest: string; memoryLimit: string }
  command: string
  args: string
  volumeMounts: { name: string; mountPath: string; readOnly: boolean }[]
}

// Deployment 表单数据
export interface DeploymentFormData {
  name: string
  replicas: number
  labelItems: { key: string; value: string }[]
  containers: ContainerConfig[]
  volumes: { name: string; type: string; source: string }[]
  imagePullSecrets: string
  serviceAccount: string
  nodeSelectorItems: { key: string; value: string }[]
}

// Service 表单数据
export interface ServiceFormData {
  name: string
  serviceType: string
  selectorItems: { key: string; value: string }[]
  servicePorts: { name: string; port: number; targetPort: string; protocol: string; nodePort?: number }[]
  sessionAffinity: string
  loadBalancerIP: string
  externalName: string
}

// ConfigMap 表单数据
export interface ConfigMapFormData {
  name: string
  labelItems: { key: string; value: string }[]
  dataItems: { key: string; value: string }[]
}

// Secret 表单数据
export interface SecretFormData {
  name: string
  secretType: string
  dataItems: { key: string; value: string }[]
  tlsCert: string
  tlsKey: string
  dockerServer: string
  dockerUsername: string
  dockerPassword: string
  dockerEmail: string
  basicUsername: string
  basicPassword: string
  sshPrivateKey: string
}

// Ingress 表单数据
export interface IngressFormData {
  name: string
  ingressClass: string
  tlsConfigs: { hosts: string; secretName: string }[]
  ingressRules: { host: string; paths: { path: string; pathType: string; serviceName: string; servicePort: number }[] }[]
  annotationItems: { key: string; value: string }[]
}

// 简单资源表单数据
export interface SimpleFormData {
  name: string
  image: string
  replicas: number
  serviceName: string
  containerPort: number
  command: string
  backoffLimit: number
  schedule: string
  storage: string
  accessMode: string
  storageClass: string
}

// 完整表单数据
export interface ResourceFormData extends DeploymentFormData, ServiceFormData, ConfigMapFormData, SecretFormData, IngressFormData, SimpleFormData {}

// 创建默认容器
export const createDefaultContainer = (): ContainerConfig => ({
  name: '',
  image: '',
  imagePullPolicy: 'IfNotPresent',
  ports: [{ name: '', containerPort: 80, protocol: 'TCP' }],
  envs: [],
  resources: { cpuRequest: '', cpuLimit: '', memoryRequest: '', memoryLimit: '' },
  command: '',
  args: '',
  volumeMounts: []
})

// 创建默认表单数据
export const createDefaultFormData = (): ResourceFormData => ({
  name: '',
  replicas: 1,
  image: '',
  containerPort: 80,
  serviceName: '',
  serviceType: 'ClusterIP',
  secretType: 'Opaque',
  ingressClass: '',
  command: '',
  backoffLimit: 4,
  schedule: '*/5 * * * *',
  storage: '1Gi',
  accessMode: 'ReadWriteOnce',
  storageClass: '',
  labelItems: [{ key: 'app', value: '' }],
  containers: [createDefaultContainer()],
  volumes: [],
  imagePullSecrets: '',
  serviceAccount: '',
  nodeSelectorItems: [],
  selectorItems: [{ key: 'app', value: '' }],
  servicePorts: [{ name: '', port: 80, targetPort: '80', protocol: 'TCP', nodePort: undefined }],
  sessionAffinity: 'None',
  loadBalancerIP: '',
  externalName: '',
  dataItems: [{ key: '', value: '' }],
  tlsCert: '',
  tlsKey: '',
  dockerServer: '',
  dockerUsername: '',
  dockerPassword: '',
  dockerEmail: '',
  basicUsername: '',
  basicPassword: '',
  sshPrivateKey: '',
  tlsConfigs: [],
  ingressRules: [{ host: '', paths: [{ path: '/', pathType: 'Prefix', serviceName: '', servicePort: 80 }] }],
  annotationItems: []
})
