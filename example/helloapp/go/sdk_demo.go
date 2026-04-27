package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	swmsdk "software-web-manager/sdk-go"
)

func runFullSDKDemo() {
	fmt.Println("\n[SDK Demo] 开始调用 sdk/go 全量能力（含管理接口）...")

	client, err := newConfiguredSWMClient()
	if err != nil {
		fmt.Printf("[SDK Demo] 跳过：%v\n", err)
		return
	}
	ctx := context.Background()

	runStep("Client.ReportHeartbeat", func() error {
		return client.ReportHeartbeat(Version)
	})

	runStep("Client.ReportEvent", func() error {
		return client.ReportEvent("sdk_demo_single_event", map[string]interface{}{
			"version":      Version,
			"version_code": VersionCode,
			"at":           time.Now().Format(time.RFC3339),
		})
	})

	runStep("Client.ReportEvents", func() error {
		events := []swmsdk.Event{
			{
				DeviceID:    client.DeviceID,
				EventName:   "check_update",
				EventTime:   time.Now(),
				ChannelCode: client.Channel,
				Properties:  map[string]interface{}{"demo": true, "seq": 1, "version": Version},
			},
			{
				DeviceID:    client.DeviceID,
				EventName:   "sdk_demo_batch_event",
				EventTime:   time.Now(),
				ChannelCode: client.Channel,
				Properties:  map[string]interface{}{"demo": true, "seq": 2, "version": Version},
			},
		}
		return client.ReportEvents(events)
	})

	runStep("Client.ReportFeedback", func() error {
		tmp, err := os.CreateTemp("", "helloapp-feedback-*.txt")
		if err != nil {
			return err
		}
		defer os.Remove(tmp.Name())
		_, _ = tmp.WriteString("helloapp sdk feedback attachment")
		_ = tmp.Close()

		rating := 5
		return client.ReportFeedback(
			"helloapp 自动化 SDK 演示反馈",
			&rating,
			"helloapp@example.local",
			[]string{tmp.Name()},
			map[string]interface{}{
				"app_version": Version,
				"demo_mode":   "full",
			},
		)
	})

	runManagementSDKDemo(ctx, client)
}

func runManagementSDKDemo(ctx context.Context, c *SWMClient) {
	fmt.Println("[SDK Demo] 管理接口调用开始...")

	authToken := strings.TrimSpace(os.Getenv("SWM_AUTH_TOKEN"))
	if authToken == "" {
		fmt.Println("[SDK Demo] 未设置 SWM_AUTH_TOKEN，管理接口将跳过。")
		return
	}

	allowWrite := parseBoolEnv("SWM_DEMO_ALLOW_WRITE", false)
	appID := strings.TrimSpace(os.Getenv("SWM_APP_ID"))
	if appID == "" {
		appID = strings.TrimSpace(SWMAppID)
	}
	if appID == "" || appID == defaultSWMAppID {
		fmt.Println("[SDK Demo] 未设置有效 SWM_APP_ID，将使用占位 ID 发起只读/失败演示请求。")
		appID = "00000000-0000-0000-0000-000000000000"
	}
	dummyID := "00000000-0000-0000-0000-000000000001"
	userID := strings.TrimSpace(os.Getenv("SWM_TEST_USER_ID"))
	if userID == "" {
		userID = dummyID
	}

	sdk := c.SDK()
	sdk.SetAuthToken(authToken)

	var createdReleaseID string
	var createdTemplateID string
	var createdAppSecretID string
	var createdReleaseChannelID string
	var createdArtifactID string

	runStep("Mgmt.GetApp", func() error {
		_, err := sdk.GetApp(ctx, appID)
		return err
	})
	runStep("Mgmt.UpdateApp", func() error {
		target := appID
		if !allowWrite {
			target = dummyID
		}
		_, err := sdk.UpdateApp(ctx, target, map[string]interface{}{
			"description": "helloapp sdk full demo update",
		})
		return err
	})
	runStep("Mgmt.ListChannels", func() error {
		_, err := sdk.ListChannels(ctx, appID)
		return err
	})
	runStep("Mgmt.CreateChannel", func() error {
		target := appID
		if !allowWrite {
			target = dummyID
		}
		_, err := sdk.CreateChannel(ctx, target, map[string]interface{}{
			"name":         "SDK Demo Channel",
			"code":         fmt.Sprintf("sdk-demo-%d", time.Now().Unix()),
			"rollout_mode": "manual",
			"is_default":   false,
		})
		return err
	})
	runStep("Mgmt.ListAppMembers", func() error {
		_, err := sdk.ListAppMembers(ctx, appID)
		return err
	})
	runStep("Mgmt.AddAppMember", func() error {
		target := appID
		targetUser := userID
		if !allowWrite {
			target = dummyID
			targetUser = dummyID
		}
		_, err := sdk.AddAppMember(ctx, target, map[string]interface{}{
			"user_id":   targetUser,
			"role_name": "viewer",
		})
		return err
	})

	runStep("Mgmt.ListReleases", func() error {
		_, err := sdk.ListReleases(ctx, appID)
		return err
	})
	runStep("Mgmt.CreateRelease", func() error {
		target := appID
		if !allowWrite {
			target = dummyID
		}
		release, err := sdk.CreateRelease(ctx, target, map[string]interface{}{
			"version":      fmt.Sprintf("sdk-demo-%d", time.Now().Unix()),
			"version_code": int(time.Now().Unix() % 1000000),
			"notes":        "helloapp sdk full demo",
		})
		if err == nil {
			createdReleaseID = mapID(release)
		}
		return err
	})
	runStep("Mgmt.UpdateRelease", func() error {
		target := pickID(createdReleaseID, os.Getenv("SWM_RELEASE_ID"), dummyID)
		_, err := sdk.UpdateRelease(ctx, target, map[string]interface{}{
			"notes": "helloapp sdk demo updated notes",
		})
		return err
	})

	runStep("Mgmt.SubmitRelease", func() error {
		target := pickID(createdReleaseID, os.Getenv("SWM_RELEASE_ID"), dummyID)
		return sdk.SubmitRelease(ctx, target, "sdk demo submit")
	})
	runStep("Mgmt.ApproveRelease", func() error {
		target := pickID(createdReleaseID, os.Getenv("SWM_RELEASE_ID"), dummyID)
		return sdk.ApproveRelease(ctx, target, "sdk demo approve")
	})
	runStep("Mgmt.RejectRelease", func() error {
		target := pickID(createdReleaseID, os.Getenv("SWM_RELEASE_ID"), dummyID)
		return sdk.RejectRelease(ctx, target, "sdk demo reject")
	})
	runStep("Mgmt.PublishRelease", func() error {
		target := pickID(createdReleaseID, os.Getenv("SWM_RELEASE_ID"), dummyID)
		_, err := sdk.PublishRelease(ctx, target, map[string]interface{}{
			"channel_code": "stable",
			"rollout":      100,
		})
		return err
	})
	runStep("Mgmt.RollbackRelease", func() error {
		target := pickID(createdReleaseID, os.Getenv("SWM_RELEASE_ID"), dummyID)
		_, err := sdk.RollbackRelease(ctx, target, map[string]interface{}{
			"reason": "sdk demo rollback",
		})
		return err
	})
	runStep("Mgmt.RevokeRelease", func() error {
		target := pickID(createdReleaseID, os.Getenv("SWM_RELEASE_ID"), dummyID)
		return sdk.RevokeRelease(ctx, target)
	})

	runStep("Mgmt.UploadArtifact", func() error {
		target := pickID(createdReleaseID, os.Getenv("SWM_RELEASE_ID"), dummyID)
		tmp, err := os.CreateTemp("", "helloapp-artifact-*.bin")
		if err != nil {
			return err
		}
		defer os.Remove(tmp.Name())
		_, _ = tmp.WriteString("helloapp sdk demo artifact")
		_ = tmp.Close()

		artifact, err := sdk.UploadArtifact(ctx, target, swmsdk.UploadArtifactOptions{
			Platform: "universal",
			Arch:     "universal",
			FileType: "universal",
			FilePath: tmp.Name(),
			Replace:  true,
		})
		if err == nil {
			createdArtifactID = mapID(artifact)
		}
		return err
	})
	runStep("Mgmt.ListArtifacts", func() error {
		target := pickID(createdReleaseID, os.Getenv("SWM_RELEASE_ID"), dummyID)
		_, err := sdk.ListArtifacts(ctx, target)
		return err
	})
	runStep("Mgmt.GetArtifactDownloadURL", func() error {
		target := pickID(createdArtifactID, os.Getenv("SWM_ARTIFACT_ID"), dummyID)
		_, err := sdk.GetArtifactDownloadURL(ctx, target)
		return err
	})

	runStep("Mgmt.ListReleaseChannels", func() error {
		_, err := sdk.ListReleaseChannels(ctx, appID)
		return err
	})
	runStep("Mgmt.CreateReleaseChannel", func() error {
		target := appID
		if !allowWrite {
			target = dummyID
		}
		ch, err := sdk.CreateReleaseChannel(ctx, target, map[string]interface{}{
			"channel_code": "stable",
			"release_id":   pickID(createdReleaseID, os.Getenv("SWM_RELEASE_ID"), dummyID),
			"rollout":      100,
		})
		if err == nil {
			createdReleaseChannelID = mapID(ch)
		}
		return err
	})
	runStep("Mgmt.UpdateReleaseChannel", func() error {
		target := pickID(createdReleaseChannelID, os.Getenv("SWM_RELEASE_CHANNEL_ID"), dummyID)
		_, err := sdk.UpdateReleaseChannel(ctx, target, map[string]interface{}{
			"rollout": 50,
		})
		return err
	})
	runStep("Mgmt.GetReleaseChannelMetrics", func() error {
		target := pickID(createdReleaseChannelID, os.Getenv("SWM_RELEASE_CHANNEL_ID"), dummyID)
		_, err := sdk.GetReleaseChannelMetrics(ctx, appID, target, "", "")
		return err
	})

	runStep("Mgmt.GetAppRegionRules", func() error {
		_, err := sdk.GetAppRegionRules(ctx, appID)
		return err
	})
	runStep("Mgmt.UpdateAppRegionRules", func() error {
		target := appID
		if !allowWrite {
			target = dummyID
		}
		_, err := sdk.UpdateAppRegionRules(ctx, target, map[string]interface{}{
			"include": []string{"CN", "US"},
		})
		return err
	})
	runStep("Mgmt.ListGeoRegions", func() error {
		_, err := sdk.ListGeoRegions(ctx)
		return err
	})
	runStep("Mgmt.ResolveGeo", func() error {
		_, err := sdk.ResolveGeo(ctx, "8.8.8.8")
		return err
	})

	runStep("Mgmt.ListReleaseTemplates", func() error {
		_, err := sdk.ListReleaseTemplates(ctx)
		return err
	})
	runStep("Mgmt.CreateReleaseTemplate", func() error {
		targetRelease := pickID(createdReleaseID, os.Getenv("SWM_RELEASE_ID"), dummyID)
		if !allowWrite {
			targetRelease = dummyID
		}
		tpl, err := sdk.CreateReleaseTemplate(ctx, map[string]interface{}{
			"name":       "SDK Demo Template",
			"description": "helloapp sdk demo template",
			"release_id": targetRelease,
		})
		if err == nil {
			createdTemplateID = mapID(tpl)
		}
		return err
	})
	runStep("Mgmt.UpdateReleaseTemplate", func() error {
		target := pickID(createdTemplateID, os.Getenv("SWM_TEMPLATE_ID"), dummyID)
		_, err := sdk.UpdateReleaseTemplate(ctx, target, map[string]interface{}{
			"name": "SDK Demo Template Updated",
		})
		return err
	})
	runStep("Mgmt.SetReleaseTemplate", func() error {
		targetRelease := pickID(createdReleaseID, os.Getenv("SWM_RELEASE_ID"), dummyID)
		targetTpl := pickID(createdTemplateID, os.Getenv("SWM_TEMPLATE_ID"), dummyID)
		_, err := sdk.SetReleaseTemplate(ctx, targetRelease, targetTpl)
		return err
	})
	runStep("Mgmt.DeleteReleaseTemplate", func() error {
		target := pickID(createdTemplateID, os.Getenv("SWM_TEMPLATE_ID"), dummyID)
		if !allowWrite {
			target = dummyID
		}
		return sdk.DeleteReleaseTemplate(ctx, target)
	})

	runStep("Mgmt.ListAppSecrets", func() error {
		_, err := sdk.ListAppSecrets(ctx, appID)
		return err
	})
	runStep("Mgmt.CreateAppSecret", func() error {
		target := appID
		if !allowWrite {
			target = dummyID
		}
		resp, err := sdk.CreateAppSecret(ctx, target, map[string]interface{}{
			"name":   fmt.Sprintf("sdk-demo-key-%d", time.Now().Unix()),
			"scopes": []string{"update:check", "event:write"},
		})
		if err == nil && resp != nil {
			createdAppSecretID = mapID(resp.Item)
			secret := strings.TrimSpace(resp.AppSecret)
			if secret != "" {
				fmt.Printf("[SDK Demo] Mgmt.CreateAppSecret 返回密钥(脱敏): %s\n", maskSecret(secret))
			}
		}
		return err
	})
	runStep("Mgmt.RevokeAppSecret", func() error {
		target := pickID(createdAppSecretID, os.Getenv("SWM_APP_SECRET_ID"), dummyID)
		if !allowWrite {
			target = dummyID
		}
		return sdk.RevokeAppSecret(ctx, target)
	})

	runStep("Mgmt.DeleteRelease", func() error {
		target := pickID(createdReleaseID, os.Getenv("SWM_RELEASE_ID"), dummyID)
		if !allowWrite {
			target = dummyID
		}
		return sdk.DeleteRelease(ctx, target)
	})
}

func runStep(name string, fn func() error) {
	start := time.Now()
	err := fn()
	if err != nil {
		fmt.Printf("[SDK Demo] %s: FAILED (%s) -> %v\n", name, time.Since(start).Round(time.Millisecond), err)
		return
	}
	fmt.Printf("[SDK Demo] %s: OK (%s)\n", name, time.Since(start).Round(time.Millisecond))
}

func pickID(primary, secondary, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return strings.TrimSpace(primary)
	}
	if strings.TrimSpace(secondary) != "" {
		return strings.TrimSpace(secondary)
	}
	return fallback
}

func mapID(item map[string]interface{}) string {
	if item == nil {
		return ""
	}
	if v, ok := item["id"]; ok {
		switch x := v.(type) {
		case string:
			return x
		case fmt.Stringer:
			return x.String()
		default:
			return fmt.Sprintf("%v", x)
		}
	}
	return ""
}

func parseBoolEnv(key string, def bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return def
	}
	return v
}

func maskSecret(v string) string {
	v = strings.TrimSpace(v)
	if len(v) <= 12 {
		return "***"
	}
	return v[:6] + "..." + v[len(v)-4:]
}
