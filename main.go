package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time" // 時刻表示用
)

var (
	NetboxURL   = os.Getenv("NETBOX_URL")
	NetboxToken = os.Getenv("NETBOX_TOKEN")
)

// --- 構造体定義 ---

type NetboxTag struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

type DeviceResponse struct {
	Results []struct {
		Name string      `json:"name"`
		Tags []NetboxTag `json:"tags"`
	} `json:"results"`
}

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
		} `json:"device"`
	} `json:"object"`
}

type DeviceData struct {
	Name       string
	PrimaryTag string
	Ports      map[string]bool
}

type ThemeColor struct {
	Header   string
	Border   string
	Text     string
	PortFill string
	Line     string
}

func main() {
	client := &http.Client{}

	// 1. デバイス情報取得
	deviceTagMap := make(map[string]string)
	{
		req, _ := http.NewRequest("GET", NetboxURL+"/api/dcim/devices/?limit=0", nil)
		req.Header.Set("Authorization", "Token "+NetboxToken)
		resp, err := client.Do(req)
		if err != nil { panic(err) }
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)

		var dResp DeviceResponse
		json.Unmarshal(body, &dResp)
		for _, dev := range dResp.Results {
			deviceTagMap[dev.Name] = resolvePrimaryTag(dev.Tags)
		}
	}

	// 2. ケーブル情報取得
	req, _ := http.NewRequest("GET", NetboxURL+"/api/dcim/cables/?limit=0", nil)
	req.Header.Set("Authorization", "Token "+NetboxToken)
	resp, err := client.Do(req)
	if err != nil { panic(err) }
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var cResp CableResponse
	json.Unmarshal(body, &cResp)

	// 3. データ整理
	devices := make(map[string]*DeviceData)
	type Connection struct {
		SrcDev, SrcPort    string
		DstDev, DstPort    string
		SrcLevel, DstLevel int
		DstTag             string
	}
	var connections []Connection

	for _, cable := range cResp.Results {
		if len(cable.ATerminations) == 0 || len(cable.BTerminations) == 0 { continue }
		termA := cable.ATerminations[0].Object
		termB := cable.BTerminations[0].Object

		nameA := termA.Device.Name
		nameB := termB.Device.Name
		tagA := deviceTagMap[nameA]
		tagB := deviceTagMap[nameB]
		if tagA == "" { tagA = "other" }
		if tagB == "" { tagB = "other" }

		registerDevice(devices, nameA, tagA, termA.Name)
		registerDevice(devices, nameB, tagB, termB.Name)

		levelA := getLevelByTag(tagA)
		levelB := getLevelByTag(tagB)

		if levelA <= levelB {
			connections = append(connections, Connection{
				SrcDev: nameA, SrcPort: termA.Name, SrcLevel: levelA,
				DstDev: nameB, DstPort: termB.Name, DstLevel: levelB,
				DstTag: tagB,
			})
		} else {
			connections = append(connections, Connection{
				SrcDev: nameB, SrcPort: termB.Name, SrcLevel: levelB,
				DstDev: nameA, DstPort: termA.Name, DstLevel: levelA,
				DstTag: tagA,
			})
		}
	}

	// 4. DOT生成
	file, _ := os.Create("topology.dot")
	defer file.Close()

	// タイムスタンプ生成
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	file.WriteString("digraph NetworkTopology {\n")
	file.WriteString("  bgcolor=\"#FFFFFF\";\n")
	file.WriteString("  rankdir=TB;\n")
	file.WriteString("  labelloc=\"t\";\n") // ラベルを上部に
	file.WriteString(fmt.Sprintf("  label=\"Network Topology - Generated at %s\";\n", timestamp))
	file.WriteString("  fontsize=30;\n")
	
	// ★サイズ調整: これでもかというほど広げる
	file.WriteString("  nodesep=4.0;\n") 
	file.WriteString("  ranksep=8.0;\n") 
	
	file.WriteString("  splines=ortho;\n")
	file.WriteString("  concentrate=false;\n")
	
	// フォント指定
	file.WriteString("  node [shape=plain fontname=\"Helvetica\" fontsize=24];\n")
	file.WriteString("  edge [dir=none style=solid penwidth=4.0];\n")

	for _, dev := range devices {
		var ports []string
		for p := range dev.Ports {
			ports = append(ports, p)
		}
		sort.Strings(ports)

		theme := getThemeByTag(dev.PrimaryTag)
		
		// ★変更点: 折り返しを廃止し、横一列にする (崩れ防止)
		// ポートが多いスイッチは横に長くなるが、接続線は正確になる
		
		// CELLPADDING="20" で箱を大きくする（サイズ指定より確実）
		label := fmt.Sprintf(`<<TABLE BORDER="0" CELLBORDER="1" CELLSPACING="0" CELLPADDING="20" COLOR="%s" BGCOLOR="#FFFFFF">`, theme.Border)
		
		// ヘッダー: 文字サイズ 40
		label += fmt.Sprintf(`<TR><TD COLSPAN="%d" BGCOLOR="%s" BORDER="0"><B><FONT COLOR="%s" POINT-SIZE="40"> %s </FONT></B></TD></TR>`, 
			len(ports), theme.Header, theme.Text, dev.Name)
		
		label += "<TR>"
		for _, p := range ports {
			// ポートセル: 文字サイズ 24
			// FIXEDSIZEを使わず、パディングで自然な大きさを確保
			label += fmt.Sprintf(`<TD PORT="%s" BGCOLOR="%s" BORDER="1" COLOR="%s"><FONT COLOR="#333333" POINT-SIZE="24">%s</FONT></TD>`, 
				escape(p), theme.PortFill, theme.Border, p)
		}
		label += "</TR></TABLE>>"

		file.WriteString(fmt.Sprintf(`  "%s" [label=%s];`+"\n", dev.Name, label))
	}

	for _, conn := range connections {
		srcComp := ":s"
		dstComp := ":n"
		if conn.SrcLevel == conn.DstLevel {
			dstComp = ":s"
		}
		
		dstTheme := getThemeByTag(conn.DstTag)
		lineColor := dstTheme.Line

		line := fmt.Sprintf(`  "%s":"%s"%s -> "%s":"%s"%s [color="%s"];`, 
			conn.SrcDev, escape(conn.SrcPort), srcComp,
			conn.DstDev, escape(conn.DstPort), dstComp,
			lineColor)
			
		file.WriteString(line + "\n")
	}

	file.WriteString("}\n")
	fmt.Println("Generated topology.dot with Timestamp and Layout Fixes")
}

// ---------------------------------------------------------
// Helper Functions
// ---------------------------------------------------------

func registerDevice(devices map[string]*DeviceData, name, tag, port string) {
	if _, ok := devices[name]; !ok {
		devices[name] = &DeviceData{Name: name, PrimaryTag: tag, Ports: make(map[string]bool)}
	}
	devices[name].Ports[port] = true
}

func resolvePrimaryTag(tags []NetboxTag) string {
	priority := []string{"onu", "router", "core-switch", "edge-switch", "server", "ap"}
	tagSet := make(map[string]bool)
	for _, t := range tags {
		tagSet[t.Slug] = true
	}
	for _, p := range priority {
		if tagSet[p] { return p }
	}
	if len(tags) > 0 { return tags[0].Slug }
	return "other"
}

func getLevelByTag(tag string) int {
	switch tag {
	case "onu": return 1
	case "router": return 2
	case "core-switch": return 3
	case "edge-switch": return 4
	case "server": return 5
	case "ap": return 5
	default: return 99
	}
}

func getThemeByTag(tag string) ThemeColor {
	switch tag {
	case "onu":
		return ThemeColor{Header: "#00BCD4", Border: "#00BCD4", Text: "#FFFFFF", PortFill: "#E0F7FA", Line: "#00BCD4"}
	case "router":
		return ThemeColor{Header: "#FF4757", Border: "#FF4757", Text: "#FFFFFF", PortFill: "#FFF0F1", Line: "#FF4757"}
	case "core-switch":
		return ThemeColor{Header: "#FFA502", Border: "#FFA502", Text: "#FFFFFF", PortFill: "#FFF6E5", Line: "#FFA502"}
	case "edge-switch":
		return ThemeColor{Header: "#1E90FF", Border: "#1E90FF", Text: "#FFFFFF", PortFill: "#F0F8FF", Line: "#1E90FF"}
	case "server":
		return ThemeColor{Header: "#57606F", Border: "#57606F", Text: "#FFFFFF", PortFill: "#F1F2F6", Line: "#747D8C"}
	case "ap":
		return ThemeColor{Header: "#5352ED", Border: "#5352ED", Text: "#FFFFFF", PortFill: "#F3F3FF", Line: "#5352ED"}
	default:
		return ThemeColor{Header: "#747D8C", Border: "#747D8C", Text: "#FFFFFF", PortFill: "#FFFFFF", Line: "#999999"}
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
