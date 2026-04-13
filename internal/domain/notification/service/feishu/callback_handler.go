package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"devops/internal/service/jenkins"

	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher/callback"
)

// GlobalClient 全局飞书客户端
var GlobalClient *Client

// InitCallbackHandler 初始化回调处理器
func InitCallbackHandler(client *Client) {
	GlobalClient = client
	SetCardActionHandler(handleCardAction)
}

// ApprovalCallbackHandler 审批回调处理器接口
type ApprovalCallbackHandler interface {
	HandleApprove(ctx context.Context, nodeInstanceID uint, userID uint, userName string, comment string) error
	HandleReject(ctx context.Context, nodeInstanceID uint, userID uint, userName string, comment string) error
}

// GlobalApprovalHandler 全局审批回调处理器
var GlobalApprovalHandler ApprovalCallbackHandler

// SetApprovalCallbackHandler 设置审批回调处理器
func SetApprovalCallbackHandler(handler ApprovalCallbackHandler) {
	GlobalApprovalHandler = handler
}

func handleCardAction(ctx context.Context, event *callback.CardActionTriggerEvent) (*callback.CardActionTriggerResponse, error) {
	action := event.Event.Action
	if action == nil || action.Value == nil {
		return toast("无效的操作数据"), nil
	}

	valueMap := action.Value

	// 检查是否是审批链的回调
	if approvalAction, ok := valueMap["action"].(string); ok {
		if approvalAction == "approve" || approvalAction == "reject" {
			return handleApprovalChainCallback(ctx, event, valueMap)
		}
	}

	requestID, _ := valueMap["request_id"].(string)
	serviceName, _ := valueMap["service"].(string)
	actionName, _ := valueMap["action"].(string)
	branch, _ := valueMap["branch"].(string)

	if requestID == "" {
		return toast("无法获取请求ID，请重试"), nil
	}

	if GlobalStore.IsActionDisabled(requestID, serviceName, actionName) {
		return toast("该操作已执行，请勿重复点击"), nil
	}

	if actionName != "batch_release_all" && actionName != "stop_batch_release" && actionName != "do_rollback" {
		GlobalStore.IncrementActionCount(requestID, serviceName, actionName)
	}

	if actionName != "do_restart" && actionName != "do_gray_release" && actionName != "batch_release_all" && actionName != "do_official_release" {
		GlobalStore.MarkActionDisabled(requestID, serviceName, actionName)
	}

	switch actionName {
	case "do_gray_release":
		fmt.Printf("Triggering Gray Release: %s, %s\n", serviceName, branch)
		go triggerAndMonitorBuild(context.Background(), serviceName, branch, "Gray", requestID)
	case "do_official_release":
		fmt.Printf("Triggering Official Release: %s, %s\n", serviceName, branch)
		go triggerAndMonitorBuild(context.Background(), serviceName, branch, "Deploy", requestID)
	case "do_rollback":
		fmt.Printf("Triggering Rollback: %s, %s\n", serviceName, branch)
		go triggerAndMonitorBuild(context.Background(), serviceName, branch, "Rollback", requestID)
	case "do_restart":
		fmt.Printf("Triggering Restart: %s, %s\n", serviceName, branch)
		go triggerAndMonitorBuild(context.Background(), serviceName, branch, "Restart", requestID)
	}

	if actionName == "batch_release_all" || actionName == "stop_batch_release" {
		branchMap := make(map[string]string)
		if branches, ok := valueMap["all_branches"]; ok {
			if bm, ok := branches.(map[string]any); ok {
				for k, v := range bm {
					if s, ok := v.(string); ok {
						branchMap[k] = s
					}
				}
			}
		}

		switch actionName {
		case "batch_release_all":
			fmt.Println("--------------------------------------------------------------")
			fmt.Printf("BatchReleaseService(ctx, branches=%v)\n", branchMap)

			reqData, ok := GlobalStore.Get(requestID)
			if !ok {
				fmt.Printf("Error: RequestID %s not found\n", requestID)
				return toast("请求数据不存在"), nil
			}

			for svc, br := range branchMap {
				deployType := "Deploy"

				var targetService *Service
				for _, s := range reqData.OriginalRequest.Services {
					if s.Name == svc {
						targetService = &s
						break
					}
				}

				if targetService != nil {
					for _, act := range targetService.Actions {
						if strings.EqualFold(act, "gray") || act == "灰度" {
							deployType = "Gray"
							break
						}
					}
				}

				fmt.Printf("Batch triggering %s for %s (Branch: %s)\n", deployType, svc, br)
				go triggerAndMonitorBuild(context.Background(), svc, br, deployType, requestID)
			}

		case "stop_batch_release":
			fmt.Println("--------------------------------------------------------------")
			fmt.Printf("StopBatchReleaseService(ctx, branches=%v)\n", branchMap)

			if reqData, ok := GlobalStore.Get(requestID); ok && GlobalClient != nil {
				newrequestID := fmt.Sprintf("req_%d", time.Now().UnixNano())

				newCardReq := reqData.OriginalRequest
				var filteredServices []Service

				for _, s := range reqData.OriginalRequest.Services {
					isOfficialDone := false
					if reqData.ActionCounts != nil {
						if count, ok := reqData.ActionCounts[s.Name+":do_official_release"]; ok && count > 0 {
							isOfficialDone = true
						}
					}

					if isOfficialDone {
						continue
					}

					newService := s
					actions := make([]string, len(s.Actions))
					copy(actions, s.Actions)
					newService.Actions = actions
					branches := make([]string, len(s.Branches))
					copy(branches, s.Branches)
					newService.Branches = branches

					filteredServices = append(filteredServices, newService)
				}
				newCardReq.Services = filteredServices

				updated := false
				if len(newCardReq.Services) > 0 {
					updated = true
				}

				if len(newCardReq.Services) > 0 {
					for i, s := range newCardReq.Services {
						newActions := []string{}
						seenOfficial := false

						for _, a := range s.Actions {
							if strings.EqualFold(a, "gray") || a == "灰度" {
								if !seenOfficial {
									newActions = append(newActions, "official")
									seenOfficial = true
								}
							} else if strings.EqualFold(a, "official") || strings.EqualFold(a, "release") || a == "正式" {
								if !seenOfficial {
									newActions = append(newActions, "official")
									seenOfficial = true
								}
							} else {
								newActions = append(newActions, a)
							}
						}
						newCardReq.Services[i].Actions = newActions
					}

					updated = true
				}

				if updated {
					GlobalStore.Save(newrequestID, newCardReq)

					if newCardReq.ReceiveID != "" && newCardReq.ReceiveIDType != "" {
						cardContent := BuildCard(newCardReq, newrequestID, nil, nil)
						cardBytes, _ := json.Marshal(cardContent)
						GlobalClient.SendMessage(ctx, newCardReq.ReceiveID, newCardReq.ReceiveIDType, "interactive", string(cardBytes))
					}
				}
			}
		}

		if actionName == "stop_batch_release" {
			GlobalStore.MarkActionDisabled(requestID, serviceName, "batch_release_all")
		}

		var serverList map[string]string
		if reqData, ok := GlobalStore.Get(requestID); ok {
			for _, service := range reqData.OriginalRequest.Services {
				if actionName == "stop_batch_release" {
					actionsToDisable := []string{"do_rollback", "do_restart", "do_gray_release"}

					for _, act := range service.Actions {
						var valueAction string
						switch strings.ToLower(act) {
						case "gray", "灰度":
							valueAction = "do_gray_release"
						case "official", "release", "正式":
							valueAction = "do_official_release"
						case "check", "验收":
							valueAction = "do_check"
						case "rollback", "回滚":
							continue
						case "restart", "重启":
							continue
						default:
							valueAction = "do_" + act
						}
						actionsToDisable = append(actionsToDisable, valueAction)
					}

					for _, act := range actionsToDisable {
						GlobalStore.MarkActionDisabled(requestID, service.Name, act)
					}
				} else {
					GlobalStore.IncrementActionCount(requestID, service.Name, "do_gray_release")
					GlobalStore.IncrementActionCount(requestID, service.Name, "do_official_release")
				}

				if serverList == nil {
					serverList = make(map[string]string)
				}
				if len(service.Branches) > 0 {
					serverList[service.Name] = service.Branches[0]
				}
			}
		}
	}

	storedReq, exists := GlobalStore.Get(requestID)

	if !exists {
		return toast("请求数据已过期或不存在"), nil
	}

	displayRequest := storedReq.OriginalRequest
	hasGray := false
	for _, s := range displayRequest.Services {
		for _, a := range s.Actions {
			if strings.EqualFold(a, "gray") || a == "灰度" {
				hasGray = true
				break
			}
		}
		if hasGray {
			break
		}
	}

	if hasGray {
		var filteredServices []Service
		for _, s := range displayRequest.Services {
			hasGrayAction := false
			for _, a := range s.Actions {
				if strings.EqualFold(a, "gray") || a == "灰度" {
					hasGrayAction = true
					break
				}
			}

			if hasGrayAction {
				newService := s
				newActions := []string{}
				for _, a := range s.Actions {
					if strings.EqualFold(a, "official") || strings.EqualFold(a, "release") || a == "正式" {
						continue
					}
					newActions = append(newActions, a)
				}
				newService.Actions = newActions
				filteredServices = append(filteredServices, newService)
			}
		}
		displayRequest.Services = filteredServices
	}

	newCard := BuildCard(displayRequest, requestID, storedReq.DisabledActions, storedReq.ActionCounts)

	if newCard == nil {
		return &callback.CardActionTriggerResponse{
			Toast: &callback.Toast{
				Type:    "success",
				Content: "操作成功",
			},
		}, nil
	}

	return &callback.CardActionTriggerResponse{
		Toast: &callback.Toast{
			Type:    "success",
			Content: "操作成功",
		},
		Card: &callback.Card{
			Type: "raw",
			Data: newCard,
		},
	}, nil
}

func toast(msg string) *callback.CardActionTriggerResponse {
	return &callback.CardActionTriggerResponse{
		Toast: &callback.Toast{
			Type:    "info",
			Content: msg,
		},
	}
}

// triggerAndMonitorBuild 触发 Jenkins 构建并监控直到完成
func triggerAndMonitorBuild(ctx context.Context, jobName, branch, deployType, requestID string) {
	var receiveID, receiveIDType string
	if reqData, ok := GlobalStore.Get(requestID); ok {
		receiveID = reqData.OriginalRequest.ReceiveID
		receiveIDType = reqData.OriginalRequest.ReceiveIDType
	} else {
		fmt.Printf("Error: RequestID %s not found in store, cannot send notifications\n", requestID)
		return
	}

	client := jenkins.NewClient()
	if client == nil {
		sendFeishuMessage(ctx, receiveID, receiveIDType, fmt.Sprintf("❌ Jenkins 初始化失败: %s", jobName))
		return
	}

	req := jenkins.BuildRequest{
		JobName:    jobName,
		Branch:     branch,
		DeployType: deployType,
	}

	queueID, err := client.Build(ctx, req)
	if err != nil {
		sendFeishuMessage(ctx, receiveID, receiveIDType, fmt.Sprintf("❌ 构建触发失败: %s\nBranch: %s\nType: %s\nError: %v", jobName, branch, deployType, err))
		return
	}

	sendFeishuMessage(ctx, receiveID, receiveIDType, fmt.Sprintf("⏳ 正在排队: %s\nBranch: %s\nType: %s\nQueueID: %d", jobName, branch, deployType, queueID))

	buildNum, err := client.WaitForBuildToStart(ctx, queueID)
	if err != nil {
		sendFeishuMessage(ctx, receiveID, receiveIDType, fmt.Sprintf("❌ 等待构建开始超时: %s\nQueueID: %d\nError: %v", jobName, queueID, err))
		return
	}

	sendFeishuMessage(ctx, receiveID, receiveIDType, fmt.Sprintf("🚀 构建已开始: %s #%d\nBranch: %s\nType: %s", jobName, buildNum, branch, deployType))

	build, err := client.MonitorBuildUntilCompletion(ctx, jobName, buildNum)
	if err != nil {
		sendFeishuMessage(ctx, receiveID, receiveIDType, fmt.Sprintf("❌ 监控构建出错: %s #%d\nError: %v", jobName, buildNum, err))
		return
	}

	result := build.GetResult()
	duration := build.Raw.Duration / 1000

	if result == "SUCCESS" {
		sendFeishuMessage(ctx, receiveID, receiveIDType, fmt.Sprintf("✅ 构建成功: %s #%d\nBranch: %s\nType: %s\nDuration: %ds", jobName, buildNum, branch, deployType, int64(duration)))
	} else {
		sendFeishuMessage(ctx, receiveID, receiveIDType, fmt.Sprintf("❌ 构建失败: %s #%d\nBranch: %s\nType: %s\nResult: %s", jobName, buildNum, branch, deployType, result))
	}
}

func sendFeishuMessage(ctx context.Context, receiveID, receiveIDType, content string) {
	if GlobalClient == nil {
		fmt.Println("GlobalClient is nil, cannot send message:", content)
		return
	}
	msgContent := map[string]any{
		"text": content,
	}
	msgBytes, _ := json.Marshal(msgContent)

	err := GlobalClient.SendMessage(ctx, receiveID, receiveIDType, "text", string(msgBytes))
	if err != nil {
		fmt.Printf("Failed to send Feishu message: %v\n", err)
	}
}

// handleApprovalChainCallback 处理审批链卡片回调
func handleApprovalChainCallback(ctx context.Context, event *callback.CardActionTriggerEvent, valueMap map[string]any) (*callback.CardActionTriggerResponse, error) {
	actionName, _ := valueMap["action"].(string)

	nodeInstanceIDFloat, ok := valueMap["node_instance_id"].(float64)
	if !ok {
		return toast("无效的审批节点ID"), nil
	}
	nodeInstanceID := uint(nodeInstanceIDFloat)

	operator := event.Event.Operator
	if operator == nil {
		return toast("无法获取操作人信息"), nil
	}

	userID := uint(0)
	userName := ""
	if operator.OpenID != "" {
		userName = operator.OpenID
	}

	if GlobalApprovalHandler == nil {
		return toast("审批服务未初始化"), nil
	}

	var err error
	switch actionName {
	case "approve":
		err = GlobalApprovalHandler.HandleApprove(ctx, nodeInstanceID, userID, userName, "")
		if err != nil {
			return toast(fmt.Sprintf("审批失败: %v", err)), nil
		}
		return &callback.CardActionTriggerResponse{
			Toast: &callback.Toast{
				Type:    "success",
				Content: "✅ 审批通过",
			},
		}, nil

	case "reject":
		err = GlobalApprovalHandler.HandleReject(ctx, nodeInstanceID, userID, userName, "")
		if err != nil {
			return toast(fmt.Sprintf("拒绝失败: %v", err)), nil
		}
		return &callback.CardActionTriggerResponse{
			Toast: &callback.Toast{
				Type:    "success",
				Content: "❌ 已拒绝",
			},
		}, nil

	default:
		return toast("未知的审批操作"), nil
	}
}
