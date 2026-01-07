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

// カラーパレット構造体
type ThemeColor struct {
	Header   string
	Border   string
	Text     string
	PortFill string
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

	// --- Graphviz設定 (ここがズレ解消の肝) ---
	file.WriteString("digraph NetworkTopology {\n")
	file.WriteString("  bgcolor=\"#FFFFFF\";\n") // 背景: 白
	file.WriteString("  rankdir=TB;\n")         // Top to Bottom
	
	// ★★★ ズレ解消ポイント1: 間隔を広げる ★★★
	// 狭いとGraphvizが線を束ねてしまい、ポート位置がズレます
	file.WriteString("  nodesep=1.5;\n")        // ノード間の横幅 (0.8 -> 1.5)
	file.WriteString("  ranksep=2.5;\n")        // 階層間の縦幅 (1.2 -> 2.5)
	
	// 直角配線設定
	file.WriteString("  splines=ortho;\n")      
	file.WriteString("  concentrate=false;\n")  // 線をまとめない（正確にポートにつなぐため）
	
	// ノードフォント設定
	file.WriteString("  node [shape=plain fontname=\"Helvetica\" fontsize=12];\n")
	
	// エッジ設定
	file.WriteString("  edge [dir=none style=solid penwidth=1.5 color=\"#888888\"];\n") 

	// ノード書き込み
	for _, dev := range devices {
		var ports []string
		for p := range dev.Ports {
			ports = append(ports, p)
		}
		sort.Strings(ports)

		// 配色の取得
		theme := getTheme(dev.Role)

		// HTML Label
		// BORDER="0" にしつつ、枠線色はデバイスの種類に合わせる
		label := fmt.Sprintf(`<<TABLE BORDER="0" CELLBORDER="1" CELLSPACING="0" CELLPADDING="4" COLOR="%s" BGCOLOR="#FFFFFF">`, theme.Border)
		
		// ヘッダー行 (角丸はGraphvizのHTMLラベルではできないが、色で表現)
		label += fmt.Sprintf(`<TR><TD COLSPAN="%d" BGCOLOR="%s" HEIGHT="35" BORDER="0"><B><FONT COLOR="%s" POINT-SIZE="14"> %s </FONT></B></TD></TR>`, 
			len(ports), theme.Header, theme.Text, dev.Name)
		
		// ポート一覧行
		label += "<TR>"
		for _, p := range ports {
			// ★★★ ズレ解消ポイント2: ポート幅を確保 ★★★
			// WIDTH="60" などを指定し、ターゲットを大きくする
			label += fmt.Sprintf(`<TD PORT="%s" WIDTH="60" HEIGHT="26" BGCOLOR="%s" BORDER="1" COLOR="%s"><FONT COLOR="#333333" POINT-SIZE="10">%s</FONT></TD>`, 
				escape(p), theme.PortFill, theme.Border, p)
		}
		label += "</TR></TABLE>>"

		file.WriteString(fmt.Sprintf(`  "%s" [label=%s];`+"\n", dev.Name, label))
	}

	// 接続書き込み
	for _, conn := range connections {
		srcComp := ":s"
		dstComp := ":n"

		if conn.SrcLevel == conn.DstLevel {
			dstComp = ":s" 
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

// モダンで鮮やかなカラーパレット
func getTheme(role string) ThemeColor {
	switch {
	case strings.Contains(role, "router") || strings.Contains(role, "firewall") || strings.Contains(role, "onu"):
		// Vivid Pink/Red
		return ThemeColor{Header: "#FF4757", Border: "#FF4757", Text: "#FFFFFF", PortFill: "#FFF0F1"}
	case strings.Contains(role, "core"):
		// Vibrant Orange
		return ThemeColor{Header: "#FFA502", Border: "#FFA502", Text: "#FFFFFF", PortFill: "#FFF6E5"}
	case strings.Contains(role, "distribution"):
		// Teal / Turquoise
		return ThemeColor{Header: "#2ED573", Border: "#2ED573", Text: "#FFFFFF", PortFill: "#EAFAF1"}
	case strings.Contains(role, "access"):
		// Modern Blue
		return ThemeColor{Header: "#1E90FF", Border: "#1E90FF", Text: "#FFFFFF", PortFill: "#F0F8FF"}
	case strings.Contains(role, "ap"):
		// Vivid Purple
		return ThemeColor{Header: "#5352ED", Border: "#5352ED", Text: "#FFFFFF", PortFill: "#F3F3FF"}
	case strings.Contains(role, "server"):
		// Slate Gray
		return ThemeColor{Header: "#57606F", Border: "#57606F", Text: "#FFFFFF", PortFill: "#F1F2F6"}
	default:
		// Default Gray
		return ThemeColor{Header: "#747D8C", Border: "#747D8C", Text: "#FFFFFF", PortFill: "#FFFFFF"}
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
