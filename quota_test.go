package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ZRZ-0805/cpa-plugin-model-quota/internal/config"
)

// scriptedClient 按请求 URL 返回对应供应商的最小合法响应，模拟固定延迟的上游。
type scriptedClient struct {
	delay time.Duration
}

func (s scriptedClient) DoHTTP(url string, headers map[string][]string, timeout time.Duration) ([]byte, error) {
	time.Sleep(s.delay)
	switch {
	case strings.Contains(url, "bigmodel.cn"), strings.Contains(url, "api.z.ai"):
		return []byte(`{"code":200,"msg":"ok","data":{"limits":[{"type":"TOKENS_LIMIT","percentage":10}],"level":"lite"}}`), nil
	case strings.Contains(url, "deepseek.com"):
		return []byte(`{"is_available":true,"balance_infos":[{"currency":"CNY","total_balance":"1.00"}]}`), nil
	case strings.Contains(url, "opencode.ai"):
		return []byte(`{"usage":{"rolling":{"status":"ok","percent":1,"resetsAt":"2026-10-31T00:00:00.000Z"},"weekly":{"status":"ok","percent":2,"resetsAt":"2026-11-01T00:00:00.000Z"},"monthly":{"status":"ok","percent":3,"resetsAt":"2026-11-02T00:00:00.000Z"}}}`), nil
	default:
		return nil, fmt.Errorf("unexpected url %s", url)
	}
}

func TestRunQuotaQueriesParallelAndOrdered(t *testing.T) {
	cfg := config.Config{GLMAPIKey: "g", DeepSeekAPIKey: "d", OpenCodeAPIKey: "o"}
	start := time.Now()
	report := runQuotaQueries(scriptedClient{delay: 100 * time.Millisecond}, cfg, 5*time.Second)
	elapsed := time.Since(start)
	// 串行执行需 300ms；并行执行应接近单项延迟 100ms。
	if elapsed > 250*time.Millisecond {
		t.Fatalf("三家供应商应并行查询，实际耗时 %s", elapsed)
	}
	if len(report.Errors) != 0 {
		t.Fatalf("不应有查询错误: %v", report.Errors)
	}
	// glm 1 条 + deepseek 1 条 + opencode-go 3 条 = 5 条
	if len(report.Entries) != 5 {
		t.Fatalf("期望 5 条条目，得到 %d", len(report.Entries))
	}
	wantOrder := []string{"glm", "deepseek", "opencode-go", "opencode-go", "opencode-go"}
	for i, want := range wantOrder {
		if report.Entries[i].Provider != want {
			t.Errorf("条目 %d 应为 %s，得到 %s", i, want, report.Entries[i].Provider)
		}
	}
	if report.FetchedAtMS < start.Add(90*time.Millisecond).UnixMilli() {
		t.Errorf("FetchedAtMS 应在查询完成后设置，start=%d fetched=%d", start.UnixMilli(), report.FetchedAtMS)
	}
}

func TestRunQuotaQueriesConfigError(t *testing.T) {
	report := runQuotaQueries(scriptedClient{}, config.Config{}, time.Second)
	if _, ok := report.Errors["config"]; !ok {
		t.Fatal("未配置任何 API Key 应返回 config 错误")
	}
	if len(report.Entries) != 0 {
		t.Errorf("未配置时不应有条目，得到 %d", len(report.Entries))
	}
}

func TestRunQuotaQueriesPartialFailure(t *testing.T) {
	cfg := config.Config{GLMAPIKey: "g", DeepSeekAPIKey: "d"}
	report := runQuotaQueries(scriptedClient{delay: time.Millisecond}, cfg, 5*time.Second)
	if len(report.Entries) != 2 {
		t.Fatalf("两家供应商共 2 条条目，得到 %d", len(report.Entries))
	}
	if report.Entries[0].Provider != "glm" || report.Entries[1].Provider != "deepseek" {
		t.Errorf("条目顺序应为 glm → deepseek，得到 %s / %s", report.Entries[0].Provider, report.Entries[1].Provider)
	}
}
