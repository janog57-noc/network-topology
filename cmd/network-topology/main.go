package main

import (
	"fmt"
	"log"

	"github.com/janog57-noc/network-topology/internal/config"
	"github.com/janog57-noc/network-topology/internal/netbox"
	"github.com/janog57-noc/network-topology/internal/renderer"
	"github.com/janog57-noc/network-topology/internal/service"
)

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

	// 5. レンダリング
	fmt.Println("Generating DOT file...")
	if err := renderer.GenerateDOT(topology, "topology.dot"); err != nil {
		log.Fatalf("Failed to generate DOT: %v", err)
	}

	fmt.Println("Generating Images (SVG/PNG)...")
	if err := renderer.GenerateImages("topology.dot", "topology.svg", "topology.png"); err != nil {
		log.Printf("Warning: Image generation failed: %v", err)
		log.Println("Ensure Graphviz is installed.")
	}

	fmt.Println("Generating HTML...")
	if err := renderer.GenerateHTML(topology, "topology.svg", "index.html"); err != nil {
		log.Printf("Warning: HTML generation failed: %v", err)
	}

	fmt.Println("Done!")
}
