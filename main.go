package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
)

// --- 設定 ---
var (
	NetboxURL   = os.Getenv("NETBOX_URL")
	NetboxToken = os.Getenv("NETBOX_TOKEN")
)

// --- データ構造 ---
type CableResponse struct {
	Results []struct {
		ATerminations []Termination `json:"a_terminations"`
		BTerminations []Termination `json:"b_terminations"`
	} `json:"results"`
}
type Termination struct {
	Object struct {
		Name   string `json:"name"`
		Device struct {
			Name string `json:"name"`
			Role struct { Slug string `json:"slug"` } `json:"role"`
		} `json:"device"`
	} `json:"object"`
}

// デバイスごとのポートリストを保持
type DeviceData struct {
	Name  string
	Role  string
	Ports map[string]bool // 重複排除用
}

func main() {
	// 1. NetBoxからケーブル情報を取得
	client := &http.Client{}
	req, _ := http.NewRequest("GET", NetboxURL+"/api/dcim/cables/?limit=0", nil)
	req.Header.Set("Authorization", "Token "+NetboxToken)
	resp, err := client.Do(req)
	if err != nil { panic(err) }
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var data CableResponse
	json.Unmarshal(body, &data)

	// 2. データの整理 (デバイスごとのポート一覧を作る)
	devices := make(map[string]*DeviceData)
	connections := []string{}

	for _, cable := range data.Results {
		if len(cable.ATerminations) == 0 || len(cable.BTerminations) == 0 { continue }
		termA := cable.ATerminations[0].Object
		termB := cable.BTerminations[0].Object

		// デバイスとポートを登録
		registerDevice(devices, termA.Device.Name, termA.Device.Role.Slug, termA.Name)
		registerDevice(devices, termB.Device.Name, termB.Device.Role.Slug, termB.Name)

		// 接続定義 (DOT言語用)
		// Graphvizでは "DevA":"PortA" -> "DevB":"PortB" と書くことでポート間接続になる
		conn := fmt.Sprintf(`  "%s":"%s" -> "%s":"%s"`, 
			termA.Device.Name, escape(termA.Name),
			termB.Device.Name, escape(termB.Name))
		connections = append(connections, conn)
	}

	// 3. DOTファイルの生成
	file, _ := os.Create("topology.dot")
	defer file.Close()

	// ヘッダー書き込み
	// splines=ortho: カクカクした配線にする
	// rankdir=TB: 上から下へ配置
	file.WriteString("digraph NetworkTopology {\n")
	file.WriteString("  rankdir=TB;\n")
	file.WriteString("  nodesep=1.0;\n") // ノード間の横幅
	file.WriteString("  ranksep=1.5;\n") // 階層間の縦幅
	file.WriteString("  splines=ortho;\n") 
	file.WriteString("  node [shape=plain];\n") // HTMLラベルを使うためplainにする

	// ノード（デバイス）の定義
	for _, dev := range devices {
		// ポート名をソート
		var ports []string
		for p := range dev.Ports {
			ports = append(ports, p)
		}
		sort.Strings(ports)

		// HTMLテーブルの構築
		// <TABLE>
		//   <TR><TD>Device Name</TD></TR>
		//   <TR><TD PORT="p1">Eth1</TD><TD PORT="p2">Eth2</TD>...</TR>
		// </TABLE>
		
		color := "#E0E0E0" // デフォルト色
		if strings.Contains(dev.Role, "core") { color = "#FFCCCC" }
		if strings.Contains(dev.Role, "access") { color = "#CCFFCC" }

		label := fmt.Sprintf(`<<TABLE BORDER="0" CELLBORDER="1" CELLSPACING="0" BGCOLOR="%s">`, color)
		// ヘッダー行（デバイス名）
		label += fmt.Sprintf(`<TR><TD COLSPAN="%d"><B>%s</B></TD></TR>`, len(ports), dev.Name)
		
		// ポート行
		label += "<TR>"
		for _, p := range ports {
			// PORT="..." が接続点になるID
			label += fmt.Sprintf(`<TD PORT="%s" BGCOLOR="#FFFFFF">%s</TD>`, escape(p), p)
		}
		label += "</TR></TABLE>>"

		file.WriteString(fmt.Sprintf(`  "%s" [label=%s];`+"\n", dev.Name, label))
	}

	// 接続の書き込み
	for _, conn := range connections {
		file.WriteString(conn + ";\n")
	}

	file.WriteString("}\n")
	fmt.Println("Generated topology.dot")
}

func registerDevice(devices map[string]*DeviceData, name, role, port string) {
	if _, ok := devices[name]; !ok {
		devices[name] = &DeviceData{Name: name, Role: role, Ports: make(map[string]bool)}
	}
	devices[name].Ports[port] = true
}

func escape(s string) string {
	// GraphvizのIDに使えない文字をエスケープまたは置換
	s = strings.ReplaceAll(s, ":", "_")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, " ", "_")
	return s
}
