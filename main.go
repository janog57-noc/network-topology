package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// 設定
var (
	NetboxURL   = os.Getenv("NETBOX_URL")
	NetboxToken = os.Getenv("NETBOX_TOKEN")
)

// --- NetBox API用 構造体 ---
type DeviceResponse struct {
	Results []struct {
		Name string `json:"name"`
		Role struct {
			Slug string `json:"slug"`
		} `json:"role"`
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

// --- Vis.js出力用 構造体 ---
type VisData struct {
	Nodes []VisNode `json:"nodes"`
	Edges []VisEdge `json:"edges"`
}
type VisNode struct {
	Id    string `json:"id"`
	Label string `json:"label"`
	Group string `json:"group"`
	Level int    `json:"level"`
}
type VisEdge struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Label  string `json:"label"`
	Arrows string `json:"arrows,omitempty"`
    Font   EdgeFont `json:"font,omitempty"`
}
type EdgeFont struct {
    Align string `json:"align"`
}

func main() {
	client := &http.Client{}

	// 1. 全デバイス情報を取得して、役割(Role)をマップ化する
	roleMap := make(map[string]string)
	{
		req, _ := http.NewRequest("GET", NetboxURL+"/api/dcim/devices/?limit=0", nil)
		req.Header.Set("Authorization", "Token "+NetboxToken)
		resp, err := client.Do(req)
		if err == nil {
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			var dResp DeviceResponse
			json.Unmarshal(body, &dResp)
			for _, dev := range dResp.Results {
				roleMap[dev.Name] = dev.Role.Slug
			}
		}
	}

	// 2. ケーブル情報を取得
	var cables CableResponse
	{
		req, _ := http.NewRequest("GET", NetboxURL+"/api/dcim/cables/?limit=0", nil)
		req.Header.Set("Authorization", "Token "+NetboxToken)
		resp, err := client.Do(req)
		if err != nil { panic(err) }
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		json.Unmarshal(body, &cables)
	}

	// 3. データ変換 (Roleに基づいてLevelを決定)
	nodeMap := make(map[string]VisNode)
	var edges []VisEdge

	for _, cable := range cables.Results {
		if len(cable.ATerminations) == 0 || len(cable.BTerminations) == 0 { continue }
		termA := cable.ATerminations[0].Object
		termB := cable.BTerminations[0].Object
		devA := termA.Device.Name
		devB := termB.Device.Name

		// ノード作成関数
		createNode := func(name string) {
			if _, exists := nodeMap[name]; !exists {
				role := roleMap[name]
				level := 10
				
				// ご自身の環境のSlugに合わせて調整してください
				switch role {
				case "edge-router", "router", "firewall":
					level = 0
				case "core-switch":
					level = 1
				case "distribution-switch":
					level = 2
				case "access-switch":
					level = 3
				case "wireless-ap", "server":
					level = 4
				}

				nodeMap[name] = VisNode{
					Id:    name,
					Label: name + "\n(" + role + ")",
					Group: role,
					Level: level,
				}
			}
		}

		createNode(devA)
		createNode(devB)

		edges = append(edges, VisEdge{
			From:  devA,
			To:    devB,
			Label: fmt.Sprintf(" %s \n %s ", termA.Name, termB.Name),
            Font:  EdgeFont{Align: "horizontal"},
		})
	}

	// 出力
	var nodes []VisNode
	for _, n := range nodeMap {
		nodes = append(nodes, n)
	}

	file, _ := os.Create("network_data.json")
	defer file.Close()
	json.NewEncoder(file).Encode(VisData{Nodes: nodes, Edges: edges})
    
    fmt.Println("Generated network_data.json with levels")
}
