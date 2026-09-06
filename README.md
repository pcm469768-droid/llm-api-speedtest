<!-- markdownlint-disable MD033 MD041 -->

<div align="center">
  <h1>LLM API Speedtest</h1>
  <p>Benchmark LLM APIs, OpenAI-compatible gateways, and API proxies with repeatable latency and throughput measurements.</p>
  <p>
    <a href="https://github.com/pcm469768-droid/llm-api-speedtest/actions/workflows/bench.yml"><img alt="Bench workflow" src="https://github.com/pcm469768-droid/llm-api-speedtest/actions/workflows/bench.yml/badge.svg"></a>
    <a href="https://go.dev/"><img alt="Go 1.26" src="https://img.shields.io/badge/Go-1.26-00ADD8.svg?logo=go"></a>
    <a href="https://github.com/pcm469768-droid/llm-api-speedtest/blob/main/LICENSE"><img alt="License MIT" src="https://img.shields.io/badge/license-MIT-green.svg"></a>
  </p>
</div>

<p align="center"><a href="README.md">English</a> · <a href="README_CN.md">中文</a></p>

## Sponsor

<table>
  <tr>
    <td width="240" align="center">
      <a href="https://argolink.io"><img src="assets/argolink.png" alt="ArgoLink" width="220"></a>
    </td>
    <td>
      <strong>ArgoLink</strong><br>
      Recommended / sponsored relay for LLM API and OpenAI-compatible gateway comparisons.<br>
      <a href="https://argolink.io">Visit ArgoLink</a>
    </td>
  </tr>
</table>

## Overview

**LLM API Speedtest** is a small Go CLI for comparing **LLM API**, **OpenAI-compatible API**, **LLM gateway**, and **API proxy** endpoints. It sends the same request to every configured station and records:

- **Tokens/s** — output throughput;
- **TTFB** — time to first byte;
- **Total latency** — complete request time;
- **Success rate and errors** — HTTP and response-level failures.

The tool is useful when you need to compare API gateways, relay services, model routers, or your own OpenAI-compatible deployment. It has no third-party Go dependencies and can run as a single binary or in **GitHub Actions**.

## Metrics

| Metric | Meaning | Better result |
| --- | --- | --- |
| Tokens/s | Output tokens divided by total request time | Higher |
| TTFB | Time until the response begins | Lower |
| Total latency | Time until the complete response is read | Lower |
| Status | Successful response or a concise error | ✅ |

Each station/model pair uses the same prompt, `temperature: 0`, `stream: false`, and default `max_tokens: 300`, so runs are easier to compare.

## Quick start

Requires Go 1.26 or newer.

```bash
git clone https://github.com/pcm469768-droid/llm-api-speedtest.git
cd llm-api-speedtest

# Keep real credentials in the ignored local file.
cp stations.example.json stations.json
# Edit stations.json with your endpoint, key, and model names.

go run ./cmd/llm-api-speedtest run \
  -config stations.json \
  -concurrency 4 \
  -timeout 120 \
  -json report.json

# Replace the ranking block in this README with the new report.
go run ./cmd/llm-api-speedtest write-readme \
  -report report.json \
  -readme README.md
```

You can also build a portable binary:

```bash
go build -o llm-api-speedtest ./cmd/llm-api-speedtest
./llm-api-speedtest run -config stations.json -json report.json
```

## Configure stations

Copy `stations.example.json` to `stations.json`. Every station needs an `id`, a display `name`, a `base_url`, and one or more model names:

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

The request is sent to `<base_url>/v1/chat/completions` using the OpenAI-compatible Chat Completions format. Never commit real API keys. For scheduled runs, inject credentials through repository secrets and keep the public example file limited to placeholders.

## Current ranking

<!-- BENCH:RANKING -->
Run `llm-api-speedtest run` and then `write-readme` to populate this table with a real report.
<!-- /BENCH:RANKING -->

## Automated runs

The [bench workflow](https://github.com/pcm469768-droid/llm-api-speedtest/actions/workflows/bench.yml) can be started manually or scheduled daily. It checks out the repository, runs the configured benchmark, and updates the ranking block. Review the station configuration and secret handling before enabling a public scheduled run.

## Hosted endpoint

The benchmark is compatible with hosted endpoints, your own gateway, and relay services, so you can compare the same model and prompt across providers.

## Development

```bash
go test ./...
```

The package tests cover configuration validation, concurrent execution, report formatting, and README ranking updates.

## License

Released under the [MIT License](https://github.com/pcm469768-droid/llm-api-speedtest/blob/main/LICENSE).

