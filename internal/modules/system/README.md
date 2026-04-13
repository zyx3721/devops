# ⚙️ 系统管理模块 (System Module)

## 功能概述
负责系统仪表盘、审计日志和OA系统集成。

## 文件结构
```
system/
├── handler/                     # HTTP处理器
│   ├── dashboard_handler.go    # 仪表盘接口
│   ├── audit_handler.go        # 审计日志接口
│   ├── oa_handler.go           # OA集成接口
│   └── oa_jenkins_handler.go   # OA-Jenkins集成接口
└── repository/                  # 数据访问层
    ├── audit_repo.go           # 审计日志数据操作
    └── oa_repo.go              # OA数据操作
```

## 主要功能
- **系统仪表盘**: 统计数据、图表展示
- **审计日志**: 操作记录、日志查询
- **OA集成**: OA系统对接、数据同步

## API接口
- `GET /dashboard/stats` - 获取仪表盘统计
- `GET /audit/logs` - 获取审计日志
- `POST /oa/webhook` - OA系统回调
- `GET /oa/data` - 获取OA数据

## 相关Service
- `internal/service/oa/` - OA业务逻辑