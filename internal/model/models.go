package model

type DeviceData struct {
	Name       string
	PrimaryTag string
	Ports      map[string]bool
	PortVlans  map[string][]int // port -> VLAN IDs
	FlipLabel  bool             // flip the order of header and ports
}

type Connection struct {
	SrcDev, SrcPort    string
	DstDev, DstPort    string
	SrcLevel, DstLevel int
	DstTag             string
}

type TopologyData struct {
	Devices     map[string]*DeviceData
	Connections []Connection
}
