package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// 設定 (環境変数から取得)
var (
	NetboxURL   = os.Getenv("NETBOX_URL")
	NetboxToken = os.Getenv("NETBOX_TOKEN")
)

// NetBox APIのレスポンス構造体 (必要な分だけ定義)
type CableResponse struct {
	Results []struct {
		ATerminations []Termination `json:"a_terminations"`
		BTerminations []Termination `json:"b_terminations"`
	} `json:"results"`
}

type Termination struct {
	Object struct {
		Name   string `json:"name"` // ポート名 (lan3:10)
		Device struct {
			Name string `json:"name"` // デバイス名 (RT-J3-01)
		} `json:"device"`
	} `json:"object"`
}

func main() {
	if NetboxURL == "" || NetboxToken == "" {
		fmt.Println("Error: NETBOX_URL or NETBOX_TOKEN is not set")
		os.Exit(1)
	}

	// 1. NetBoxからケーブル一覧を取得
	// ※件数が多い場合は &limit=0 を付ける
	req, _ := http.NewRequest("GET", NetboxURL+"/api/dcim/cables/?limit=0", nil)
	req.Header.Set("Authorization", "Token "+NetboxToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var data CableResponse
	if err := json.Unmarshal(body, &data); err != nil {
		panic(err)
	}

	// 2. D2フォーマットの構築
	// デバイスとポートのセットを保持して重複を防ぐ
	devices := make(map[string]map[string]bool) // map[DeviceName]map[PortName]bool
	connections := []string{}

	for _, cable := range data.Results {
		if len(cable.ATerminations) == 0 || len(cable.BTerminations) == 0 {
			continue
		}
		a := cable.ATerminations[0].Object
		b := cable.BTerminations[0].Object

		// デバイスとポートを登録
		addDevicePort(devices, a.Device.Name, a.Name)
		addDevicePort(devices, b.Device.Name, b.Name)

		// 接続定義 (D2記法: DevA.PortA -- DevB.PortB)
		// IDにスペースや記号が含まれる場合のサニタイズはD2がダブルクォートで吸収してくれる
		conn := fmt.Sprintf(`"%s"."%s" -- "%s"."%s"`, a.Device.Name, a.Name, b.Device.Name, b.Name)
		connections = append(connections, conn)
	}

	// 3. ファイル出力 (topology.d2)
	file, _ := os.Create("topology.d2")
	defer file.Close()

	// D2ヘッダー書き込み
	file.WriteString("direction: right\n")
	file.WriteString("\n# Nodes (Devices and Ports)\n")

	// ノード定義書き込み
	for devName, ports := range devices {
		file.WriteString(fmt.Sprintf("\"%s\": {\n", devName))
		for portName := range ports {
			file.WriteString(fmt.Sprintf("  \"%s\"\n", portName))
		}
		file.WriteString("}\n")
	}

	file.WriteString("\n# Connections\n")
	for _, conn := range connections {
		file.WriteString(conn + "\n")
	}

	fmt.Println("Successfully generated topology.d2")
}

func addDevicePort(devices map[string]map[string]bool, dev, port string) {
	if _, ok := devices[dev]; !ok {
		devices[dev] = make(map[string]bool)
	}
	devices[dev][port] = true
}
