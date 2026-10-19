// Package config 解析宿主传入的插件配置 YAML。
// 宿主把 plugins.configs.balance-monitor 下的字段标准化为 YAML 字节后传给
// plugin.register / plugin.reconfigure，本包只解析插件自有字段。
package config

import (
	"gopkg.in/yaml.v3"
)

// Config 是插件自有的配置字段。宿主保留的 enabled/priority 由宿主自行处理。
type Config struct {
	// GLMAPIKey 是智谱开放平台的 API Key（含 Coding Plan 套餐用户）。
	GLMAPIKey string `yaml:"glm_api_key"`
	// GLMRegion 选择智谱站点：cn 使用 open.bigmodel.cn，intl 使用 api.z.ai。
	GLMRegion string `yaml:"glm_region"`
	// DeepSeekAPIKey 是 DeepSeek 开放平台的 API Key。
	DeepSeekAPIKey string `yaml:"deepseek_api_key"`
	// OpenCodeAPIKey 是 OpenCode Zen Go 套餐的 API Key。
	OpenCodeAPIKey string `yaml:"opencode_api_key"`
	// TimeoutSeconds 是单次上游查询的超时秒数，0 表示使用默认值。
	TimeoutSeconds int `yaml:"timeout_seconds"`
}

// DefaultTimeoutSeconds 是未配置超时时的默认值。
const DefaultTimeoutSeconds = 15

// Parse 从宿主传入的标准化 YAML 字节解析插件配置。
// 解析失败时返回错误，由调用方决定上报方式；空输入视为空配置。
func Parse(data []byte) (Config, error) {
	parsed := Config{}
	if len(data) == 0 {
		return parsed, nil
	}
	if errUnmarshal := yaml.Unmarshal(data, &parsed); errUnmarshal != nil {
		return Config{}, errUnmarshal
	}
	return parsed, nil
}

// GLMEndpoint 根据区域返回智谱用量查询接口地址。未知区域按国内处理。
func (c Config) GLMEndpoint() string {
	if c.GLMRegion == "intl" {
		return "https://api.z.ai/api/monitor/usage/quota/limit"
	}
	return "https://open.bigmodel.cn/api/monitor/usage/quota/limit"
}

// DeepSeekEndpoint 返回 DeepSeek 余额查询接口地址。
func (c Config) DeepSeekEndpoint() string {
	return "https://api.deepseek.com/user/balance"
}

// OpenCodeEndpoint 返回 OpenCode Zen Go 用量查询接口地址。
func (c Config) OpenCodeEndpoint() string {
	return "https://opencode.ai/zen/go/v1/usage"
}

// Timeout 返回有效的查询超时秒数。
func (c Config) Timeout() int {
	if c.TimeoutSeconds > 0 {
		return c.TimeoutSeconds
	}
	return DefaultTimeoutSeconds
}
