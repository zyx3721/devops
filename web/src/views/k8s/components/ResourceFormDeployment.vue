<template>
  <a-form :label-col="{ span: 5 }" :wrapper-col="{ span: 18 }">
    <a-divider orientation="left">基本信息</a-divider>
    <a-form-item label="名称" required>
      <a-input v-model:value="form.name" placeholder="资源名称" />
    </a-form-item>
    <a-form-item label="副本数">
      <a-input-number v-model:value="form.replicas" :min="0" :max="100" style="width: 120px" />
    </a-form-item>
    <a-form-item label="标签">
      <div v-for="(item, index) in form.labelItems" :key="'label'+index" style="display: flex; gap: 8px; margin-bottom: 8px">
        <a-input v-model:value="item.key" placeholder="Key" style="width: 150px" />
        <a-input v-model:value="item.value" placeholder="Value" style="width: 200px" />
        <a-button @click="removeLabelItem(index)" danger size="small"><MinusOutlined /></a-button>
      </div>
      <a-button @click="addLabelItem" type="dashed" size="small"><PlusOutlined /> 添加标签</a-button>
    </a-form-item>

    <a-divider orientation="left">容器配置</a-divider>
    <div v-for="(container, cIdx) in form.containers" :key="'container'+cIdx" 
         style="border: 1px solid #f0f0f0; padding: 12px; margin-bottom: 12px; border-radius: 4px">
      <div style="display: flex; justify-content: space-between; margin-bottom: 8px">
        <span style="font-weight: 500">容器 {{ cIdx + 1 }}</span>
        <a-button v-if="form.containers.length > 1" @click="removeContainer(cIdx)" danger size="small">
          <MinusOutlined />
        </a-button>
      </div>
      <a-form-item label="容器名称" required>
        <a-input v-model:value="container.name" placeholder="容器名称" />
      </a-form-item>
      <a-form-item label="镜像" required>
        <a-input v-model:value="container.image" placeholder="如 nginx:latest" />
      </a-form-item>
      <a-form-item label="镜像拉取策略">
        <a-select v-model:value="container.imagePullPolicy" style="width: 150px">
          <a-select-option value="IfNotPresent">IfNotPresent</a-select-option>
          <a-select-option value="Always">Always</a-select-option>
          <a-select-option value="Never">Never</a-select-option>
        </a-select>
      </a-form-item>
      <a-form-item label="端口">
        <div v-for="(port, pIdx) in container.ports" :key="'port'+pIdx" style="display: flex; gap: 8px; margin-bottom: 8px">
          <a-input v-model:value="port.name" placeholder="名称" style="width: 100px" />
          <a-input-number v-model:value="port.containerPort" placeholder="端口" :min="1" :max="65535" style="width: 100px" />
          <a-select v-model:value="port.protocol" style="width: 80px">
            <a-select-option value="TCP">TCP</a-select-option>
            <a-select-option value="UDP">UDP</a-select-option>
          </a-select>
          <a-button @click="removePort(container, pIdx)" danger size="small"><MinusOutlined /></a-button>
        </div>
        <a-button @click="addPort(container)" type="dashed" size="small"><PlusOutlined /> 添加端口</a-button>
      </a-form-item>
      <a-form-item label="环境变量">
        <div v-for="(env, eIdx) in container.envs" :key="'env'+eIdx" style="display: flex; gap: 8px; margin-bottom: 8px">
          <a-input v-model:value="env.name" placeholder="变量名" style="width: 150px" />
          <a-input v-model:value="env.value" placeholder="值" style="flex: 1" />
          <a-button @click="removeEnv(container, eIdx)" danger size="small"><MinusOutlined /></a-button>
        </div>
        <a-button @click="addEnv(container)" type="dashed" size="small"><PlusOutlined /> 添加环境变量</a-button>
      </a-form-item>
      <a-form-item label="资源限制">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-input v-model:value="container.resources.cpuRequest" placeholder="如 100m" addon-before="CPU请求" />
          </a-col>
          <a-col :span="12">
            <a-input v-model:value="container.resources.cpuLimit" placeholder="如 500m" addon-before="CPU限制" />
          </a-col>
        </a-row>
        <a-row :gutter="16" style="margin-top: 8px">
          <a-col :span="12">
            <a-input v-model:value="container.resources.memoryRequest" placeholder="如 128Mi" addon-before="内存请求" />
          </a-col>
          <a-col :span="12">
            <a-input v-model:value="container.resources.memoryLimit" placeholder="如 256Mi" addon-before="内存限制" />
          </a-col>
        </a-row>
      </a-form-item>
      <a-form-item label="命令">
        <a-input v-model:value="container.command" placeholder="命令，逗号分隔，如: /bin/sh,-c,echo hello" />
      </a-form-item>
      <a-form-item label="参数">
        <a-input v-model:value="container.args" placeholder="参数，逗号分隔" />
      </a-form-item>
      <a-form-item label="卷挂载">
        <div v-for="(mount, mIdx) in container.volumeMounts" :key="'mount'+mIdx" style="display: flex; gap: 8px; margin-bottom: 8px">
          <a-input v-model:value="mount.name" placeholder="卷名称" style="width: 150px" />
          <a-input v-model:value="mount.mountPath" placeholder="挂载路径" style="flex: 1" />
          <a-checkbox v-model:checked="mount.readOnly">只读</a-checkbox>
          <a-button @click="removeVolumeMount(container, mIdx)" danger size="small"><MinusOutlined /></a-button>
        </div>
        <a-button @click="addVolumeMount(container)" type="dashed" size="small"><PlusOutlined /> 添加挂载</a-button>
      </a-form-item>
    </div>
    <a-button @click="addContainer" type="dashed" block><PlusOutlined /> 添加容器</a-button>

    <a-divider orientation="left">卷配置</a-divider>
    <div v-for="(vol, vIdx) in form.volumes" :key="'vol'+vIdx" style="display: flex; gap: 8px; margin-bottom: 8px; align-items: center">
      <a-input v-model:value="vol.name" placeholder="卷名称" style="width: 150px" />
      <a-select v-model:value="vol.type" style="width: 120px" @change="() => vol.source = ''">
        <a-select-option value="emptyDir">EmptyDir</a-select-option>
        <a-select-option value="configMap">ConfigMap</a-select-option>
        <a-select-option value="secret">Secret</a-select-option>
        <a-select-option value="pvc">PVC</a-select-option>
        <a-select-option value="hostPath">HostPath</a-select-option>
      </a-select>
      <a-input v-if="vol.type !== 'emptyDir'" v-model:value="vol.source" 
               :placeholder="vol.type === 'hostPath' ? '主机路径' : '资源名称'" style="flex: 1" />
      <a-button @click="removeVolume(vIdx)" danger size="small"><MinusOutlined /></a-button>
    </div>
    <a-button @click="addVolume" type="dashed" size="small"><PlusOutlined /> 添加卷</a-button>

    <a-divider orientation="left">高级配置</a-divider>
    <a-form-item label="镜像拉取密钥">
      <a-input v-model:value="form.imagePullSecrets" placeholder="Secret 名称，多个用逗号分隔" />
    </a-form-item>
    <a-form-item label="服务账号">
      <a-input v-model:value="form.serviceAccount" placeholder="ServiceAccount 名称" />
    </a-form-item>
    <a-form-item label="节点选择器">
      <div v-for="(item, index) in form.nodeSelectorItems" :key="'ns'+index" style="display: flex; gap: 8px; margin-bottom: 8px">
        <a-input v-model:value="item.key" placeholder="Key" style="width: 150px" />
        <a-input v-model:value="item.value" placeholder="Value" style="width: 200px" />
        <a-button @click="removeNodeSelectorItem(index)" danger size="small"><MinusOutlined /></a-button>
      </div>
      <a-button @click="addNodeSelectorItem" type="dashed" size="small"><PlusOutlined /> 添加选择器</a-button>
    </a-form-item>
  </a-form>
</template>

<script setup lang="ts">
import { PlusOutlined, MinusOutlined } from '@ant-design/icons-vue'
import type { DeploymentFormData, ContainerConfig } from './types'

const props = defineProps<{ form: DeploymentFormData }>()

const createDefaultContainer = (): ContainerConfig => ({
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

// 标签操作
const addLabelItem = () => props.form.labelItems.push({ key: '', value: '' })
const removeLabelItem = (index: number) => {
  props.form.labelItems.splice(index, 1)
  if (props.form.labelItems.length === 0) props.form.labelItems.push({ key: '', value: '' })
}

// 容器操作
const addContainer = () => props.form.containers.push(createDefaultContainer())
const removeContainer = (index: number) => props.form.containers.splice(index, 1)
const addPort = (c: ContainerConfig) => c.ports.push({ name: '', containerPort: 80, protocol: 'TCP' })
const removePort = (c: ContainerConfig, i: number) => c.ports.splice(i, 1)
const addEnv = (c: ContainerConfig) => c.envs.push({ name: '', value: '' })
const removeEnv = (c: ContainerConfig, i: number) => c.envs.splice(i, 1)
const addVolumeMount = (c: ContainerConfig) => c.volumeMounts.push({ name: '', mountPath: '', readOnly: false })
const removeVolumeMount = (c: ContainerConfig, i: number) => c.volumeMounts.splice(i, 1)

// 卷操作
const addVolume = () => props.form.volumes.push({ name: '', type: 'emptyDir', source: '' })
const removeVolume = (i: number) => props.form.volumes.splice(i, 1)

// 节点选择器
const addNodeSelectorItem = () => props.form.nodeSelectorItems.push({ key: '', value: '' })
const removeNodeSelectorItem = (i: number) => props.form.nodeSelectorItems.splice(i, 1)
</script>
