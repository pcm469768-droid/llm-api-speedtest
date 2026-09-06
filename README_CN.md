<!-- markdownlint-disable MD033 MD041 -->

<div align="center">
  <h1>LLM API Speedtest</h1>
  <p>用可复现的延迟和吞吐量指标，对比 LLM API、OpenAI 兼容网关和 API 中转站。</p>
  <p>
    <a href="https://github.com/pcm469768-droid/llm-api-speedtest/actions/workflows/bench.yml"><img alt="Bench workflow" src="https://github.com/pcm469768-droid/llm-api-speedtest/actions/workflows/bench.yml/badge.svg"></a>
    <a href="https://go.dev/"><img alt="Go 1.26" src="https://img.shields.io/badge/Go-1.26-00ADD8.svg?logo=go"></a>
    <a href="https://github.com/pcm469768-droid/llm-api-speedtest/blob/main/LICENSE"><img alt="License MIT" src="https://img.shields.io/badge/license-MIT-green.svg"></a>
  </p>
</div>

<p align="center"><a href="README.md">English</a> · <a href="README_CN.md">中文</a></p>

> 🔥 推荐 / 赞助中转：<a href="https://argolink.io">ArgoLink</a> — 可作为测速对比中的托管接口。

## 赞助中转

<table>
  <tr>
    <td width="240" align="center">
      <a href="https://argolink.io"><img src="assets/argolink.png" alt="ArgoLink" width="220"></a>
    </td>
    <td>
      <strong>ArgoLink</strong><br>
      推荐 / 赞助的 LLM API 与 OpenAI 兼容网关中转服务。<br>
      <a href="https://argolink.io">访问 ArgoLink 官网</a>
    </td>
  </tr>
</table>

## 项目简介

**LLM API Speedtest** 是一个轻量 Go 命令行工具，用于对比 **LLM API**、**OpenAI 兼容 API**、**LLM gateway** 和 **API proxy**。它向每个配置的站点发送相同请求，并记录：

- **Tokens/s**：输出吞吐速度；
- **TTFB**：首字节延迟；
- **总耗时**：完整响应耗时；
- **成功状态和错误**：HTTP 及响应级别的失败原因。

它适合比较 API 网关、中转服务、模型路由器和自己的 OpenAI 兼容部署。项目只使用 Go 标准库，可编译成单个二进制文件，也可以在 **GitHub Actions** 中运行。

## 指标说明

| 指标 | 含义 | 越好 |
| --- | --- | --- |
| Tokens/s | 输出 token 数 ÷ 请求总耗时 | 越高越好 |
| TTFB | 收到响应首字节的时间 | 越低越好 |
| 总耗时 | 读完完整响应的时间 | 越低越好 |
| 状态 | 成功响应或简短错误 | ✅ |

每个站点和模型组合使用相同提示词、`temperature: 0`、`stream: false`，默认 `max_tokens: 300`，方便横向比较。

## 快速开始

需要 Go 1.26 或更高版本。

```bash
git clone https://github.com/pcm469768-droid/llm-api-speedtest.git
cd llm-api-speedtest

# 真实密钥只放在被忽略的本地文件中
cp stations.example.json stations.json
# 编辑 stations.json，填写接口地址、密钥和模型名

go run ./cmd/llm-api-speedtest run \\
  -config stations.json \\
  -concurrency 4 \\
  -timeout 120 \\
  -json report.json

# 用最新报告替换本 README 的排行榜区域
go run ./cmd/llm-api-speedtest write-readme \\
  -report report.json \\
  -readme README.md
```

也可以编译成可分发的二进制文件：

```bash
go build -o llm-api-speedtest ./cmd/llm-api-speedtest
./llm-api-speedtest run -config stations.json -json report.json
```

## 配置站点

复制 `stations.example.json` 为 `stations.json`。每个站点需要 `id`、显示名称 `name`、`base_url` 和一个或多个模型名：

```json
{
  "stations": [
    {
      "id": "my-gateway",
      "name": "My gateway",
      "base_url": "https://api.example.com",
      "key": "YOUR_API_KEY",
      "models": ["gpt-4o-mini"]
    }
  ]
}
```

请求会按 OpenAI 兼容 Chat Completions 格式发送到 `<base_url>/v1/chat/completions`。不要提交真实 API 密钥；定时运行时请使用仓库 Secrets 注入，并让公开示例文件只保留占位符。

## 当前排行榜

<!-- BENCH:RANKING -->
运行 `llm-api-speedtest run`，再运行 `write-readme`，即可用真实报告填充此表。
<!-- /BENCH:RANKING -->

## 自动测速

[bench workflow](https://github.com/pcm469768-droid/llm-api-speedtest/actions/workflows/bench.yml) 支持手动触发和每日定时运行。它会检出仓库、执行测速并更新排行榜。公开启用定时任务前，请先检查站点配置和密钥处理方式。

## 推荐中转

工具兼容托管接口、自己的网关和中转站，因此可以用相同模型和提示词比较不同服务商。

## 开发

```bash
go test ./...
```

测试覆盖配置校验、并发执行、报告格式化和 README 排行榜更新。

## 许可证

本项目采用 [MIT License](https://github.com/pcm469768-droid/llm-api-speedtest/blob/main/LICENSE)。
