package config

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// UpstreamConfig 对应 config.yaml 中 api-keys 相关的最小骨架。
type rawConfigYAML struct {
	APIKeys struct {
		DeepSeek []struct {
			BaseURL string `yaml:"base-url"`
			Keys    []struct {
				APIKey string `yaml:"api-key"`
			} `yaml:"keys"`
		} `yaml:"deepseek"`
		Codex []struct {
			BaseURL string `yaml:"base-url"`
			Keys    []struct {
				APIKey string `yaml:"api-key"`
			} `yaml:"keys"`
		} `yaml:"codex"`
		OpenAICompat []struct {
			Name    string `yaml:"name"`
			BaseURL string `yaml:"base-url"`
			Keys    []struct {
				APIKey string `yaml:"api-key"`
			} `yaml:"keys"`
		} `yaml:"openai-compatibility"`
	} `yaml:"api-keys"`
}

// DiscoverKeysFromYAML 解析 config.yaml 内容，自动提取 GLM 和 DeepSeek 的 API Key。
func DiscoverKeysFromYAML(data []byte) (glmKey string, deepseekKey string) {
	var raw rawConfigYAML
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return "", ""
	}

	// 1. 寻找 DeepSeek Key
	// 先在 api-keys.deepseek 找
	for _, group := range raw.APIKeys.DeepSeek {
		for _, k := range group.Keys {
			if strings.TrimSpace(k.APIKey) != "" {
				deepseekKey = strings.TrimSpace(k.APIKey)
				break
			}
		}
		if deepseekKey != "" {
			break
		}
	}
	// 如果没有，再在 codex 找 base-url 包含 deepseek.com 的
	if deepseekKey == "" {
		for _, group := range raw.APIKeys.Codex {
			if strings.Contains(strings.ToLower(group.BaseURL), "deepseek") {
				for _, k := range group.Keys {
					if strings.TrimSpace(k.APIKey) != "" {
						deepseekKey = strings.TrimSpace(k.APIKey)
						break
					}
				}
			}
			if deepseekKey != "" {
				break
			}
		}
	}

	// 2. 寻找 GLM / 智谱 Key
	for _, group := range raw.APIKeys.OpenAICompat {
		lowerURL := strings.ToLower(group.BaseURL)
		lowerName := strings.ToLower(group.Name)
		if strings.Contains(lowerURL, "bigmodel") || strings.Contains(lowerURL, "z.ai") ||
			strings.Contains(lowerName, "glm") || strings.Contains(lowerName, "zhipu") {
			for _, k := range group.Keys {
				if strings.TrimSpace(k.APIKey) != "" {
					glmKey = strings.TrimSpace(k.APIKey)
					break
				}
			}
		}
		if glmKey != "" {
			break
		}
	}

	return glmKey, deepseekKey
}

// AutoDiscoverAPIKeys 尝试在常见位置寻找宿主的 config.yaml 并提取未配置的 Key。
func (c *Config) AutoDiscoverAPIKeys() {
	if c.GLMAPIKey != "" && c.DeepSeekAPIKey != "" {
		return
	}

	// 尝试可能的路径
	candidates := []string{
		"config.yaml",
		"../config.yaml",
		"../../config.yaml",
	}

	// 如果能获取当前执行文件或工作目录
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(dir, "config.yaml"),
			filepath.Join(dir, "..", "config.yaml"),
		)
	}

	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err == nil && len(data) > 0 {
			glm, ds := DiscoverKeysFromYAML(data)
			if c.GLMAPIKey == "" && glm != "" {
				c.GLMAPIKey = glm
			}
			if c.DeepSeekAPIKey == "" && ds != "" {
				c.DeepSeekAPIKey = ds
			}
			if c.GLMAPIKey != "" && c.DeepSeekAPIKey != "" {
				break
			}
		}
	}
}
