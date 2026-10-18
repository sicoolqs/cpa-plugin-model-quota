package config

import (
	"testing"
)

func TestDiscoverKeysFromYAML(t *testing.T) {
	yamlData := []byte(`
api-keys:
    codex:
        - base-url: "https://api.deepseek.com"
          keys:
            - api-key: "sk-ds-test-key-123"
          name: "deepseek-1"
    openai-compatibility:
        - base-url: "https://open.bigmodel.cn/api/coding/paas/v4"
          keys:
            - api-key: "glm-test-token-456"
          name: "GLM"
`)

	glmKey, deepseekKey := DiscoverKeysFromYAML(yamlData)
	if glmKey != "glm-test-token-456" {
		t.Errorf("提取 GLM Key 失败，得到: %q", glmKey)
	}
	if deepseekKey != "sk-ds-test-key-123" {
		t.Errorf("提取 DeepSeek Key 失败，得到: %q", deepseekKey)
	}
}
