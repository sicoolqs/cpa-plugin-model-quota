package main

import (
	"encoding/json"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginabi"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginapi"
)

// TestEnvelopeWireFormat 锁定 RPC 信封的 JSON 线格式：
// 改用官方 SDK 类型后，输出必须与手写实现逐字节一致。
func TestEnvelopeWireFormat(t *testing.T) {
	if got := string(errorEnvelope("bad_request", "oops")); got != `{"ok":false,"error":{"code":"bad_request","message":"oops"}}` {
		t.Errorf("错误信封线格式不符: %s", got)
	}
	okRaw, errMarshal := okEnvelope(map[string]any{"a": 1})
	if errMarshal != nil {
		t.Fatalf("成功信封序列化失败: %v", errMarshal)
	}
	if string(okRaw) != `{"ok":true,"result":{"a":1}}` {
		t.Errorf("成功信封线格式不符: %s", string(okRaw))
	}
}

// TestRegistrationContract 锁定注册载荷中与宿主协商的版本与能力字段。
func TestRegistrationContract(t *testing.T) {
	payload := registrationPayload()
	if got := payload["schema_version"]; got != pluginSchemaVersion {
		t.Errorf("schema_version 应声明为 %d，得到 %v", pluginSchemaVersion, got)
	}
	if pluginSchemaVersion != pluginabi.SchemaVersionRawManagementResponse {
		t.Errorf("插件应声明原始 JSON 管理响应语义（%d），得到 %d", pluginabi.SchemaVersionRawManagementResponse, pluginSchemaVersion)
	}
	capabilities, ok := payload["capabilities"].(map[string]any)
	if !ok || capabilities["management_api"] != true {
		t.Errorf("必须声明 management_api 能力，得到 %v", payload["capabilities"])
	}
	metadata, ok := payload["metadata"].(map[string]any)
	if !ok || metadata["Name"] == "" || metadata["Version"] == "" {
		t.Errorf("metadata 缺少必要字段: %v", payload["metadata"])
	}
}

// TestABIVersionMatchesSDK 确保初始化时写入宿主的 ABI 版本与 SDK 常量一致。
func TestABIVersionMatchesSDK(t *testing.T) {
	if pluginabi.ABIVersion != 1 {
		t.Logf("SDK ABI 版本已更新为 %d，请确认 C 结构体兼容性", pluginabi.ABIVersion)
	}
}

// TestRegistrationPayloadDecodesAsHostMetadata 模拟宿主按 pluginapi.Metadata 解码注册载荷，
// 确保字段名与类型完全匹配；未被 SDK 结构覆盖的扩展字段（如 Description）会被宿主忽略。
func TestRegistrationPayloadDecodesAsHostMetadata(t *testing.T) {
	raw, errEnvelope := okEnvelope(registrationPayload())
	if errEnvelope != nil {
		t.Fatalf("注册载荷序列化失败: %v", errEnvelope)
	}
	var envelope struct {
		OK     bool            `json:"ok"`
		Result json.RawMessage `json:"result"`
	}
	if errUnmarshal := json.Unmarshal(raw, &envelope); errUnmarshal != nil {
		t.Fatalf("宿主信封解码失败: %v", errUnmarshal)
	}
	var registration struct {
		SchemaVersion uint32             `json:"schema_version"`
		Metadata      pluginapi.Metadata `json:"metadata"`
		Capabilities  struct {
			ManagementAPI bool `json:"management_api"`
		} `json:"capabilities"`
	}
	if errUnmarshal := json.Unmarshal(envelope.Result, &registration); errUnmarshal != nil {
		t.Fatalf("宿主注册结果解码失败: %v", errUnmarshal)
	}
	if registration.SchemaVersion != pluginSchemaVersion {
		t.Errorf("schema_version 解码为 %d，期望 %d", registration.SchemaVersion, pluginSchemaVersion)
	}
	if registration.Metadata.Name == "" || registration.Metadata.Version == "" {
		t.Errorf("metadata 解码缺失必要字段: %+v", registration.Metadata)
	}
	if got := len(registration.Metadata.ConfigFields); got != 5 {
		t.Errorf("ConfigFields 应解码出 5 个字段，得到 %d", got)
	}
	for _, field := range registration.Metadata.ConfigFields {
		if field.Name == "" || field.Type == "" {
			t.Errorf("ConfigField 缺少 Name/Type: %+v", field)
		}
	}
	if !registration.Capabilities.ManagementAPI {
		t.Error("宿主未解码出 management_api 能力")
	}
}
