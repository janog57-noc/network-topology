package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/janog57-noc/network-topology/internal/config"
	"github.com/janog57-noc/network-topology/internal/netbox"
	"github.com/janog57-noc/network-topology/internal/renderer"
	"github.com/janog57-noc/network-topology/internal/service"
)

const outDir = "out"

func main() {
	// 1. 設定読み込み
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Config error: %v", err)
	}

	// 2. NetBoxクライアント初期化
	client := netbox.NewClient(cfg.NetboxURL, cfg.NetboxToken)

	// 3. データ取得
	fmt.Println("Fetching data from NetBox...")
	devices, err := client.FetchDevices()
	if err != nil {
		log.Fatalf("Failed to fetch devices: %v", err)
	}
	interfaces, err := client.FetchInterfaces()
	if err != nil {
		log.Fatalf("Failed to fetch interfaces: %v", err)
	}
	cables, err := client.FetchCables()
	if err != nil {
		log.Fatalf("Failed to fetch cables: %v", err)
	}

	// 4. トポロジーデータ構築
	fmt.Println("Building topology...")
	builder := service.NewBuilder()
	topology := builder.Build(devices, interfaces, cables)

	// 5. 出力ディレクトリ作成
	if err := os.MkdirAll(outDir, 0755); err != nil {
		log.Fatalf("Failed to create output dir: %v", err)
	}

	dotPath := filepath.Join(outDir, "topology.dot")
	svgPath := filepath.Join(outDir, "topology.svg")
	pngPath := filepath.Join(outDir, "topology.png")
	htmlPath := filepath.Join(outDir, "index.html")

	fmt.Println("Generating DOT file...")
	if err := renderer.GenerateDOT(topology, dotPath); err != nil {
		log.Fatalf("Failed to generate DOT: %v", err)
	}

	fmt.Println("Generating Images (SVG/PNG)...")
	if err := renderer.GenerateImages(dotPath, svgPath, pngPath); err != nil {
		log.Printf("Warning: Image generation failed: %v", err)
		log.Println("Ensure Graphviz is installed.")
	}

	fmt.Println("Generating HTML...")
	if err := renderer.GenerateHTML(topology, svgPath, htmlPath); err != nil {
		log.Printf("Warning: HTML generation failed: %v", err)
	}

	fmt.Println("Done!")
}
