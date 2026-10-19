// Package balance 定义统一的余额/额度数据结构与各家供应商的解析逻辑。
// 查询结果统一为 Report：每个供应商产生一组 Entry，页面按 Entry 渲染。
package balance

// Entry 是一条归一化的额度/余额记录。
type Entry struct {
	// Provider 是供应商标识：glm 或 deepseek。
	Provider string `json:"provider"`
	// Label 是展示名，如 "5小时额度"。
	Label string `json:"label"`
	// UsedPercent 是已用百分比（0-100）。仅比例型条目有值。
	UsedPercent *float64 `json:"used_percent,omitempty"`
	// Remaining 是剩余量的文本展示，如 "70%" 或 "¥120.50"。
	Remaining string `json:"remaining,omitempty"`
	// Detail 是补充说明，如套餐档位、币种。
	Detail string `json:"detail,omitempty"`
	// ResetAtMS 是额度窗口重置时间的 Unix 毫秒时间戳。0 表示未知。
	ResetAtMS int64 `json:"reset_at_ms,omitempty"`
}

// Report 是一次完整查询的聚合结果。
type Report struct {
	// Entries 是全部供应商的额度条目，按配置顺序排列。
	Entries []Entry `json:"entries"`
	// Errors 按供应商记录查询失败原因；查询失败的供应商不产生 Entries。
	Errors map[string]string `json:"errors,omitempty"`
	// FetchedAtMS 是查询完成的 Unix 毫秒时间戳。
	FetchedAtMS int64 `json:"fetched_at_ms"`
}

// GLMUsageResponse 对应智谱 /api/monitor/usage/quota/limit 的返回结构。
type GLMUsageResponse struct {
	Code int          `json:"code"`
	Msg  string       `json:"msg"`
	Data GLMUsageData `json:"data"`
}

// GLMUsageData 是智谱返回的套餐额度数据。
type GLMUsageData struct {
	Limits []GLMLimit `json:"limits"`
	Level  string     `json:"level"`
}

// GLMLimit 是一条智谱额度窗口。TOKENS_LIMIT 的两条分别对应 5 小时和每周窗口。
type GLMLimit struct {
	Type          string  `json:"type"`
	Percentage    float64 `json:"percentage"`
	Usage         float64 `json:"usage"`
	CurrentValue  float64 `json:"currentValue"`
	Remaining     float64 `json:"remaining"`
	NextResetTime int64   `json:"nextResetTime"`
}

// OpenCodeUsageResponse 对应 OpenCode Zen Go /zen/go/v1/usage 的返回结构。
type OpenCodeUsageResponse struct {
	Usage OpenCodeUsage `json:"usage"`
}

// OpenCodeUsage 是 OpenCode Go 套餐的三个额度窗口。
type OpenCodeUsage struct {
	Rolling OpenCodeWindow `json:"rolling"`
	Weekly  OpenCodeWindow `json:"weekly"`
	Monthly OpenCodeWindow `json:"monthly"`
}

// OpenCodeWindow 是一个滚动窗口的用量。ResetsAt 是 RFC3339 时间字符串。
type OpenCodeWindow struct {
	Status   string  `json:"status"`
	Percent  float64 `json:"percent"`
	ResetsAt string  `json:"resetsAt"`
}

// DeepSeekBalanceResponse 对应 DeepSeek /user/balance 的返回结构。
type DeepSeekBalanceResponse struct {
	IsAvailable  bool                     `json:"is_available"`
	BalanceInfos []DeepSeekBalanceInfo    `json:"balance_infos"`
}

// DeepSeekBalanceInfo 是一种币种的余额。
type DeepSeekBalanceInfo struct {
	Currency         string `json:"currency"`
	TotalBalance     string `json:"total_balance"`
	GrantedBalance   string `json:"granted_balance"`
	ToppedUpBalance  string `json:"topped_up_balance"`
}
