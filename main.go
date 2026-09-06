// relay-bench 是中转站/LLM API 测速对比 CLI:
//   relay-bench run -config stations.json [-concurrency 4] [-timeout 120s] [-json report.json]
//   relay-bench write-readme -report report.json -readme README.md
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/pcm469768-droid/llm-api-speedtest/internal/bench"
)

func main() {
	log.SetPrefix("[relay-bench] ")
	runCmd := flag.NewFlagSet("run", flag.ExitOnError)
	var (
		configPath  = runCmd.String("config", "stations.json", "测速配置")
		concurrency = runCmd.Int("concurrency", 4, "并发数")
		timeoutSec  = runCmd.Int("timeout", 120, "单次请求超时(秒)")
		jsonPath    = runCmd.String("json", "", "报告输出路径(可选)")
	)
	readmeCmd := flag.NewFlagSet("write-readme", flag.ExitOnError)
	var (
		reportPath = readmeCmd.String("report", "report.json", "报告 JSON 路径")
		readmePath = readmeCmd.String("readme", "README.md", "README 路径")
	)

	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "run":
		_ = runCmd.Parse(os.Args[2:])
		cfg, err := bench.LoadConfig(*configPath)
		if err != nil {
			log.Fatalf("配置: %v", err)
		}
		report, err := bench.Run(context.Background(), cfg, *concurrency, time.Duration(*timeoutSec)*time.Second)
		if err != nil {
			log.Fatalf("测速: %v", err)
		}
		fmt.Print(bench.FormatTable(report))
		if *jsonPath != "" {
			if err := bench.WriteReport(*jsonPath, report); err != nil {
				log.Fatalf("写报告: %v", err)
			}
			log.Printf("报告已写入 %s", *jsonPath)
		}
	case "write-readme":
		_ = readmeCmd.Parse(os.Args[2:])
		report, err := loadReport(*reportPath)
		if err != nil {
			log.Fatalf("报告: %v", err)
		}
		readme, err := os.ReadFile(*readmePath)
		if err != nil {
			log.Fatalf("README: %v", err)
		}
		updated, err := bench.UpdateREADME(string(readme), report)
		if err != nil {
			log.Fatalf("更新 README: %v", err)
		}
		if err := os.WriteFile(*readmePath, []byte(updated), 0o644); err != nil {
			log.Fatalf("写 README: %v", err)
		}
		log.Printf("README 排行榜已更新")
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "用法:\n")
	fmt.Fprintf(os.Stderr, "  relay-bench run -config stations.json [-concurrency 4] [-timeout 120] [-json report.json]\n")
	fmt.Fprintf(os.Stderr, "  relay-bench write-readme -report report.json -readme README.md\n")
}

func loadReport(path string) (bench.Report, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return bench.Report{}, err
	}
	var rep bench.Report
	if err := json.Unmarshal(data, &rep); err != nil {
		return bench.Report{}, fmt.Errorf("报告解码: %w", err)
	}
	return rep, nil
}
