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

type DeviceData struct {
	Name  string
	Role  string
	Ports map[string]bool
}

func main() {
	// 1. NetBoxデータ取得
	client := &http.Client{}
	req, _ := http.NewRequest("GET", NetboxURL+"/api/dcim/cables/?limit=0", nil)
	req.Header.Set("Authorization", "Token "+NetboxToken)
	resp, err := client.Do(req)
	if err != nil { panic(err) }
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var data CableResponse
	json.Unmarshal(body, &data)

	// 2. データ整理
	devices := make(map[string]*DeviceData)
	type Connection struct {
		SrcDev, SrcPort string
		DstDev, DstPort string
		SrcLevel, DstLevel int
	}
	var connections []Connection

	for _, cable := range data.Results {
		if len(cable.ATerminations) == 0 || len(cable.BTerminations) == 0 { continue }
		termA := cable.ATerminations[0].Object
		termB := cable.BTerminations[0].Object

		registerDevice(devices, termA.Device.Name, termA.Device.Role.Slug, termA.Name)
		registerDevice(devices, termB.Device.Name, termB.Device.Role.Slug, termB.Name)

		levelA := getRoleLevel(termA.Device.Role.Slug)
		levelB := getRoleLevel(termB.Device.Role.Slug)

		// 上位レベル(数値が小さい)をSrc、下位レベルをDstにする
		if levelA <= levelB {
			connections = append(connections, Connection{
				SrcDev: termA.Device.Name, SrcPort: termA.Name, SrcLevel: levelA,
				DstDev: termB.Device.Name, DstPort: termB.Name, DstLevel: levelB,
			})
		} else {
			connections = append(connections, Connection{
				SrcDev: termB.Device.Name, SrcPort: termB.Name, SrcLevel: levelB,
				DstDev: termA.Device.Name, DstPort: termA.Name, DstLevel: levelA,
			})
		}
	}

	// 3. DOT生成
	file, _ := os.Create("topology.dot")
	defer file.Close()

	// --- ライトテーマ設定 ---
	file.WriteString("digraph NetworkTopology {\n")
	file.WriteString("  bgcolor=\"#FFFFFF\";\n") // 背景: 白
	file.WriteString("  rankdir=TB;\n")         // Top to Bottom
	file.WriteString("  nodesep=0.8;\n")        // ノード間の横幅
	file.WriteString("  ranksep=1.2;\n")        // 階層間の縦幅
	file.WriteString("  splines=ortho;\n")      // カクカクした配線
	
	// ノードフォント設定 (黒文字)
	file.WriteString("  node [shape=plain fontname=\"Helvetica\" fontcolor=\"black\"];\n")
	
	// エッジ設定 (線はダークグレー)
	file.WriteString("  edge [dir=none style=solid penwidth=2.0 color=\"#555555\"];\n") 

	// ノード書き込み
	for _, dev := range devices {
		var ports []string
		for p := range dev.Ports {
			ports = append(ports, p)
		}
		sort.Strings(ports)

		// 色の決定 (ヘッダー色のみ取得)
		headerColor := getHeaderColor(dev.Role)

		// HTML Label (全体枠線は薄いグレー)
		label := fmt.Sprintf(`<<TABLE BORDER="0" CELLBORDER="1" CELLSPACING="0" COLOR="#888888" BGCOLOR="#FFFFFF">`)
		
		// ヘッダー行 (背景色あり、文字は白で強調)
		// POINT-SIZEでフォントサイズ調整
		label += fmt.Sprintf(`<TR><TD COLSPAN="%d" BGCOLOR="%s" HEIGHT="30"><B><FONT COLOR="#FFFFFF" POINT-SIZE="14">%s</FONT></B></TD></TR>`, 
			len(ports), headerColor, dev.Name)
		
		// ポート一覧行
		label += "<TR>"
		for _, p := range ports {
			// ポートセル: 背景は非常に薄いグレー(#F9F9F9)、文字は黒
			// WIDTHを高めに設定してクリックしやすく
			label += fmt.Sprintf(`<TD PORT="%s" WIDTH="40" HEIGHT="24" BGCOLOR="#F9F9F9"><FONT COLOR="#000000" POINT-SIZE="10">%s</FONT></TD>`, escape(p), p)
		}
		label += "</TR></TABLE>>"

		file.WriteString(fmt.Sprintf(`  "%s" [label=%s];`+"\n", dev.Name, label))
	}

	// 接続書き込み
	for _, conn := range connections {
		srcComp := ":s" // 上位は下から出す
		dstComp := ":n" // 下位は上から受ける

		if conn.SrcLevel == conn.DstLevel {
			dstComp = ":s" // 同階層なら下-下接続
		}

		line := fmt.Sprintf(`  "%s":"%s"%s -> "%s":"%s"%s`, 
			conn.SrcDev, escape(conn.SrcPort), srcComp,
			conn.DstDev, escape(conn.DstPort), dstComp)
		
		file.WriteString(line + ";\n")
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

func getRoleLevel(role string) int {
	if strings.Contains(role, "core") || strings.Contains(role, "router") { return 1 }
	if strings.Contains(role, "distribution") { return 2 }
	if strings.Contains(role, "access") || strings.Contains(role, "switch") { return 3 }
	if strings.Contains(role, "ap") || strings.Contains(role, "server") { return 4 }
	return 99
}

func getHeaderColor(role string) string {
	switch {
	case strings.Contains(role, "router") || strings.Contains(role, "firewall") || strings.Contains(role, "onu"):
		return "#D9534F" // Bootstrap Red (Router)
	case strings.Contains(role, "core"):
		return "#F0AD4E" // Bootstrap Orange (Core)
	case strings.Contains(role, "distribution"):
		return "#5BC0DE" // Bootstrap Cyan (Dist)
	case strings.Contains(role, "access"):
		return "#428BCA" // Bootstrap Blue (Access)
	case strings.Contains(role, "ap"):
		return "#9B59B6" // Purple (Wireless)
	case strings.Contains(role, "server"):
		return "#777777" // Grey (Server)
	default:
		return "#333333" // Dark Grey (Others)
	}
}

func escape(s string) string {
	s = strings.ReplaceAll(s, ":", "_")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, ".", "_")
	return s
}
