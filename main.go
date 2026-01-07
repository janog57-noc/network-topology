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

// --- 構造体定義 (型エラーを防ぐため独立させる) ---

// NetboxTag : タグ情報の共通構造体
type NetboxTag struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// DeviceResponse : デバイス一覧取得用
type DeviceResponse struct {
	Results []struct {
		Name string      `json:"name"`
		Tags []NetboxTag `json:"tags"`
	} `json:"results"`
}

// CableResponse : ケーブル一覧取得用
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
	Name       string
	PrimaryTag string
	Ports      map[string]bool
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
		// limit=0 で全件取得
		req, _ := http.NewRequest("GET", NetboxURL+"/api/dcim/devices/?limit=0", nil)
		req.Header.Set("Authorization", "Token "+NetboxToken)
		resp, err := client.Do(req)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)

		var dResp DeviceResponse
		if err := json.Unmarshal(body, &dResp); err != nil {
			panic(fmt.Sprintf("Failed to parse devices: %v", err))
		}

		for _, dev := range dResp.Results {
			deviceTagMap[dev.Name] = resolvePrimaryTag(dev.Tags)
		}
	}

	// ---------------------------------------------------------
	// 2. ケーブル情報を取得
	// ---------------------------------------------------------
	req, _ := http.NewRequest("GET", NetboxURL+"/api/dcim/cables/?limit=0", nil)
	req.Header.Set("Authorization", "Token "+NetboxToken)
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var cResp CableResponse
	if err := json.Unmarshal(body, &cResp); err != nil {
		panic(fmt.Sprintf("Failed to parse cables: %v", err))
	}

	// ---------------------------------------------------------
	// 3. データ整理
	// ---------------------------------------------------------
	devices := make(map[string]*DeviceData)
	type Connection struct {
		SrcDev, SrcPort    string
		DstDev, DstPort    string
		SrcLevel, DstLevel int
	}
	var connections []Connection

	for _, cable := range cResp.Results {
		if len(cable.ATerminations) == 0 || len(cable.BTerminations) == 0 {
			continue
		}
		termA := cable.ATerminations[0].Object
		termB := cable.BTerminations[0].Object

		nameA := termA.Device.Name
		nameB := termB.Device.Name

		// タグ情報をマップから引く
		tagA := deviceTagMap[nameA]
		tagB := deviceTagMap[nameB]
		// タグがない場合はデフォルト処理
		if tagA == "" {
			tagA = "other"
		}
		if tagB == "" {
			tagB = "other"
		}

		registerDevice(devices, nameA, tagA, termA.Name)
		registerDevice(devices, nameB, tagB, termB.Name)

		levelA := getLevelByTag(tagA)
		levelB := getLevelByTag(tagB)

		// 上位レベル(数値が小さい)をSrc、下位レベルをDstにする
		// これにより配線が上から下へ流れる
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

	// Graphviz設定
	file.WriteString("digraph NetworkTopology {\n")
	file.WriteString("  bgcolor=\"#FFFFFF\";\n")
	file.WriteString("  rankdir=TB;\n")
	file.WriteString("  nodesep=1.5;\n")   // 横間隔広め（ズレ防止）
	file.WriteString("  ranksep=2.5;\n")   // 縦間隔広め（ズレ防止）
	file.WriteString("  splines=ortho;\n") // 直角配線
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
		// 同階層なら下同士でつなぐ（ループ回避）
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

// resolvePrimaryTag: タグリストを受け取り、優先順位に基づいてメインの役割を返す
func resolvePrimaryTag(tags []NetboxTag) string {
	// 優先順位リスト
	priority := []string{"onu", "router", "core-switch", "edge-switch", "server", "ap"}

	tagSet := make(map[string]bool)
	for _, t := range tags {
		tagSet[t.Slug] = true
	}

	for _, p := range priority {
		if tagSet[p] {
			return p
		}
	}
	// マッチしなかった場合は最初に見つかったタグ、それもなければ "other"
	if len(tags) > 0 {
		return tags[0].Slug
	}
	return "other"
}

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
		return 5
	default:
		return 99
	}
}

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
		// Gray (Unknown)
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
