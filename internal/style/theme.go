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
	default:
		return ThemeColor{Header: "#747D8C", Border: "#747D8C", Text: "#FFFFFF", PortFill: "#FFFFFF", Line: "#999999"}
	}
}

func GetVlanColor(vlanID int) string {
	vlanColors := map[int]string{
		10: "#FF6B6B", // 明るい赤
		20: "#4DABF7", // 明るい青
		30: "#51CF66", // 明るい緑
		40: "#FFD43B", // 明るい黄色
	}
	if color, ok := vlanColors[vlanID]; ok {
		return color
	}
	return "#FFFFFF"
}

func GetLevelLineColor(level int) string {
	switch level {
	case 1:
		return "#00BCD4" // ONU
	case 2:
		return "#FF4757" // Router
	case 3:
		return "#7d7d7dff" // Core Switch
	case 4:
		return "#1E90FF" // Edge Switch
	case 5:
		return "#57606F" // Server/AP
	default:
		return "#999999" // Other
	}
}
