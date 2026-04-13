# 🔨 基础设施模块 (Infrastructure Module)

## 功能概述
负责Jenkins集成和Kubernetes集群管理。

## 文件结构
```
infrastructure/
├── handler/                           # HTTP处理器
│   ├── jenkins_instance_handler.go   # Jenkins实例管理接口
│   ├── jenkins_build_handler.go      # Jenkins构建接口
│   ├── k8s_cluster_handler.go        # K8s集群管理接口
│   ├── k8s_deployment_handler.go     # K8s部署管理接口
│   ├── k8s_pod_handler.go            # K8s Pod管理接口
│   ├── k8s_resource_handler.go       # K8s资源管理接口
│   └── k8s_ops_ioc.go                # K8s操作依赖注入
└── repository/                        # 数据访问层
    ├── jenkins_repo.go               # Jenkins数据操作
    └── k8s_repo.go                   # K8s数据操作
```

## 主要功能
- **Jenkins集成**: 实例管理、构建触发、状态监控
- **K8s集群管理**: 集群配置、连接管理
- **K8s资源管理**: Deployment、Pod、Service管理
- **K8s操作**: 重启、扩缩容、日志查看

## API接口
- `GET /jenkins/instances` - 获取Jenkins实例列表
- `POST /jenkins/instances` - 创建Jenkins实例
- `POST /jenkins/build` - 触发构建
- `GET /k8s/clusters` - 获取K8s集群列表
- `POST /k8s/clusters` - 创建K8s集群
- `GET /k8s/pods` - 获取Pod列表
- `POST /k8s/restart` - 重启应用

## 相关Service
- `internal/service/jenkins/` - Jenkins业务逻辑
- `internal/service/kubernetes/` - K8s业务逻辑