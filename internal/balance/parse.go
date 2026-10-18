package balance

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ParseGLM 把智谱用量接口的响应体解析为归一化条目。
// limits 里 TOKENS_LIMIT 按窗口顺序对应 5 小时与每周额度，TIME_LIMIT 是 MCP 月度调用。
func ParseGLM(body []byte) ([]Entry, error) {
	var parsed GLMUsageResponse
	if errUnmarshal := json.Unmarshal(body, &parsed); errUnmarshal != nil {
		return nil, fmt.Errorf("解析智谱响应失败: %w", errUnmarshal)
	}
	if parsed.Code != 200 && parsed.Code != 0 {
		return nil, fmt.Errorf("智谱接口返回错误 %d: %s", parsed.Code, parsed.Msg)
	}
	// 上游数组顺序不保证：TOKENS_LIMIT 按窗口先后对应 5 小时/每周，TIME_LIMIT 是 MCP 月度调用。
	// 分别收集后固定按 5小时 → 每周 → MCP 输出。
	var tokenLimits []GLMLimit
	var timeLimits []GLMLimit
	for _, limit := range parsed.Data.Limits {
		switch limit.Type {
		case "TOKENS_LIMIT":
			tokenLimits = append(tokenLimits, limit)
		case "TIME_LIMIT":
			timeLimits = append(timeLimits, limit)
		}
	}
	if len(tokenLimits) == 0 && len(timeLimits) == 0 {
		return nil, fmt.Errorf("智谱响应中没有额度数据，可能不是 Coding Plan 套餐 key")
	}
	entries := make([]Entry, 0, len(tokenLimits)+len(timeLimits))
	tokenLabels := []string{"5小时额度", "每周额度"}
	for i, limit := range tokenLimits {
		label := tokenLabels[len(tokenLabels)-1]
		if i < len(tokenLabels) {
			label = tokenLabels[i]
		}
		used := limit.Percentage
		entries = append(entries, Entry{
			Provider:    "glm",
			Label:       label,
			UsedPercent: &used,
			Remaining:   percentRemaining(used),
			Detail:      planLevel(parsed.Data.Level),
			ResetAtMS:   normalizeResetMS(limit.NextResetTime),
		})
	}
	for _, limit := range timeLimits {
		entries = append(entries, Entry{
			Provider:  "glm",
			Label:     "MCP 每月调用",
			Remaining: fmt.Sprintf("%s / %s 次", trimFloat(limit.CurrentValue), trimFloat(limit.Usage)),
			Detail:    planLevel(parsed.Data.Level),
			ResetAtMS: normalizeResetMS(limit.NextResetTime),
		})
	}
	return entries, nil
}

// ParseDeepSeek 把 DeepSeek 余额接口的响应体解析为归一化条目。
func ParseDeepSeek(body []byte) ([]Entry, error) {
	var parsed DeepSeekBalanceResponse
	if errUnmarshal := json.Unmarshal(body, &parsed); errUnmarshal != nil {
		return nil, fmt.Errorf("解析 DeepSeek 响应失败: %w", errUnmarshal)
	}
	if len(parsed.BalanceInfos) == 0 {
		return nil, fmt.Errorf("DeepSeek 响应中没有余额数据")
	}
	entries := make([]Entry, 0, len(parsed.BalanceInfos))
	for _, info := range parsed.BalanceInfos {
		detail := "余额充足"
		if !parsed.IsAvailable {
			detail = "账户不可用"
		}
		entries = append(entries, Entry{
			Provider:  "deepseek",
			Label:     "账户余额",
			Remaining: strings.TrimSpace(info.TotalBalance),
			Detail:    currencyLabel(info.Currency) + " · " + detail,
		})
	}
	return entries, nil
}

func percentRemaining(used float64) string {
	remaining := 100 - used
	if remaining < 0 {
		remaining = 0
	}
	return fmt.Sprintf("%s%%", trimFloat(remaining))
}

// normalizeResetMS 把智谱的 nextResetTime 统一为 Unix 毫秒。
// 上游可能给秒或毫秒时间戳：低于 1e12 视为秒并换算；0 或异常值返回 0（未知）。
func normalizeResetMS(value int64) int64 {
	switch {
	case value <= 0:
		return 0
	case value < 1e12:
		return value * 1000
	default:
		return value
	}
}

func planLevel(level string) string {
	if level == "" {
		return ""
	}
	return "套餐: " + level
}

func currencyLabel(code string) string {
	switch strings.ToUpper(code) {
	case "CNY":
		return "人民币"
	case "USD":
		return "美元"
	default:
		return code
	}
}

// trimFloat 输出整数时去掉小数点，避免 "70.0%" 这类展示。
func trimFloat(value float64) string {
	if value == float64(int64(value)) {
		return fmt.Sprintf("%d", int64(value))
	}
	return fmt.Sprintf("%.1f", value)
}
