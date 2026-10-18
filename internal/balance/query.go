package balance

import (
	"encoding/json"
	"fmt"
	"time"
)

// HostClient 通过宿主回调执行 HTTP 请求的抽象。
// 生产实现走 C ABI 的 host.http.do；测试里可以注入假实现。
type HostClient interface {
	// DoHTTP 通过宿主执行一次 HTTP GET，返回响应体字节。
	// 上游返回非 2xx 时返回包含状态码的错误。
	DoHTTP(url string, headers map[string][]string, timeout time.Duration) ([]byte, error)
}

// GLMAuthorizationHeader 返回智谱用量接口的认证头值。
// 注意：cc-switch 社区实测该接口使用裸 token，不带 Bearer 前缀。
func GLMAuthorizationHeader(apiKey string) string {
	return apiKey
}

// QueryGLM 查询智谱 Coding Plan 用量并解析。
func QueryGLM(client HostClient, endpoint, apiKey string, timeout time.Duration) ([]Entry, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("未配置 glm_api_key")
	}
	body, errDo := client.DoHTTP(endpoint, map[string][]string{
		"Authorization": {GLMAuthorizationHeader(apiKey)},
		"Accept":        {"application/json"},
	}, timeout)
	if errDo != nil {
		return nil, errDo
	}
	return ParseGLM(body)
}

// QueryDeepSeek 查询 DeepSeek 账户余额并解析。
func QueryDeepSeek(client HostClient, endpoint, apiKey string, timeout time.Duration) ([]Entry, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("未配置 deepseek_api_key")
	}
	body, errDo := client.DoHTTP(endpoint, map[string][]string{
		"Authorization": {"Bearer " + apiKey},
		"Accept":        {"application/json"},
	}, timeout)
	if errDo != nil {
		return nil, errDo
	}
	return ParseDeepSeek(body)
}

// MarshalResult 把 Report 序列化为 management.handle 返回的 JSON 字节。
func MarshalResult(report Report) ([]byte, error) {
	return json.Marshal(report)
}
