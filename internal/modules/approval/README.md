# ✅ 审批流程模块 (Approval Module)

## 功能概述
负责多级审批链、审批规则配置和发布窗口管理。

## 文件结构
```
approval/
├── handler/                           # HTTP处理器
│   ├── approval_handler.go           # 审批操作接口
│   ├── approval_chain_handler.go     # 审批链管理接口
│   ├── approval_rule_handler.go      # 审批规则接口
│   ├── approval_ioc.go               # 依赖注入配置
│   └── deploy_window_handler.go      # 发布窗口接口
└── repository/                        # 数据访问层
    ├── approval_chain_repo.go        # 审批链数据操作
    ├── approval_instance_repo.go     # 审批实例数据操作
    ├── approval_node_repo.go         # 审批节点数据操作
    ├── approval_node_instance_repo.go # 节点实例数据操作
    ├── approval_action_repo.go       # 审批动作数据操作
    ├── approval_rule_repo.go         # 审批规则数据操作
    └── deploy_window_repo.go         # 发布窗口数据操作
```

## 主要功能
- **审批链管理**: 多级审批链设计、节点配置
- **审批流程**: 审批实例执行、状态流转
- **审批规则**: 规则配置、条件匹配
- **发布窗口**: 时间窗口控制、紧急发布

## API接口
- `GET /approval/chains` - 获取审批链列表
- `POST /approval/chains` - 创建审批链
- `POST /approval/approve` - 执行审批
- `POST /approval/reject` - 拒绝审批
- `GET /approval/pending` - 获取待审批列表
- `GET /deploy/windows` - 获取发布窗口

## 相关Service
- `internal/service/approval/` - 审批业务逻辑