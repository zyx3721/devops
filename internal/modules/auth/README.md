# 🔐 认证授权模块 (Auth Module)

## 功能概述
负责用户认证、权限管理和RBAC权限控制。

## 文件结构
```
auth/
├── handler/                    # HTTP处理器
│   ├── auth_handler.go        # 登录认证接口
│   ├── user_handler.go        # 用户管理接口
│   └── rbac_handler.go        # 权限管理接口
└── repository/                 # 数据访问层
    ├── user_repo.go           # 用户数据操作
    └── rbac_repo.go           # 权限数据操作
```

## 主要功能
- **用户认证**: 登录、登出、Token验证
- **用户管理**: 用户CRUD、密码管理、状态管理
- **权限控制**: 角色管理、权限分配、RBAC控制

## API接口
- `POST /auth/login` - 用户登录
- `POST /auth/logout` - 用户登出
- `GET /users` - 获取用户列表
- `POST /users` - 创建用户
- `PUT /users/:id` - 更新用户
- `DELETE /users/:id` - 删除用户
- `GET /roles` - 获取角色列表
- `POST /roles` - 创建角色

## 相关Service
- `internal/service/user/` - 用户业务逻辑