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

// --- NetBoxからのレスポンス構造体 ---
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
			Id   int    `json:"id"`
			Name string `json:"name"`
			Role struct { 
				Slug string `json:"slug"` // 階層判定用にRoleを取得すると良い
			} `json:"role"`
		} `json:"device"`
	} `json:"object"`
}

// --- Vis.js用 出力データ構造体 ---
type VisData struct {
	Nodes []VisNode `json:"nodes"`
	Edges []VisEdge `json:"edges"`
}
type VisNode struct {
	Id    string `json:"id"`
	Label string `json:"label"`
	Group string `json:"group,omitempty"` // 色分け用
	Level int    `json:"level,omitempty"` // ★階層の手動指定（必要なら）
}
type VisEdge struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Label string `json:"label"` // 線の上に表示する文字
    Arrows string `json:"arrows,omitempty"`
}

func main() {
	// (NetBoxへのリクエスト部分は前回と同じなので省略。API URLに role も含める設定が必要かもですが、まずはシンプルに)
	// ※注意: NetBox APIのURLパラメータで ?depth=2 などを付けないと device.role が取れない場合があります

	req, _ := http.NewRequest("GET", NetboxURL+"/api/dcim/cables/?limit=0", nil)
	req.Header.Set("Authorization", "Token "+NetboxToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil { panic(err) }
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var data CableResponse
	json.Unmarshal(body, &data)

	// データ変換
	nodeMap := make(map[string]VisNode)
	var edges []VisEdge

	for _, cable := range data.Results {
		if len(cable.ATerminations) == 0 || len(cable.BTerminations) == 0 { continue }
		a := cable.ATerminations[0].Object
		b := cable.BTerminations[0].Object

		// ノード登録（重複排除）
		if _, exists := nodeMap[a.Device.Name]; !exists {
			nodeMap[a.Device.Name] = VisNode{Id: a.Device.Name, Label: a.Device.Name, Group: "router"} 
		}
		if _, exists := nodeMap[b.Device.Name]; !exists {
			nodeMap[b.Device.Name] = VisNode{Id: b.Device.Name, Label: b.Device.Name, Group: "switch"}
		}

		// エッジ登録 (A -> B)
        // ポート名を線の上に表示
		edges = append(edges, VisEdge{
			From:  a.Device.Name,
			To:    b.Device.Name,
			Label: fmt.Sprintf("%s\n|\n%s", a.Name, b.Name),
            // Arrows: "to", // 矢印をつけたい場合はコメントアウト解除
		})
	}

	// MapをSliceに変換
	var nodes []VisNode
	for _, n := range nodeMap {
		nodes = append(nodes, n)
	}

	// JSONファイル出力
	outputData := VisData{Nodes: nodes, Edges: edges}
	file, _ := os.Create("network_data.json")
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encoder.Encode(outputData)
    
    fmt.Println("Generated network_data.json")
}
