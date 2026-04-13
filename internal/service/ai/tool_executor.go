package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"

	"devops/internal/models/ai"
	"devops/internal/repository"
	"devops/pkg/llm"
)

// ToolExecutor 工具执行器
type ToolExecutor struct {
	db           *gorm.DB
	opLogRepo    *repository.AIOperationLogRepository
	tools        map[string]Tool
	mu           sync.RWMutex
	permChecker  PermissionChecker
}

// Tool 工具接口
type Tool interface {
	// Name 工具名称
	Name() string
	// Description 工具描述
	Description() string
	// Parameters 参数定义（JSON Schema）
	Parameters() map[string]interface{}
	// Execute 执行工具
	Execute(ctx context.Context, userID uint, params map[string]interface{}) (interface{}, error)
	// RequiredPermissions 所需权限
	RequiredPermissions() []string
	// IsDangerous 是否为危险操作
	IsDangerous() bool
}

// PermissionChecker 权限检查器接口
type PermissionChecker interface {
	HasPermission(ctx context.Context, userID uint, permission string) bool
	HasAnyPermission(ctx context.Context, userID uint, permissions []string) bool
}

// DefaultPermissionChecker 默认权限检查器（允许所有操作）
type DefaultPermissionChecker struct{}

func (d *DefaultPermissionChecker) HasPermission(ctx context.Context, userID uint, permission string) bool {
	return true
}

func (d *DefaultPermissionChecker) HasAnyPermission(ctx context.Context, userID uint, permissions []string) bool {
	return true
}

// NewToolExecutor 创建工具执行器
func NewToolExecutor(db *gorm.DB, permChecker PermissionChecker) *ToolExecutor {
	if permChecker == nil {
		permChecker = &DefaultPermissionChecker{}
	}

	executor := &ToolExecutor{
		db:          db,
		opLogRepo:   repository.NewAIOperationLogRepository(db),
		tools:       make(map[string]Tool),
		permChecker: permChecker,
	}

	// 注册内置工具
	executor.registerBuiltinTools()

	return executor
}

// registerBuiltinTools 注册内置工具
func (e *ToolExecutor) registerBuiltinTools() {
	// 查询类工具
	e.RegisterTool(&QueryLogsTool{db: e.db})
	e.RegisterTool(&QueryAlertsTool{db: e.db})
	e.RegisterTool(&QueryMetricsTool{db: e.db})

	// 操作类工具
	e.RegisterTool(&RestartAppTool{db: e.db})
	e.RegisterTool(&ScalePodTool{db: e.db})
	e.RegisterTool(&RollbackTool{db: e.db})
	e.RegisterTool(&SilenceAlertTool{db: e.db})
}

// RegisterTool 注册工具
func (e *ToolExecutor) RegisterTool(tool Tool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.tools[tool.Name()] = tool
}

// GetTool 获取工具
func (e *ToolExecutor) GetTool(name string) (Tool, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	tool, ok := e.tools[name]
	return tool, ok
}

// Execute 执行工具
func (e *ToolExecutor) Execute(ctx context.Context, userID uint, username string, toolName string, params map[string]interface{}, conversationID, messageID string) (*ExecuteResult, error) {
	startTime := time.Now()

	// 获取工具
	tool, ok := e.GetTool(toolName)
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", toolName)
	}

	// 检查权限
	requiredPerms := tool.RequiredPermissions()
	if len(requiredPerms) > 0 && !e.permChecker.HasAnyPermission(ctx, userID, requiredPerms) {
		return &ExecuteResult{
			Success:      false,
			NeedConfirm:  false,
			ErrorMessage: "权限不足，无法执行此操作",
		}, nil
	}

	// 检查是否需要确认
	if tool.IsDangerous() {
		confirmed, ok := params["_confirmed"].(bool)
		if !ok || !confirmed {
			return &ExecuteResult{
				Success:      false,
				NeedConfirm:  true,
				ConfirmMsg:   fmt.Sprintf("此操作（%s）可能影响系统运行，是否确认执行？", tool.Description()),
				ToolName:     toolName,
				Params:       params,
			}, nil
		}
	}

	// 执行工具
	result, err := tool.Execute(ctx, userID, params)
	duration := time.Since(startTime)

	// 记录操作日志
	opLog := &ai.AIOperationLog{
		UserID:         userID,
		Username:       username,
		Action:         ai.OperationAction(toolName),
		ActionName:     tool.Description(),
		DurationMs:     int(duration.Milliseconds()),
	}

	if conversationID != "" {
		opLog.ConversationID = &conversationID
	}
	if messageID != "" {
		opLog.MessageID = &messageID
	}

	// 序列化参数
	if paramsJSON, err := json.Marshal(params); err == nil {
		opLog.Params = make(ai.JSONData)
		json.Unmarshal(paramsJSON, &opLog.Params)
	}

	if err != nil {
		opLog.Success = false
		opLog.ErrorMsg = err.Error()
		e.opLogRepo.Create(ctx, opLog)
		return &ExecuteResult{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	opLog.Success = true
	if resultJSON, err := json.Marshal(result); err == nil {
		opLog.Result = make(ai.JSONData)
		json.Unmarshal(resultJSON, &opLog.Result)
	}
	e.opLogRepo.Create(ctx, opLog)

	return &ExecuteResult{
		Success: true,
		Data:    result,
	}, nil
}

// GetAvailableTools 获取用户可用的工具列表
func (e *ToolExecutor) GetAvailableTools(ctx context.Context, userID uint) []llm.Tool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var tools []llm.Tool
	for _, tool := range e.tools {
		// 检查权限
		requiredPerms := tool.RequiredPermissions()
		if len(requiredPerms) > 0 && !e.permChecker.HasAnyPermission(ctx, userID, requiredPerms) {
			continue
		}

		tools = append(tools, llm.Tool{
			Type: "function",
			Function: llm.FunctionDef{
				Name:        tool.Name(),
				Description: tool.Description(),
				Parameters:  tool.Parameters(),
			},
		})
	}

	return tools
}

// GetAllToolDefinitions 获取所有工具定义
func (e *ToolExecutor) GetAllToolDefinitions() []llm.Tool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var tools []llm.Tool
	for _, tool := range e.tools {
		tools = append(tools, llm.Tool{
			Type: "function",
			Function: llm.FunctionDef{
				Name:        tool.Name(),
				Description: tool.Description(),
				Parameters:  tool.Parameters(),
			},
		})
	}

	return tools
}

// ExecuteResult 执行结果
type ExecuteResult struct {
	Success      bool                   `json:"success"`
	NeedConfirm  bool                   `json:"need_confirm,omitempty"`
	ConfirmMsg   string                 `json:"confirm_msg,omitempty"`
	ToolName     string                 `json:"tool_name,omitempty"`
	Params       map[string]interface{} `json:"params,omitempty"`
	Data         interface{}            `json:"data,omitempty"`
	ErrorMessage string                 `json:"error_message,omitempty"`
}

// ParseToolCall 解析工具调用
func ParseToolCall(toolCall *llm.ToolCall) (string, map[string]interface{}, error) {
	if toolCall == nil {
		return "", nil, fmt.Errorf("tool call is nil")
	}

	var params map[string]interface{}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params); err != nil {
		return "", nil, fmt.Errorf("parse arguments: %w", err)
	}

	return toolCall.Function.Name, params, nil
}
