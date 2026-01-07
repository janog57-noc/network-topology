package style

type ThemeColor struct {
	Header   string
	Border   string
	Text     string
	PortFill string
	Line     string
}

func GetThemeByTag(tag string) ThemeColor {
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
	case "console-server":
		return ThemeColor{Header: "#8e44ad", Border: "#8e44ad", Text: "#FFFFFF", PortFill: "#F5E6FF", Line: "#8e44ad"}
	case "ocx":
		return ThemeColor{Header: "#00D9FF", Border: "#00D9FF", Text: "#000000", PortFill: "#E0FFFF", Line: "#00D9FF"}
	default:
		return ThemeColor{Header: "#747D8C", Border: "#747D8C", Text: "#FFFFFF", PortFill: "#FFFFFF", Line: "#999999"}
	}
}

func GetVlanColor(vlanID int) string {
	vlanColors := map[int]string{
		10:  "#FF6B6B", // 明るい赤
		20:  "#4DABF7", // 明るい青
		30:  "#51CF66", // 明るい緑
		40:  "#FFD43B", // 明るい黄色
		100: "#6C5CE7", // インディゴ（VLAN 100）
	}
	if color, ok := vlanColors[vlanID]; ok {
		return color
	}
	return "#FFFFFF"
}

func GetLevelLineColor(level int) string {
	switch level {
	case 0:
		return "#f639a4ff" // OCX
	case 1:
		return "#00BCD4" // ONU
	case 2:
		return "#FF4757" // Router
	case 3:
		return "#d78c00ff" // Core Switch
	case 4:
		return "#184cdaff" // Edge Switch
	case 5:
		return "#57606F" // Server/AP
	case 6:
		return "#8e44ad" // Console Server
	default:
		return "#999999" // Other
	}
}
func GetCableTypeColor(cableType string) string {
	cableColors := map[string]string{
		"mmf-om3": "#00BCD4", // 水色（OM3マルチモード）
		"mmf-om4": "#184cdaff", // 青色（OM4マルチモード）
		"smf-os2": "#FFC300", // 黄色（OS2シングルモード）
		"cat6a":   "#FF6B9D", // ピンク（カテゴリ6a銅線）
	}
	if color, ok := cableColors[cableType]; ok {
		return color
	}
	return "#999999" // Unknown cable type
}
