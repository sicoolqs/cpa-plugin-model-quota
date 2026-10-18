package main

// hostBridge 通过 C ABI 宿主回调执行 HTTP 请求（host.http.do）。
// 真实请求由宿主执行，代理、传输策略与请求日志沿用内核设置。

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ZRZ-0805/cpa-plugin-model-quota/internal/balance"
)

// hostHTTPRequest 是 host.http.do 的请求体。Headers 值为数组以匹配 http.Header。
type hostHTTPRequest struct {
	Method  string              `json:"method"`
	URL     string              `json:"url"`
	Headers map[string][]string `json:"headers,omitempty"`
	Timeout int                 `json:"timeout_ms,omitempty"`
}

// hostHTTPResponse 是 host.http.do 的返回体。Body 为原始字节，经 JSON base64。
type hostHTTPResponse struct {
	StatusCode int                 `json:"StatusCode"`
	Headers    map[string][]string `json:"Headers,omitempty"`
	Body       []byte              `json:"Body,omitempty"`
}

// hostHTTPDo 调用宿主执行一次 HTTP 请求并解析信封与响应体。
// 返回的 body 已解码；非 2xx 状态返回携带状态码的错误，由上层归类。
func hostHTTPDo(callHost func(string, []byte) ([]byte, error), request hostHTTPRequest, timeout time.Duration) ([]byte, error) {
	if request.Timeout == 0 && timeout > 0 {
		request.Timeout = int(timeout / time.Millisecond)
	}
	requestBytes, errMarshal := json.Marshal(request)
	if errMarshal != nil {
		return nil, fmt.Errorf("序列化 host.http.do 请求失败: %w", errMarshal)
	}
	resultBytes, errCall := callHost("host.http.do", requestBytes)
	if errCall != nil {
		return nil, fmt.Errorf("宿主 HTTP 回调失败: %w", errCall)
	}
	var envelope struct {
		OK     bool            `json:"ok"`
		Result json.RawMessage `json:"result,omitempty"`
		Error  *struct {
			Code       string `json:"code"`
			Message    string `json:"message"`
			HTTPStatus int    `json:"http_status,omitempty"`
		} `json:"error,omitempty"`
	}
	if errUnmarshal := json.Unmarshal(resultBytes, &envelope); errUnmarshal != nil {
		return nil, fmt.Errorf("解析宿主回调信封失败: %w", errUnmarshal)
	}
	if !envelope.OK || envelope.Error != nil {
		message := "宿主 HTTP 回调出错"
		if envelope.Error != nil {
			message = fmt.Sprintf("%s: %s", envelope.Error.Code, envelope.Error.Message)
		}
		return nil, fmt.Errorf("%s", message)
	}
	var response hostHTTPResponse
	if errUnmarshal := json.Unmarshal(envelope.Result, &response); errUnmarshal != nil {
		return nil, fmt.Errorf("解析宿主 HTTP 响应失败: %w", errUnmarshal)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, &balance.UpstreamError{
			StatusCode: response.StatusCode,
			Message:    fmt.Sprintf("上游返回 HTTP %d: %s", response.StatusCode, truncate(string(response.Body), 200)),
		}
	}
	return response.Body, nil
}

// truncate 截断过长文本用于错误消息。
func truncate(text string, max int) string {
	runes := []rune(text)
	if len(runes) <= max {
		return text
	}
	return string(runes[:max]) + "…"
}

// compactJSON 去除 JSON 无关空白（调试与日志用）。
func compactJSON(raw []byte) string {
	var buffer bytes.Buffer
	if errCompact := json.Compact(&buffer, raw); errCompact != nil {
		return string(raw)
	}
	return buffer.String()
}
