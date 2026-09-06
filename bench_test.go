package bench

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeLLM 起一个 OpenAI 兼容假服务,按固定延迟返回固定 token 数。
func fakeLLM(t *testing.T, delay time.Duration, totalTokens int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			http.NotFound(w, r)
			return
		}
		time.Sleep(delay)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    "chatcmpl-test",
			"usage": map[string]int{"total_tokens": totalTokens},
		})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestLoadConfigDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stations.json")
	cfg := Config{
		Stations: []Station{{ID: "s1", Name: "站点1", BaseURL: "https://x.example", Models: []string{"m1"}}},
	}
	data, _ := json.Marshal(cfg)
	if err := writeFile(path, data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if got.Prompt != DefaultPrompt || got.MaxTokens != 300 {
		t.Fatalf("默认值缺失: %+v", got)
	}
}

func TestLoadConfigRejectsEmptyStations(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "s.json")
	_ = writeFile(path, []byte(`{"stations":[]}`), 0o644)
	if _, err := LoadConfig(path); err == nil {
		t.Fatal("空站点列表应报错")
	}
}

func TestRunRanksBySpeed(t *testing.T) {
	fast := fakeLLM(t, 100*time.Millisecond, 500) // 5000 tok/s
	slow := fakeLLM(t, 300*time.Millisecond, 300) // 1000 tok/s
	cfg := Config{
		Prompt:    "hi",
		MaxTokens: 300,
		Stations: []Station{
			{ID: "fast", Name: "快站", BaseURL: fast.URL, Models: []string{"m"}},
			{ID: "slow", Name: "慢站", BaseURL: slow.URL, Models: []string{"m"}},
		},
	}
	rep, err := Run(context.Background(), cfg, 2, 10*time.Second)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(rep.Results) != 2 {
		t.Fatalf("结果数量: %d", len(rep.Results))
	}
	if !rep.Results[0].OK || !rep.Results[1].OK {
		t.Fatalf("应全部成功: %+v", rep.Results)
	}
	if rep.Results[0].StationID != "fast" {
		t.Fatalf("应按速度降序,首个应为 fast: %+v", rep.Results)
	}
	if rep.Results[0].TokensPerSec <= rep.Results[1].TokensPerSec {
		t.Fatalf("速度排序错误: %v vs %v", rep.Results[0].TokensPerSec, rep.Results[1].TokensPerSec)
	}
	if rep.Results[0].OutputTokens != 500 || rep.Results[0].TTFBMS <= 0 {
		t.Fatalf("指标缺失: %+v", rep.Results[0])
	}
}

func TestRunCapturesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"无效密钥"}}`))
	}))
	defer srv.Close()
	cfg := Config{
		Prompt:    "hi",
		MaxTokens: 100,
		Stations:  []Station{{ID: "bad", Name: "坏站", BaseURL: srv.URL, Models: []string{"m"}}},
	}
	rep, err := Run(context.Background(), cfg, 1, 5*time.Second)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rep.Results[0].OK || !strings.Contains(rep.Results[0].Error, "401") {
		t.Fatalf("应记录 HTTP 错误: %+v", rep.Results[0])
	}
}

func TestRunTimeout(t *testing.T) {
	srv := fakeLLM(t, 3*time.Second, 100)
	cfg := Config{
		Prompt:    "hi",
		MaxTokens: 100,
		Stations:  []Station{{ID: "t", Name: "超时站", BaseURL: srv.URL, Models: []string{"m"}}},
	}
	rep, err := Run(context.Background(), cfg, 1, 500*time.Millisecond)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rep.Results[0].OK {
		t.Fatalf("超时应失败: %+v", rep.Results[0])
	}
}

func TestUpdateREADME(t *testing.T) {
	rep := Report{
		At: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		Results: []Result{{
			StationID: "s1", StationName: "站点1", Model: "m1", OK: true,
			TTFBMS: 100, TotalMS: 500, OutputTokens: 400, TokensPerSec: 800,
		}},
	}
	readme := "标题\n\n" + markerStart + "\n旧内容\n" + markerEnd + "\n\n结尾"
	out, err := UpdateREADME(readme, rep)
	if err != nil {
		t.Fatalf("UpdateREADME: %v", err)
	}
	if !strings.Contains(out, "站点1") || strings.Contains(out, "旧内容") {
		t.Fatalf("排行榜未替换: %s", out)
	}
	if !strings.Contains(out, "800.0") {
		t.Fatalf("缺速度数据: %s", out)
	}
	// 无标记时追加。
	out2, err := UpdateREADME("纯文本", rep)
	if err != nil || !strings.Contains(out2, markerStart) {
		t.Fatalf("应追加排行榜块: %v / %s", err, out2)
	}
}

func TestFormatTableContainsHeader(t *testing.T) {
	table := FormatTable(Report{At: time.Now().UTC()})
	for _, want := range []string{"排名", "Token/s", "TTFB"} {
		if !strings.Contains(table, want) {
			t.Fatalf("表格缺 %q: %s", want, table)
		}
	}
}
