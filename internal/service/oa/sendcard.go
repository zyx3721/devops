package oa

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"devops/internal/config"
	"devops/internal/domain/notification/service/feishu"
)

// SendCard 发送卡片
func SendCard(ctx context.Context, receiveID, receiveIDType string, jobs []*JenkinsJob) error {
	var req feishu.SendGrayCardRequest
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	client := feishu.NewClient(cfg)
	sender := feishu.NewAPISender(client)

	requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())

	req.CardData.ReceiveID = receiveID
	req.CardData.ReceiveIDType = receiveIDType

	services := make([]feishu.Service, 0, len(jobs))
	for _, job := range jobs {
		services = append(services, feishu.Service{
			Name:     job.JobName,
			ObjectID: job.JobName,
			Branches: []string{job.JobBranch},
			Actions:  []string{"check", "gray", "official"},
		})
	}
	req.CardData.Services = services

	feishu.GlobalStore.Save(requestID, req.CardData)

	displayCardData := req.CardData
	hasGray := false
	for _, s := range req.CardData.Services {
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
		var filteredServices []feishu.Service
		for _, s := range req.CardData.Services {
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
		displayCardData.Services = filteredServices
	}

	cardContent := feishu.BuildCard(displayCardData, requestID, nil, nil)
	cardBytes, err := json.Marshal(cardContent)
	if err != nil {
		return err
	}

	err = sender.Send(ctx, receiveID, receiveIDType, "interactive", string(cardBytes))
	if err != nil {
		return err
	}
	return nil
}
