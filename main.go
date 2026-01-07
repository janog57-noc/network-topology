package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"sort"
	"strings"
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
	Line     string // 線の色を追加
}

func main() {
	client := &http.Client{}

	// 1. デバイス情報取得 (タグマッピング用)
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
		DstTag             string // 線の色を決めるために接続先のタグを保持
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

		// 上位(小さい数値) -> 下位(大きい数値) の向きに統一
		if levelA <= levelB {
			connections = append(connections, Connection{
				SrcDev: nameA, SrcPort: termA.Name, SrcLevel: levelA,
				DstDev: nameB, DstPort: termB.Name, DstLevel: levelB,
				DstTag: tagB, // 接続先のタグを保存
			})
		} else {
			connections = append(connections, Connection{
				SrcDev: nameB, SrcPort: termB.Name, SrcLevel: levelB,
				DstDev: nameA, DstPort: termA.Name, DstLevel: levelA,
				DstTag: tagA, // 接続先のタグを保存
			})
		}
	}

	// 4. DOT生成
	file, _ := os.Create("topology.dot")
	defer file.Close()

	file.WriteString("digraph NetworkTopology {\n")
	file.WriteString("  bgcolor=\"#FFFFFF\";\n")
	file.WriteString("  rankdir=TB;\n")
	
	// ★サイズ調整: 間隔をさらに広くして重なりを防ぐ
	file.WriteString("  nodesep=2.0;\n") 
	file.WriteString("  ranksep=3.5;\n")
	
	file.WriteString("  splines=ortho;\n")
	file.WriteString("  concentrate=false;\n")
	
	// ★サイズ調整: フォントサイズを大きく
	file.WriteString("  node [shape=plain fontname=\"Helvetica\" fontsize=16];\n")
	// ★サイズ調整: 線を太く
	file.WriteString("  edge [dir=none style=solid penwidth=3.0];\n")

	for _, dev := range devices {
		var ports []string
		for p := range dev.Ports {
			ports = append(ports, p)
		}
		sort.Strings(ports)

		theme := getThemeByTag(dev.PrimaryTag)
		
		// ポートが多い場合は折り返す (最大列数を設定)
		const maxCols = 12
		portRows := splitPorts(ports, maxCols)

		// HTML Label
		label := fmt.Sprintf(`<<TABLE BORDER="0" CELLBORDER="1" CELLSPACING="0" CELLPADDING="8" COLOR="%s" BGCOLOR="#FFFFFF">`, theme.Border)
		
		// ヘッダー: 文字サイズ大きく (POINT-SIZE="20")
		// COLSPANは全列分確保
		totalCols := len(portRows[0]) // 1行目の列数を基準に
		if len(portRows) > 1 && len(portRows[0]) < maxCols {
			// 全行の中で最大列数に合わせる処理が必要だが、簡易的にmaxColsを使うか、計算する
			// ここでは簡易的に 1行目の長さ * 行数 ではなく、単純に maxCols に合わせる
			// 正確には「最も列数が多い行」に合わせるべき
			maxLen := 0
			for _, r := range portRows {
				if len(r) > maxLen { maxLen = len(r) }
			}
			totalCols = maxLen
		}
		
		label += fmt.Sprintf(`<TR><TD COLSPAN="%d" BGCOLOR="%s" HEIGHT="50" BORDER="0"><B><FONT COLOR="%s" POINT-SIZE="20"> %s </FONT></B></TD></TR>`, 
			totalCols, theme.Header, theme.Text, dev.Name)
		
		// ポート行の生成
		for _, row := range portRows {
			label += "<TR>"
			for _, p := range row {
				// ★サイズ調整: ポートの幅と高さを大きく (WIDTH="100", HEIGHT="40")
				// ★文字サイズ: POINT-SIZE="16"
				label += fmt.Sprintf(`<TD PORT="%s" WIDTH="100" HEIGHT="40" BGCOLOR="%s" BORDER="1" COLOR="%s"><FONT COLOR="#333333" POINT-SIZE="16">%s</FONT></TD>`, 
					escape(p), theme.PortFill, theme.Border, p)
			}
			// 行の列数が足りない場合、空セルで埋める（レイアウト崩れ防止）
			for i := len(row); i < totalCols; i++ {
				label += `<TD BORDER="0"></TD>`
			}
			label += "</TR>"
		}
		label += "</TABLE>>"

		file.WriteString(fmt.Sprintf(`  "%s" [label=%s];`+"\n", dev.Name, label))
	}

	for _, conn := range connections {
		srcComp := ":s"
		dstComp := ":n"
		if conn.SrcLevel == conn.DstLevel {
			dstComp = ":s"
		}
		
		// ★色分け: 接続先のデバイスタイプに合わせて線の色を変える
		dstTheme := getThemeByTag(conn.DstTag)
		lineColor := dstTheme.Line

		line := fmt.Sprintf(`  "%s":"%s"%s -> "%s":"%s"%s [color="%s"];`, 
			conn.SrcDev, escape(conn.SrcPort), srcComp,
			conn.DstDev, escape(conn.DstPort), dstComp,
			lineColor) // 色を指定
			
		file.WriteString(line + "\n")
	}

	file.WriteString("}\n")
	fmt.Println("Generated topology.dot with Enhanced Visibility")
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

// ポートリストを指定された列数で分割する関数
func splitPorts(ports []string, maxCols int) [][]string {
	var rows [][]string
	length := len(ports)
	
	// ポート数が少ないなら1行で返す
	if length <= maxCols {
		return [][]string{ports}
	}
	
	// 分割計算
	for i := 0; i < length; i += maxCols {
		end := i + maxCols
		if end > length {
			end = length
		}
		rows = append(rows, ports[i:end])
	}
	return rows
}

// カラーパレット (線色 Line を追加)
func getThemeByTag(tag string) ThemeColor {
	switch tag {
	case "onu":
		return ThemeColor{Header: "#00BCD4", Border: "#00BCD4", Text: "#FFFFFF", PortFill: "#E0F7FA", Line: "#00BCD4"}
	case "router":
		return ThemeColor{Header: "#FF4757", Border: "#FF4757", Text: "#FFFFFF", PortFill: "#FFF0F1", Line: "#FF4757"}
	case "core-switch":
		return ThemeColor{Header: "#FFA502", Border: "#FFA502", Text: "#FFFFFF", PortFill: "#FFF6E5", Line: "#FFA502"}
	case "edge-switch":
		// 青
		return ThemeColor{Header: "#1E90FF", Border: "#1E90FF", Text: "#FFFFFF", PortFill: "#F0F8FF", Line: "#1E90FF"}
	case "server":
		// グレー
		return ThemeColor{Header: "#57606F", Border: "#57606F", Text: "#FFFFFF", PortFill: "#F1F2F6", Line: "#747D8C"}
	case "ap":
		// 紫
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
