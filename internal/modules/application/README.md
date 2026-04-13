# 📱 应用管理模块 (Application Module)

## 功能概述
负责应用配置管理、部署管理和发布锁控制。

## 文件结构
```
application/
├── handler/                        # HTTP处理器
│   ├── application_handler.go     # 应用管理接口
│   ├── deploy_handler.go          # 部署管理接口
│   └── deploy_lock_handler.go     # 发布锁接口
└── repository/                     # 数据访问层
    ├── application_repo.go        # 应用数据操作
    └── deploy_lock_repo.go        # 发布锁数据操作
```

## 主要功能
- **应用管理**: 应用CRUD、环境配置、团队管理
- **部署管理**: 部署记录、版本管理、回滚操作
- **发布锁**: 防止并发部署、锁定管理

## API接口
- `GET /applications` - 获取应用列表
- `POST /applications` - 创建应用
- `PUT /applications/:id` - 更新应用
- `DELETE /applications/:id` - 删除应用
- `POST /deploy` - 执行部署
- `POST /deploy/rollback` - 回滚部署
- `GET /deploy/locks` - 获取发布锁

## 相关Service
- `internal/service/deploy/` - 部署业务逻辑