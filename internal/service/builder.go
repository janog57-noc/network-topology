package service

import (
	"github.com/janog57-noc/network-topology/internal/model"
	"github.com/janog57-noc/network-topology/internal/netbox"
)

type Builder struct{}

func NewBuilder() *Builder {
	return &Builder{}
}

func (b *Builder) Build(
	dResp *netbox.DeviceResponse,
	iResp *netbox.InterfaceResponse,
	cResp *netbox.CableResponse,
) *model.TopologyData {

	// 1. タグマップ作成
	deviceTagMap := make(map[string]string)
	for _, dev := range dResp.Results {
		deviceTagMap[dev.Name] = resolvePrimaryTag(dev.Tags)
	}

	// 2. ポートVLANマップ作成
	portVlanMap := make(map[string]map[string][]int)
	for _, iface := range iResp.Results {
		devName := iface.Device.Name
		portName := iface.Name
		if portVlanMap[devName] == nil {
			portVlanMap[devName] = make(map[string][]int)
		}
		// VLAN収集（重複排除）
		vlanSet := make(map[int]bool)
		var vlans []int
		if iface.UntaggedVlan.Vid > 0 {
			vlanSet[iface.UntaggedVlan.Vid] = true
		}
		for _, tv := range iface.TaggedVlans {
			vlanSet[tv.Vid] = true
		}
		for vid := range vlanSet {
			vlans = append(vlans, vid)
		}
		portVlanMap[devName][portName] = vlans
	}

	// 3. データ構築
	devices := make(map[string]*model.DeviceData)
	var connections []model.Connection

	for _, cable := range cResp.Results {
		if len(cable.ATerminations) == 0 || len(cable.BTerminations) == 0 {
			continue
		}
		termA := cable.ATerminations[0].Object
		termB := cable.BTerminations[0].Object

		nameA := termA.Device.Name
		nameB := termB.Device.Name
		tagA := getTag(deviceTagMap, nameA)
		tagB := getTag(deviceTagMap, nameB)

		registerDevice(devices, nameA, tagA, termA.Name, portVlanMap[nameA][termA.Name])
		registerDevice(devices, nameB, tagB, termB.Name, portVlanMap[nameB][termB.Name])

		levelA := getLevelByTag(tagA)
		levelB := getLevelByTag(tagB)

		conn := model.Connection{
			SrcDev: nameA, SrcPort: termA.Name, SrcLevel: levelA,
			DstDev: nameB, DstPort: termB.Name, DstLevel: levelB,
			DstTag: tagB,
		}
		if levelA > levelB {
			conn = model.Connection{
				SrcDev: nameB, SrcPort: termB.Name, SrcLevel: levelB,
				DstDev: nameA, DstPort: termA.Name, DstLevel: levelA,
				DstTag: tagA,
			}
		}
		connections = append(connections, conn)
	}

	return &model.TopologyData{
		Devices:     devices,
		Connections: connections,
	}
}

// --- Helper Functions ---

func getTag(m map[string]string, name string) string {
	if t, ok := m[name]; ok && t != "" {
		return t
	}
	return "other"
}

func registerDevice(devices map[string]*model.DeviceData, name, tag, port string, vlans []int) {
	if _, ok := devices[name]; !ok {
		flipLabel := shouldFlipLabel(tag, name)
		devices[name] = &model.DeviceData{
			Name:       name,
			PrimaryTag: tag,
			Ports:      make(map[string]bool),
			PortVlans:  make(map[string][]int),
			FlipLabel:  flipLabel,
		}
	}
	devices[name].Ports[port] = true
	devices[name].PortVlans[port] = vlans
}

func shouldFlipLabel(tag, deviceName string) bool {
	// server タグの機器は全て反転
	if tag == "server" {
		return true
	}
	// その他の特定の機器
	flipDevices := map[string]bool{
		"AL-J3":        true,
		"SAKURA-J3-01": true,
		"SW-J3-02":     true,
		"CS-J3-01":     true,
		// J4 系
		"SW-J4-01": true,
		"SW-J4-02": true,
		// J6 系
		"SW-J6-01": true,
		"SW-J6-04": true,
		"SW-J6-05": true,
	}
	return flipDevices[deviceName]
}

func resolvePrimaryTag(tags []netbox.NetboxTag) string {
	priority := []string{"ocx", "onu", "router", "core-switch", "edge-switch", "server", "console-server", "ap"}
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
	case "ocx":
		return 0
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
	case "console-server":
		return 6
	default:
		return 99
	}
}
