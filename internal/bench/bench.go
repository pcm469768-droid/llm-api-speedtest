// Package bench 是中转站/LLM API 测速对比的核心:站点配置 schema、
// 并发测速执行与报告。只依赖标准库,零依赖便于分发与复刻。
package bench

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// DefaultPrompt 是测速用固定提示词:长度适中、输出 token 数稳定,
// 保证不同站点之间可比。
const DefaultPrompt = "用流畅的中文写一篇约200字的短文,介绍人工智能对日常生活的帮助。不要使用列表,直接分段。"

// Station 是一个被测中转站/API 站点。
type Station struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	BaseURL string   `json:"base_url"`
	Key     string   `json:"key,omitempty"`
	Models  []string `json:"models"`
}

// Config 是测速配置。
type Config struct {
	Stations  []Station `json:"stations"`
	Prompt    string    `json:"prompt,omitempty"`
	MaxTokens int       `json:"max_tokens,omitempty"`
}

// Result 是单个「站点×模型」的测速结果。
type Result struct {
	StationID    string    `json:"station_id"`
	StationName  string    `json:"station_name"`
	Model        string    `json:"model"`
	OK           bool      `json:"ok"`
	Error        string    `json:"error,omitempty"`
	TotalMS      int64     `json:"total_ms"`
	TTFBMS       int64     `json:"ttfb_ms"`
	OutputTokens int       `json:"output_tokens"`
	TokensPerSec float64   `json:"tokens_per_sec"`
	At           time.Time `json:"at"`
}

// Report 是一轮完整测速报告。
type Report struct {
	At      time.Time `json:"at"`
	Results []Result  `json:"results"`
}

// LoadConfig 读取配置;prompt/max_tokens 缺省时用默认值。
func LoadConfig(path string) (Config, error) {
	data, err := readFile(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("配置解码: %w", err)
	}
	if cfg.Prompt == "" {
		cfg.Prompt = DefaultPrompt
	}
	if cfg.MaxTokens <= 0 {
		cfg.MaxTokens = 300
	}
	if len(cfg.Stations) == 0 {
		return Config{}, fmt.Errorf("配置里没有站点:请先在 stations.json 中添加")
	}
	for _, s := range cfg.Stations {
		if s.ID == "" || s.BaseURL == "" {
			return Config{}, fmt.Errorf("站点 %q 缺少 id 或 base_url", s.Name)
		}
		if len(s.Models) == 0 {
			return Config{}, fmt.Errorf("站点 %q 没有模型列表", s.Name)
		}
	}
	return cfg, nil
}

// Run 并发测速全部「站点×模型」组合,返回按 Token/s 降序的报告。
func Run(ctx context.Context, cfg Config, concurrency int, timeout time.Duration) (Report, error) {
	if concurrency <= 0 {
		concurrency = 4
	}
	type task struct {
		station Station
		model   string
	}
	var tasks []task
	for _, s := range cfg.Stations {
		for _, m := range s.Models {
			tasks = append(tasks, task{station: s, model: m})
		}
	}
	results := make([]Result, len(tasks))
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	for i, t := range tasks {
		wg.Add(1)
		go func(i int, t task) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[i] = testOne(ctx, t.station, t.model, cfg.Prompt, cfg.MaxTokens, timeout)
		}(i, t)
	}
	wg.Wait()
	sort.SliceStable(results, func(a, b int) bool {
		return results[a].TokensPerSec > results[b].TokensPerSec
	})
	return Report{At: time.Now().UTC(), Results: results}, nil
}

// testOne 测试单个「站点×模型」:非流式 chat completion,记录 TTFB、
// 总耗时与输出 token 数。
func testOne(ctx context.Context, s Station, model, prompt string, maxTokens int, timeout time.Duration) Result {
	r := Result{StationID: s.ID, StationName: s.Name, Model: model, At: time.Now().UTC()}
	body := map[string]any{
		"model":       model,
		"messages":    []map[string]string{{"role": "user", "content": prompt}},
		"max_tokens":  maxTokens,
		"temperature": 0,
		"stream":      false,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		r.Error = err.Error()
		return r
	}
	endpoint := strings.TrimRight(s.BaseURL, "/") + "/v1/chat/completions"
	reqCtx := ctx
	cancel := func() {}
	if timeout > 0 {
		reqCtx, cancel = context.WithTimeout(ctx, timeout)
	}
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		r.Error = err.Error()
		return r
	}
	req.Header.Set("Content-Type", "application/json")
	if s.Key != "" {
		req.Header.Set("Authorization", "Bearer "+s.Key)
	}
	client := &http.Client{Transport: &http.Transport{}, Timeout: timeout}
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		r.Error = err.Error()
		r.TotalMS = time.Since(start).Milliseconds()
		return r
	}
	defer resp.Body.Close()
	r.TTFBMS = time.Since(start).Milliseconds()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		r.Error = err.Error()
		r.TotalMS = time.Since(start).Milliseconds()
		return r
	}
	r.TotalMS = time.Since(start).Milliseconds()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		r.Error = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, truncate(string(data), 200))
		return r
	}
	var out struct {
		Usage struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		r.Error = "响应解码: " + err.Error()
		return r
	}
	if out.Error != nil {
		r.Error = out.Error.Message
		return r
	}
	r.OK = true
	r.OutputTokens = out.Usage.TotalTokens
	if r.TotalMS > 0 && r.OutputTokens > 0 {
		r.TokensPerSec = float64(r.OutputTokens) / (float64(r.TotalMS) / 1000.0)
	}
	return r
}

// FormatTable 生成按 Token/s 排名的可读表格。
func FormatTable(rep Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "更新时间:%s(UTC)\n\n", rep.At.Format("2006-01-02 15:04"))
	b.WriteString("| 排名 | 中转站 | 模型 | 速度(Token/s) | 延迟(TTFB) | 耗时 | 状态 |\n")
	b.WriteString("|---|---|---|---|---|---|---|\n")
	for i, r := range rep.Results {
		status := "✅"
		if !r.OK {
			status = "❌ " + truncate(r.Error, 40)
		}
		speed := "—"
		if r.OK {
			speed = fmt.Sprintf("%.1f", r.TokensPerSec)
		}
		b.WriteString(fmt.Sprintf("| %d | %s | %s | %s | %dms | %.1fs | %s |\n",
			i+1, r.StationName, r.Model, speed, r.TTFBMS, float64(r.TotalMS)/1000.0, status))
	}
	return b.String()
}

// WriteReport 把报告写入 JSON 文件。
func WriteReport(path string, rep Report) error {
	data, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeFile(path, data, 0o644)
}

// 排行榜在 README 中的占位标记。
const (
	markerStart = "<!-- BENCH:RANKING -->"
	markerEnd   = "<!-- /BENCH:RANKING -->"
)

// UpdateREADME 用报告替换 README 中排行榜占位块;缺标记时在文件末尾
// 追加排行榜块。
func UpdateREADME(readme string, rep Report) (string, error) {
	table := FormatTable(rep)
	block := "\n" + markerStart + "\n" + table + "\n" + markerEnd + "\n"
	if strings.Contains(readme, markerStart) {
		start := strings.Index(readme, markerStart)
		end := strings.Index(readme, markerEnd) + len(markerEnd)
		return readme[:start] + strings.TrimSpace(block) + "\n" + readme[end:], nil
	}
	return readme + block, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
