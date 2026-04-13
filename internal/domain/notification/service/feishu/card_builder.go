package feishu

import (
	"fmt"
	"strings"

	"devops/pkg/logger"
)

// BuildCard 构建灰度发布卡片
// 使用 V1 Message Card 格式以支持 action 模块的多组件布局
func BuildCard(req GrayCardRequest, requestID string, disabledActions map[string]bool, actionCounts map[string]int) map[string]any {
	log := logger.NewLogger("ERROR")

	if len(req.Services) == 0 {
		log.Error("Services list is empty")
		return nil
	}
	if req.Services[0].ObjectID == "" {
		log.Error("ObjectID is empty")
		return nil
	}
	if len(req.Services[0].Actions) == 0 {
		log.Error("Actions list is empty")
		return nil
	}
	if len(req.Services[0].Branches) == 0 {
		log.Error("Branches list is empty")
		return nil
	}

	req.ObjectID = req.Services[0].ObjectID
	req.Title = fmt.Sprintf("🚀%s-服务发布通知", req.ObjectID)

	elements := []any{
		map[string]any{
			"tag": "div",
			"text": map[string]any{
				"tag": "lark_md",
			},
		},
		map[string]any{
			"tag": "hr",
		},
		map[string]any{
			"tag": "div",
			"text": map[string]any{
				"tag":     "lark_md",
				"content": "📋 **服务列表与操作**",
			},
		},
	}

	for i, service := range req.Services {
		elements = append(elements, map[string]any{
			"tag": "div",
			"text": map[string]any{
				"tag":     "lark_md",
				"content": fmt.Sprintf("**%d. 服务名称：** `%s`", i+1, service.Name),
			},
		})

		var branchDisplay string
		if len(service.Branches) == 0 {
			branchDisplay = "无分支"
			log.Error("Service %s has no branches", service.Name)
		} else {
			branchDisplay = service.Branches[0]
		}

		elements = append(elements, map[string]any{
			"tag": "div",
			"text": map[string]any{
				"tag":     "lark_md",
				"content": fmt.Sprintf("📦 **发布分支：** `%s`", branchDisplay),
			},
		})

		actionsList := []any{}

		var currentActions []string
		hasRollback := false
		hasRestart := false

		for _, a := range service.Actions {
			if strings.EqualFold(a, "check") || strings.EqualFold(a, "验收") {
				continue
			}
			if strings.EqualFold(a, "rollback") || strings.EqualFold(a, "回滚") {
				hasRollback = true
			}
			if strings.EqualFold(a, "restart") || strings.EqualFold(a, "重启") {
				hasRestart = true
			}
			currentActions = append(currentActions, a)
		}

		if !hasRollback {
			currentActions = append(currentActions, "rollback")
		}
		if !hasRestart {
			currentActions = append(currentActions, "restart")
		}

		for _, action := range currentActions {
			var text string
			var valueAction string
			var btnType string = "primary"

			switch strings.ToLower(action) {
			case "gray", "灰度":
				text = "🚀 灰度"
				valueAction = "do_gray_release"
			case "official", "release", "正式":
				text = "🎉 正式"
				valueAction = "do_official_release"
				btnType = "danger"
			case "rollback", "回滚":
				text = "🔙 回滚"
				valueAction = "do_rollback"
				btnType = "danger"
			case "restart", "重启":
				text = "🔄 重启"
				valueAction = "do_restart"
				btnType = "primary"
			default:
				text = action
				valueAction = "do_" + action
			}

			isDisabled := false
			count := 0
			key := fmt.Sprintf("%s:%s", service.Name, valueAction)

			if disabledActions != nil {
				if disabledActions[key] {
					isDisabled = true
					btnType = "default"
				}
			}

			if actionCounts != nil {
				count = actionCounts[key]
			}

			if count > 0 {
				text = fmt.Sprintf("%s (%d)", text, count)
			}

			button := map[string]any{
				"tag": "button",
				"text": map[string]any{
					"tag":     "plain_text",
					"content": text,
				},
				"type":     btnType,
				"disabled": isDisabled,
				"value": map[string]any{
					"action":     valueAction,
					"service":    service.Name,
					"request_id": requestID,
					"branch":     branchDisplay,
				},
				"confirm": map[string]any{
					"title": map[string]any{
						"tag":     "plain_text",
						"content": "是否确认？",
					},
					"ok_text": map[string]any{
						"tag":     "plain_text",
						"content": "确认",
					},
					"cancel_text": map[string]any{
						"tag":     "plain_text",
						"content": "取消",
					},
				},
			}
			actionsList = append(actionsList, button)
		}

		actionElement := map[string]any{
			"tag":     "action",
			"actions": actionsList,
		}
		elements = append(elements, actionElement)

		if i < len(req.Services)-1 {
			elements = append(elements, map[string]any{
				"tag": "hr",
			})
		}
	}

	// 批量操作按钮
	elements = append(elements, map[string]any{
		"tag": "hr",
	})
	elements = append(elements, map[string]any{
		"tag": "div",
		"text": map[string]any{
			"tag":     "lark_md",
			"content": "⚡ **批量操作**",
		},
	})

	allBranches := make(map[string]string)
	for _, svc := range req.Services {
		if len(svc.Branches) > 0 {
			allBranches[svc.Name] = svc.Branches[0]
		}
	}

	batchActions := []any{}

	batchBtns := []struct {
		Text   string
		Type   string
		Action string
	}{
		{Text: "🚀 批量发布", Type: "primary", Action: "batch_release_all"},
		{Text: "⏹️ 结束批量发布", Type: "danger", Action: "stop_batch_release"},
	}

	for _, btn := range batchBtns {
		text := btn.Text
		btnType := btn.Type
		isDisabled := false

		if disabledActions != nil {
			key := fmt.Sprintf("BATCH:%s", btn.Action)
			if disabledActions[key] {
				isDisabled = true
				text = text + " (已执行)"
				btnType = "default"
			}
		}

		batchActions = append(batchActions, map[string]any{
			"tag": "button",
			"text": map[string]any{
				"tag":     "plain_text",
				"content": text,
			},
			"type":     btnType,
			"disabled": isDisabled,
			"value": map[string]any{
				"action":       btn.Action,
				"service":      "BATCH",
				"request_id":   requestID,
				"all_branches": allBranches,
			},
			"confirm": map[string]any{
				"title": map[string]any{
					"tag":     "plain_text",
					"content": fmt.Sprintf("是否确认%s所有服务？", strings.TrimPrefix(strings.TrimPrefix(btn.Text, "🚀 "), "⏹️ ")),
				},
				"ok_text": map[string]any{
					"tag":     "plain_text",
					"content": "确认",
				},
				"cancel_text": map[string]any{
					"tag":     "plain_text",
					"content": "取消",
				},
			},
		})
	}

	elements = append(elements, map[string]any{
		"tag":     "action",
		"actions": batchActions,
	})

	return map[string]any{
		"header": map[string]any{
			"title": map[string]any{
				"content": req.Title,
				"tag":     "plain_text",
			},
			"template": "blue",
		},
		"elements": elements,
	}
}
