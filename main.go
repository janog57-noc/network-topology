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

// --- APIレスポンス構造体 ---

// 1. デバイス一覧取得用 (タグ情報を確実に取るため)
type DeviceResponse struct {
	Results []struct {
		Name string `json:"name"`
		Tags []struct {
			Slug string `json:"slug"`
		} `json:"tags"`
	} `json:"results"`
}

// 2. ケーブル一覧取得用
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

// 内部データ構造
type DeviceData struct {
	Name      string
	PrimaryTag string // ONU, Router, etc.
	Ports     map[string]bool
}

// カラーパレット構造体
type ThemeColor struct {
	Header   string
	Border   string
	Text     string
	PortFill string
}

func main() {
	client := &http.Client{}

	// ---------------------------------------------------------
	// 1. デバイス情報を全取得して、タグのマップを作成する
	// ---------------------------------------------------------
	deviceTagMap := make(map[string]string) // Name -> PrimaryTag

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
			// タグリストから「最も重要なタグ」を一つ選定してマップに登録
			deviceTagMap[dev.Name] = resolvePrimaryTag(dev.Tags)
		}
	}

	// ---------------------------------------------------------
	// 2. ケーブル情報を取得
	// ---------------------------------------------------------
	req, _ := http.NewRequest("GET", NetboxURL+"/api/dcim/cables/?limit=0", nil)
	req.Header.Set("Authorization", "Token "+NetboxToken)
	resp, err := client.Do(req)
	if err != nil { panic(err) }
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var cResp CableResponse
	json.Unmarshal(body, &cResp)

	// ---------------------------------------------------------
	// 3. データ整理 & DOT生成準備
	// ---------------------------------------------------------
	devices := make(map[string]*DeviceData)
	type Connection struct {
		SrcDev, SrcPort string
		DstDev, DstPort string
		SrcLevel, DstLevel int
	}
	var connections []Connection

	for _, cable := range cResp.Results {
		if len(cable.ATerminations) == 0 || len(cable.BTerminations) == 0 { continue }
		termA := cable.ATerminations[0].Object
		termB := cable.BTerminations[0].Object

		nameA := termA.Device.Name
		nameB := termB.Device.Name

		// マップからタグ情報を取得 (なければ "unknown")
		tagA := deviceTagMap[nameA]
		tagB := deviceTagMap[nameB]

		registerDevice(devices, nameA, tagA, termA.Name)
		registerDevice(devices, nameB, tagB, termB.Name)

		levelA := getLevelByTag(tagA)
		levelB := getLevelByTag(tagB)

		// 上位レベル(数値が小さい)をSrc、下位レベルをDstにする
		if levelA <= levelB {
			connections = append(connections, Connection{
				SrcDev: nameA, SrcPort: termA.Name, SrcLevel: levelA,
				DstDev: nameB, DstPort: termB.Name, DstLevel: levelB,
			})
		} else {
			connections = append(connections, Connection{
				SrcDev: nameB, SrcPort: termB.Name, SrcLevel: levelB,
				DstDev: nameA, DstPort: termA.Name, DstLevel: levelA,
			})
		}
	}

	// ---------------------------------------------------------
	// 4. DOT生成
	// ---------------------------------------------------------
	file, _ := os.Create("topology.dot")
	defer file.Close()

	file.WriteString("digraph NetworkTopology {\n")
	file.WriteString("  bgcolor=\"#FFFFFF\";\n") 
	file.WriteString("  rankdir=TB;\n")         
	file.WriteString("  nodesep=1.5;\n")        // 横間隔広め
	file.WriteString("  ranksep=2.5;\n")        // 縦間隔広め
	file.WriteString("  splines=ortho;\n")      // 直角配線
	file.WriteString("  concentrate=false;\n")
	file.WriteString("  node [shape=plain fontname=\"Helvetica\" fontsize=12];\n")
	file.WriteString("  edge [dir=none style=solid penwidth=1.5 color=\"#888888\"];\n") 

	for _, dev := range devices {
		var ports []string
		for p := range dev.Ports {
			ports = append(ports, p)
		}
		sort.Strings(ports)

		theme := getThemeByTag(dev.PrimaryTag)

		// HTML Label
		label := fmt.Sprintf(`<<TABLE BORDER="0" CELLBORDER="1" CELLSPACING="0" CELLPADDING="4" COLOR="%s" BGCOLOR="#FFFFFF">`, theme.Border)
		
		// Header
		label += fmt.Sprintf(`<TR><TD COLSPAN="%d" BGCOLOR="%s" HEIGHT="35" BORDER="0"><B><FONT COLOR="%s" POINT-SIZE="14"> %s </FONT></B></TD></TR>`, 
			len(ports), theme.Header, theme.Text, dev.Name)
		
		// Ports
		label += "<TR>"
		for _, p := range ports {
			label += fmt.Sprintf(`<TD PORT="%s" WIDTH="60" HEIGHT="26" BGCOLOR="%s" BORDER="1" COLOR="%s"><FONT COLOR="#333333" POINT-SIZE="10">%s</FONT></TD>`, 
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

		line := fmt.Sprintf(`  "%s":"%s"%s -> "%s":"%s"%s`, 
			conn.SrcDev, escape(conn.SrcPort), srcComp,
			conn.DstDev, escape(conn.DstPort), dstComp)
		file.WriteString(line + ";\n")
	}

	file.WriteString("}\n")
	fmt.Println("Generated topology.dot with Tags")
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

// タグリストから優先順位の高いものを抽出
func resolvePrimaryTag(tags []struct{ Slug string }) string {
	// 優先順位リスト (上位からマッチさせる)
	priority := []string{"onu", "router", "core-switch", "edge-switch", "server", "ap"}
	
	// APIから返ってきたタグのスラグセットを作成
	tagSet := make(map[string]bool)
	for _, t := range tags {
		tagSet[t.Slug] = true
	}

	for _, p := range priority {
		if tagSet[p] {
			return p
		}
	}
	return "other"
}

// タグに基づく階層レベル定義 (小さいほど上)
func getLevelByTag(tag string) int {
	switch tag {
	case "onu":
		return 1
	case "router":
		return 2
	case "core-switch":
		return 3
	case "edge-switch":
		return 4
	case "server":
		return 5
	case "ap":
		return 5 // Serverと同じ最下層
	default:
		return 99
	}
}

// タグに基づくカラーパレット定義
func getThemeByTag(tag string) ThemeColor {
	switch tag {
	case "onu":
		// Cyan / Aqua (Uplink)
		return ThemeColor{Header: "#00BCD4", Border: "#00BCD4", Text: "#FFFFFF", PortFill: "#E0F7FA"}
	case "router":
		// Red / Pink (Core Routing)
		return ThemeColor{Header: "#FF4757", Border: "#FF4757", Text: "#FFFFFF", PortFill: "#FFF0F1"}
	case "core-switch":
		// Orange (Core/Dist Switch)
		return ThemeColor{Header: "#FFA502", Border: "#FFA502", Text: "#FFFFFF", PortFill: "#FFF6E5"}
	case "edge-switch":
		// Blue (Access Switch)
		return ThemeColor{Header: "#1E90FF", Border: "#1E90FF", Text: "#FFFFFF", PortFill: "#F0F8FF"}
	case "server":
		// Slate Gray (End Device)
		return ThemeColor{Header: "#57606F", Border: "#57606F", Text: "#FFFFFF", PortFill: "#F1F2F6"}
	case "ap":
		// Purple (Wireless)
		return ThemeColor{Header: "#5352ED", Border: "#5352ED", Text: "#FFFFFF", PortFill: "#F3F3FF"}
	default:
		// Gray
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
