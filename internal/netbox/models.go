package netbox

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
