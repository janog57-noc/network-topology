package renderer

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/janog57-noc/network-topology/internal/model"
	"github.com/janog57-noc/network-topology/internal/style"
	"github.com/janog57-noc/network-topology/internal/utils"
)

func GenerateDOT(data *model.TopologyData, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")

	// Header
	file.WriteString("digraph NetworkTopology {\n")
	file.WriteString("  bgcolor=\"#FFFFFF\";\n")
	file.WriteString("  rankdir=TB;\n")
	file.WriteString("  labelloc=\"t\";\n")
	file.WriteString("  labeljust=\"r\";\n")
	file.WriteString(fmt.Sprintf("  label=\"Network Topology - Generated at %s\";\n", timestamp))
	file.WriteString("  fontsize=60;\n")
	file.WriteString("  nodesep=2.0;\n")
	file.WriteString("  ranksep=10.0;\n")
	file.WriteString("  splines=line;\n")
	file.WriteString("  concentrate=false;\n")
	file.WriteString("  node [shape=plain fontname=\"Helvetica\" fontsize=64];\n")
	file.WriteString("  edge [dir=none style=solid penwidth=20.0];\n")

	// VLAN Legend
	file.WriteString("  VLANLegend [label=<<TABLE BORDER=\"0\" CELLBORDER=\"3\" CELLSPACING=\"0\" CELLPADDING=\"20\" BGCOLOR=\"#FFFFFF\">\n")
	file.WriteString("    <TR><TD COLSPAN=\"2\" BGCOLOR=\"#333333\" BORDER=\"0\"><B><FONT COLOR=\"#FFFFFF\" POINT-SIZE=\"80\">VLAN Legend</FONT></B></TD></TR>\n")
	vlanList := []struct {
		id   int
		name string
	}{
		{10, "VLAN 10"}, {20, "VLAN 20"}, {30, "VLAN 30"}, {40, "VLAN 40"},
	}
	for _, v := range vlanList {
		color := style.GetVlanColor(v.id)
		file.WriteString(fmt.Sprintf("    <TR><TD BGCOLOR=\"%s\" WIDTH=\"120\" HEIGHT=\"60\" BORDER=\"3\"></TD><TD ALIGN=\"LEFT\"><FONT POINT-SIZE=\"64\"><B>%s</B></FONT></TD></TR>\n", color, v.name))
	}
	file.WriteString("  </TABLE>>, shape=plaintext, rank=max];\n\n")

	// Devices
	for _, dev := range data.Devices {
		var ports []string
		for p := range dev.Ports {
			ports = append(ports, p)
		}
		sort.Strings(ports)

		theme := style.GetThemeByTag(dev.PrimaryTag)

		label := fmt.Sprintf(`<<TABLE BORDER="0" CELLBORDER="3" CELLSPACING="0" CELLPADDING="15" COLOR="%s" BGCOLOR="#FFFFFF">`, theme.Border)

		// Build port cells
		portRow := "<TR>"
		for _, p := range ports {
			vlans := dev.PortVlans[p]
			if len(vlans) == 0 {
				portRow += fmt.Sprintf(`<TD PORT="%s" BGCOLOR="%s" BORDER="3" COLOR="%s" WIDTH="100" HEIGHT="70"><FONT COLOR="#333333" POINT-SIZE="64">%s</FONT></TD>`,
					utils.Escape(p), theme.PortFill, theme.Border, p)
			} else if len(vlans) == 1 {
				portColor := style.GetVlanColor(vlans[0])
				portRow += fmt.Sprintf(`<TD PORT="%s" BGCOLOR="%s" BORDER="3" COLOR="%s" WIDTH="100" HEIGHT="70"><FONT COLOR="#333333" POINT-SIZE="64">%s</FONT></TD>`,
					utils.Escape(p), portColor, theme.Border, p)
			} else {
				portRow += fmt.Sprintf(`<TD PORT="%s" BORDER="3" COLOR="%s">`, utils.Escape(p), theme.Border)
				portRow += "<TABLE BORDER=\"0\" CELLBORDER=\"0\" CELLSPACING=\"0\" CELLPADDING=\"0\" WIDTH=\"100\" HEIGHT=\"70\">"
				portRow += fmt.Sprintf(`<TR><TD COLSPAN="%d" BGCOLOR="#FFFFFF" HEIGHT="25"><FONT COLOR="#333333" POINT-SIZE="64">%s</FONT></TD></TR>`, len(vlans), p)
				portRow += "<TR>"
				widthPercent := 100.0 / float64(len(vlans))
				for _, vlan := range vlans {
					color := style.GetVlanColor(vlan)
					portRow += fmt.Sprintf(`<TD BGCOLOR="%s" WIDTH="%.1f%%" HEIGHT="45"></TD>`, color, widthPercent)
				}
				portRow += "</TR></TABLE></TD>"
			}
		}
		portRow += "</TR>"

		// Header row
		headerRow := fmt.Sprintf(`<TR><TD COLSPAN="%d" BGCOLOR="%s" BORDER="0" HEIGHT="80" ALIGN="CENTER" VALIGN="MIDDLE"><B><FONT COLOR="%s" POINT-SIZE="96"> %s </FONT></B></TD></TR>`,
			len(ports), theme.Header, theme.Text, dev.Name)

		// Add rows in appropriate order
		if dev.FlipLabel {
			label += portRow + headerRow
		} else {
			label += headerRow + portRow
		}

		label += "</TABLE>>"

		file.WriteString(fmt.Sprintf(`  "%s" [label=%s];`+"\n", dev.Name, label))
	}

	// Connections
	for _, conn := range data.Connections {
		lineColor := style.GetLevelLineColor(conn.SrcLevel)
		// console-server タグを持つ機器が片端でもあれば紫色
		if srcDev, ok := data.Devices[conn.SrcDev]; ok && srcDev.PrimaryTag == "console-server" {
			lineColor = "#8E44AD"
		}
		if dstDev, ok := data.Devices[conn.DstDev]; ok && dstDev.PrimaryTag == "console-server" {
			lineColor = "#8E44AD"
		}

		// 接続方向の決定：上位レベルから下位レベルへ接続
		srcDir := getConnectionDirection(conn.SrcLevel, conn.DstLevel, "src")
		dstDir := getConnectionDirection(conn.SrcLevel, conn.DstLevel, "dst")

		file.WriteString(fmt.Sprintf(`  "%s":"%s"%s -> "%s":"%s"%s [color="%s", penwidth=20.0];`+"\n",
			conn.SrcDev, utils.Escape(conn.SrcPort), srcDir,
			conn.DstDev, utils.Escape(conn.DstPort), dstDir,
			lineColor))
	}

	file.WriteString("}\n")
	return nil
}

func getConnectionDirection(srcLevel, dstLevel int, connType string) string {
	// 上位レベルから下位レベルへの接続
	if srcLevel < dstLevel {
		if connType == "src" {
			return ":s" // 送信元は下に
		}
		return ":n" // 受信先は上に
	} else if srcLevel > dstLevel {
		if connType == "src" {
			return ":n" // 送信元は上に
		}
		return ":s" // 受信先は下に
	}
	// 同じレベルの場合
	return ":c" // 中央
}
