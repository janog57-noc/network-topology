package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
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

type InterfaceResponse struct {
	Results []struct {
		Name   string `json:"name"`
		Device struct {
			Name string `json:"name"`
		} `json:"device"`
		UntaggedVlan struct {
			Vid  int    `json:"vid"`
			Name string `json:"name"`
		} `json:"untagged_vlan"`
		TaggedVlans []struct {
			Vid  int    `json:"vid"`
			Name string `json:"name"`
		} `json:"tagged_vlans"`
	} `json:"results"`
}

type DeviceData struct {
	Name       string
	PrimaryTag string
	Ports      map[string]bool
	PortVlans  map[string][]int // port -> VLAN IDs
}

type Connection struct {
	SrcDev, SrcPort    string
	DstDev, DstPort    string
	SrcLevel, DstLevel int
	DstTag             string
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
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)

		var dResp DeviceResponse
		json.Unmarshal(body, &dResp)
		for _, dev := range dResp.Results {
			deviceTagMap[dev.Name] = resolvePrimaryTag(dev.Tags)
		}
	}

	// 2. インターフェイス情報とVLANを取得
	portVlanMap := make(map[string]map[string][]int) // device -> port -> vlan IDs
	{
		req, _ := http.NewRequest("GET", NetboxURL+"/api/dcim/interfaces/?limit=0", nil)
		req.Header.Set("Authorization", "Token "+NetboxToken)
		resp, err := client.Do(req)
		if err == nil {
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			var iResp InterfaceResponse
			json.Unmarshal(body, &iResp)
			for _, iface := range iResp.Results {
				devName := iface.Device.Name
				portName := iface.Name
				if portVlanMap[devName] == nil {
					portVlanMap[devName] = make(map[string][]int)
				}
				var vlans []int
				if iface.UntaggedVlan.Vid > 0 {
					vlans = append(vlans, iface.UntaggedVlan.Vid)
				}
				for _, tv := range iface.TaggedVlans {
					vlans = append(vlans, tv.Vid)
				}
				portVlanMap[devName][portName] = vlans
			}
		}
	}

	// 3. ケーブル情報を取得
	req, _ := http.NewRequest("GET", NetboxURL+"/api/dcim/cables/?limit=0", nil)
	req.Header.Set("Authorization", "Token "+NetboxToken)
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var cResp CableResponse
	json.Unmarshal(body, &cResp)

	// 3. データ整理
	devices := make(map[string]*DeviceData)
	var connections []Connection

	for _, cable := range cResp.Results {
		if len(cable.ATerminations) == 0 || len(cable.BTerminations) == 0 {
			continue
		}
		termA := cable.ATerminations[0].Object
		termB := cable.BTerminations[0].Object

		nameA := termA.Device.Name
		nameB := termB.Device.Name
		tagA := deviceTagMap[nameA]
		tagB := deviceTagMap[nameB]
		if tagA == "" {
			tagA = "other"
		}
		if tagB == "" {
			tagB = "other"
		}

		registerDevice(devices, nameA, tagA, termA.Name, portVlanMap[nameA][termA.Name])
		registerDevice(devices, nameB, tagB, termB.Name, portVlanMap[nameB][termB.Name])

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
	file.WriteString("  labelloc=\"t\";\n")  // ラベルを上部に
	file.WriteString("  labeljust=\"r\";\n") // 右寄せ
	file.WriteString(fmt.Sprintf("  label=\"Network Topology - Generated at %s\";\n", timestamp))
	file.WriteString("  fontsize=60;\n") // フォントサイズを大きく

	file.WriteString("  nodesep=2.0;\n")
	file.WriteString("  ranksep=3.0;\n")

	file.WriteString("  splines=spline;\n") // 曲線で見やすく
	file.WriteString("  concentrate=false;\n")

	// フォント指定: 文字を大きく
	file.WriteString("  node [shape=plain fontname=\"Helvetica\" fontsize=64];\n")
	file.WriteString("  edge [dir=none style=solid penwidth=12.0];\n")

	// VLAN凡例を追加
	file.WriteString("  VLANLegend [label=<<TABLE BORDER=\"0\" CELLBORDER=\"3\" CELLSPACING=\"0\" CELLPADDING=\"20\" BGCOLOR=\"#FFFFFF\">\n")
	file.WriteString("    <TR><TD COLSPAN=\"2\" BGCOLOR=\"#333333\" BORDER=\"0\"><B><FONT COLOR=\"#FFFFFF\" POINT-SIZE=\"80\">VLAN Legend</FONT></B></TD></TR>\n")
	vlanList := []struct {
		id   int
		name string
	}{
		{10, "VLAN 10"}, {20, "VLAN 20"}, {30, "VLAN 30"}, {40, "VLAN 40"},
	}
	for _, v := range vlanList {
		color := getVlanColor(v.id)
		file.WriteString(fmt.Sprintf("    <TR><TD BGCOLOR=\"%s\" WIDTH=\"120\" HEIGHT=\"60\" BORDER=\"3\"></TD><TD ALIGN=\"LEFT\"><FONT POINT-SIZE=\"64\"><B>%s</B></FONT></TD></TR>\n", color, v.name))
	}
	file.WriteString("  </TABLE>>, shape=plaintext, rank=max];\n")
	file.WriteString("\n")

	for _, dev := range devices {
		var ports []string
		for p := range dev.Ports {
			ports = append(ports, p)
		}
		sort.Strings(ports)

		theme := getThemeByTag(dev.PrimaryTag)

		// すべて1段レイアウトに統一（ポート接続位置の精度を優先）
		label := fmt.Sprintf(`<<TABLE BORDER="0" CELLBORDER="3" CELLSPACING="0" CELLPADDING="15" COLOR="%s" BGCOLOR="#FFFFFF">`, theme.Border)

		// ヘッダー: 文字サイズ 96
		label += fmt.Sprintf(`<TR><TD COLSPAN="%d" BGCOLOR="%s" BORDER="0" HEIGHT="80"><B><FONT COLOR="%s" POINT-SIZE="96"> %s </FONT></B></TD></TR>`,
			len(ports), theme.Header, theme.Text, dev.Name)

		// 1段レイアウト
		label += "<TR>"
		for _, p := range ports {
			// VLAN情報に基づいてポートの色を決定
			vlans := dev.PortVlans[p]
			if len(vlans) == 0 {
				// VLANなし：デフォルト色
				label += fmt.Sprintf(`<TD PORT="%s" BGCOLOR="%s" BORDER="3" COLOR="%s" WIDTH="100" HEIGHT="70"><FONT COLOR="#333333" POINT-SIZE="64">%s</FONT></TD>`,
					escape(p), theme.PortFill, theme.Border, p)
			} else if len(vlans) == 1 {
				// 単一VLAN：そのVLANの色
				portColor := getVlanColor(vlans[0])
				label += fmt.Sprintf(`<TD PORT="%s" BGCOLOR="%s" BORDER="3" COLOR="%s" WIDTH="100" HEIGHT="70"><FONT COLOR="#333333" POINT-SIZE="64">%s</FONT></TD>`,
					escape(p), portColor, theme.Border, p)
			} else {
				// 複数VLAN：テーブルで色分割
				label += fmt.Sprintf(`<TD PORT="%s" BORDER="3" COLOR="%s">`, escape(p), theme.Border)
				label += "<TABLE BORDER=\"0\" CELLBORDER=\"0\" CELLSPACING=\"0\" CELLPADDING=\"0\" WIDTH=\"100\" HEIGHT=\"70\">"
				// ポート名を表示する行
				label += fmt.Sprintf(`<TR><TD COLSPAN="%d" BGCOLOR="#FFFFFF" HEIGHT="25"><FONT COLOR="#333333" POINT-SIZE="64">%s</FONT></TD></TR>`, len(vlans), p)
				// 色分割の行
				label += "<TR>"
				for _, vlan := range vlans {
					color := getVlanColor(vlan)
					label += fmt.Sprintf(`<TD BGCOLOR="%s" WIDTH="%d" HEIGHT="45"></TD>`, color, 100/len(vlans))
				}
				label += "</TR></TABLE></TD>"
			}
		}
		label += "</TR>"
		label += "</TABLE>>"

		file.WriteString(fmt.Sprintf(`  "%s" [label=%s];`+"\n", dev.Name, label))
	}

	for _, conn := range connections {
		// ポートの中央から必ず線を出す
		srcComp := ":c"
		dstComp := ":c"

		// ソースポートのVLAN情報に基づいて線の色を決定
		lineColor := "#999999" // デフォルトはグレー
		if srcDev, ok := devices[conn.SrcDev]; ok {
			if vlans, ok := srcDev.PortVlans[conn.SrcPort]; ok && len(vlans) > 0 {
				// 最初のVLANの色を使用
				lineColor = getVlanColor(vlans[0])
			}
		}

		// VLAN色で線を描画
		line := fmt.Sprintf(`  "%s":"%s"%s -> "%s":"%s"%s [color="%s", penwidth=8.0];`,
			conn.SrcDev, escape(conn.SrcPort), srcComp,
			conn.DstDev, escape(conn.DstPort), dstComp,
			lineColor)

		file.WriteString(line + "\n")
	}

	file.WriteString("}\n")
	fmt.Println("Generated topology.dot with Timestamp and Layout Fixes")

	// 5. Graphvizで画像生成
	if err := generateImages(); err != nil {
		fmt.Printf("Warning: Failed to generate images: %v\n", err)
		fmt.Println("You can manually generate with: dot -Tsvg topology.dot -o topology.svg")
	} else {
		fmt.Println("Generated topology.svg and topology.png")

		// 6. インタラクティブHTMLを生成
		if err := generateInteractiveHTML(connections); err != nil {
			fmt.Printf("Warning: Failed to generate interactive HTML: %v\n", err)
		} else {
			fmt.Println("Generated index.html with interactive features")
		}
	}
}

// ---------------------------------------------------------
// Helper Functions
// ---------------------------------------------------------

func registerDevice(devices map[string]*DeviceData, name, tag, port string, vlans []int) {
	if _, ok := devices[name]; !ok {
		devices[name] = &DeviceData{Name: name, PrimaryTag: tag, Ports: make(map[string]bool), PortVlans: make(map[string][]int)}
	}
	devices[name].Ports[port] = true
	devices[name].PortVlans[port] = vlans
}

func resolvePrimaryTag(tags []NetboxTag) string {
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

// getVlanColor returns a color based on VLAN ID
func getVlanColor(vlanID int) string {
	vlanColors := map[int]string{
		10: "#FF6B6B", // 明るい赤
		20: "#4DABF7", // 明るい青
		30: "#51CF66", // 明るい緑
		40: "#FFD43B", // 明るい黄色
	}

	if color, ok := vlanColors[vlanID]; ok {
		return color
	}

	// デフォルトは白
	return "#FFFFFF"
}

// generateInteractiveHTML creates an HTML file with interactive hover effects
func generateInteractiveHTML(connections []Connection) error {
	// 接続情報をJSONに変換
	type ConnJSON struct {
		SrcDev  string `json:"src_dev"`
		SrcPort string `json:"src_port"`
		DstDev  string `json:"dst_dev"`
		DstPort string `json:"dst_port"`
	}
	var connList []ConnJSON
	for _, conn := range connections {
		connList = append(connList, ConnJSON{
			SrcDev:  conn.SrcDev,
			SrcPort: escape(conn.SrcPort),
			DstDev:  conn.DstDev,
			DstPort: escape(conn.DstPort),
		})
	}
	connJSON, _ := json.Marshal(connList)

	// HTMLテンプレート（既存のindex.htmlと同じ構造に合わせる）
	htmlContent := fmt.Sprintf(`<!DOCTYPE html>
<html>

<head>
    <title>Physical Network Topology</title>
    <script src="https://cdn.jsdelivr.net/npm/svg-pan-zoom@3.6.1/dist/svg-pan-zoom.min.js"></script>
    <style>
        body,
        html {
            margin: 0;
            padding: 0;
            width: 100%%;
            height: 100%%;
            overflow: hidden;
        }

        #svg-container {
            width: 100%%;
            height: 100%%;
            border: 1px solid #ccc;
        }
        
        /* インタラクティブ機能用のスタイル */
        .edge-dimmed {
            opacity: 0.15 !important;
        }
        .edge-highlight {
            opacity: 1 !important;
            filter: drop-shadow(0 0 4px currentColor);
        }
        .port-highlight {
            filter: drop-shadow(0 0 8px #FFD700) brightness(1.2);
        }
    </style>
</head>

<body>
    <div id="svg-container">
        <object id="demo-tiger" type="image/svg+xml" data="topology.svg"
            style="width: 100%%; height: 100%%; border:1px solid black;"></object>
    </div>

    <script>
        const connections = %s;
        
        // SVGが読み込まれたら Pan-Zoom 機能とインタラクティブ機能を有効化
        window.onload = function () {
            var svgObject = document.getElementById('demo-tiger');
            svgObject.addEventListener('load', function () {
                // Pan-Zoom機能を有効化
                svgPanZoom(svgObject, {
                    zoomEnabled: true,
                    controlIconsEnabled: true,
                    fit: true,
                    center: true
                });
                
                // インタラクティブ機能を追加
                try {
                    const svgDoc = svgObject.contentDocument;
                    if (!svgDoc) return;
                    
                    const svg = svgDoc.querySelector('svg');
                    if (!svg) return;
                    
                    const edges = svg.querySelectorAll('g.edge');
                    
                    // 各エッジにホバーイベントを設定
                    edges.forEach(edge => {
                        const title = edge.querySelector('title');
                        if (!title) return;
                        
                        const titleText = title.textContent.trim();
                        const match = titleText.match(/^(.+?)->(.+?)$/);
                        if (!match) return;
                        
                        const [, srcNode, dstNode] = match;
                        
                        // 接続情報を検索
                        const conn = connections.find(c => 
                            (c.src_dev === srcNode.trim() || srcNode.includes(c.src_dev)) &&
                            (c.dst_dev === dstNode.trim() || dstNode.includes(c.dst_dev))
                        );
                        
                        if (!conn) return;
                        
                        edge.style.cursor = 'pointer';
                        
                        edge.addEventListener('mouseenter', () => {
                            // すべてのエッジを薄くする
                            edges.forEach(e => e.classList.add('edge-dimmed'));
                            
                            // 現在のエッジを強調
                            edge.classList.remove('edge-dimmed');
                            edge.classList.add('edge-highlight');
                            
                            // 接続されているポートを強調
                            const srcPortId = conn.src_dev + ':' + conn.src_port;
                            const dstPortId = conn.dst_dev + ':' + conn.dst_port;
                            
                            svg.querySelectorAll('[id]').forEach(el => {
                                const id = el.getAttribute('id');
                                if (id === srcPortId || id === dstPortId) {
                                    el.classList.add('port-highlight');
                                }
                            });
                        });
                        
                        edge.addEventListener('mouseleave', () => {
                            // すべての強調を解除
                            edges.forEach(e => {
                                e.classList.remove('edge-dimmed');
                                e.classList.remove('edge-highlight');
                            });
                            svg.querySelectorAll('.port-highlight').forEach(el => {
                                el.classList.remove('port-highlight');
                            });
                        });
                    });
                } catch (e) {
                    console.error('Failed to add interactive features:', e);
                }
            });
        };
    </script>
</body>

</html>`, string(connJSON))

	// HTMLファイルを書き込み（index.htmlに上書き）
	return os.WriteFile("index.html", []byte(htmlContent), 0644)
}

// generateImages generates SVG and PNG from the DOT file using Graphviz
func generateImages() error {
	ctx := context.Background()

	// Check if dot command is available
	if _, err := exec.LookPath("dot"); err != nil {
		return fmt.Errorf("graphviz (dot) not found - install with: brew install graphviz")
	}

	// Generate SVG
	cmdSVG := exec.CommandContext(ctx, "dot", "-Tsvg", "topology.dot", "-o", "topology.svg")
	if err := cmdSVG.Run(); err != nil {
		return fmt.Errorf("failed to generate SVG: %w", err)
	}

	// Generate PNG
	cmdPNG := exec.CommandContext(ctx, "dot", "-Tpng", "topology.dot", "-o", "topology.png")
	if err := cmdPNG.Run(); err != nil {
		return fmt.Errorf("failed to generate PNG: %w", err)
	}

	return nil
}
