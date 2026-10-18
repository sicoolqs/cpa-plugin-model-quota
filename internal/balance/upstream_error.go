package balance

import "fmt"

// UpstreamError 携带上游 HTTP 状态码，供调用方映射到 JSON 信封的 http_status。
type UpstreamError struct {
	StatusCode int
	Message    string
}

func (e *UpstreamError) Error() string {
	return e.Message
}

// NewUpstreamError 构造一个上游状态码错误。
func NewUpstreamError(statusCode int, format string, args ...any) *UpstreamError {
	return &UpstreamError{StatusCode: statusCode, Message: fmt.Sprintf(format, args...)}
}
