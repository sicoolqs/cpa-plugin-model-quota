package config

import "testing"

func TestParseEmpty(t *testing.T) {
	parsed, errParse := Parse(nil)
	if errParse != nil {
		t.Fatalf("空配置不应报错: %v", errParse)
	}
	if parsed.GLMEndpoint() == "" || parsed.DeepSeekEndpoint() == "" {
		t.Fatal("端点不应为空")
	}
	if parsed.Timeout() != DefaultTimeoutSeconds {
		t.Errorf("默认超时应为 %d，得到 %d", DefaultTimeoutSeconds, parsed.Timeout())
	}
}

func TestParseFields(t *testing.T) {
	parsed, errParse := Parse([]byte(
		"glm_api_key: key-1\nglm_region: intl\ndeepseek_api_key: key-2\ntimeout_seconds: 30\n"))
	if errParse != nil {
		t.Fatalf("解析失败: %v", errParse)
	}
	if parsed.GLMAPIKey != "key-1" || parsed.DeepSeekAPIKey != "key-2" {
		t.Errorf("key 解析不符: %+v", parsed)
	}
	if parsed.OpenCodeEndpoint() != "https://opencode.ai/zen/go/v1/usage" {
		t.Errorf("OpenCode 端点不符: %s", parsed.OpenCodeEndpoint())
	}
	if parsed.GLMEndpoint() != "https://api.z.ai/api/monitor/usage/quota/limit" {
		t.Errorf("国际版端点不符: %s", parsed.GLMEndpoint())
	}
	if parsed.Timeout() != 30 {
		t.Errorf("超时应为 30，得到 %d", parsed.Timeout())
	}
}

func TestParseDefaultRegion(t *testing.T) {
	parsed, errParse := Parse([]byte("glm_api_key: k\n"))
	if errParse != nil {
		t.Fatalf("解析失败: %v", errParse)
	}
	if parsed.GLMEndpoint() != "https://open.bigmodel.cn/api/monitor/usage/quota/limit" {
		t.Errorf("默认应为国内端点: %s", parsed.GLMEndpoint())
	}
}
