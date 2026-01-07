package renderer

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/janog57-noc/network-topology/internal/model"
	"github.com/janog57-noc/network-topology/internal/utils"
)

func GenerateHTML(data *model.TopologyData, svgFilename, outFilename string) error {
	svgData, err := os.ReadFile(svgFilename)
	if err != nil {
		return err
	}

	type ConnJSON struct {
		SrcDev  string `json:"src_dev"`
		SrcPort string `json:"src_port"`
		DstDev  string `json:"dst_dev"`
		DstPort string `json:"dst_port"`
	}
	var connList []ConnJSON
	for _, conn := range data.Connections {
		connList = append(connList, ConnJSON{
			SrcDev:  conn.SrcDev,
			SrcPort: utils.Escape(conn.SrcPort),
			DstDev:  conn.DstDev,
			DstPort: utils.Escape(conn.DstPort),
		})
	}
	connJSON, _ := json.Marshal(connList)

	htmlContent := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Physical Network Topology</title>
    <script src="https://cdn.jsdelivr.net/npm/svg-pan-zoom@3.6.1/dist/svg-pan-zoom.min.js"></script>
    <style>
        body, html { margin: 0; padding: 0; width: 100%%; height: 100%%; overflow: hidden; }
        #svg-container { width: 100%%; height: 100%%; border: 1px solid #ccc; }
        .edge-dimmed { opacity: 0.15 !important; }
        .edge-highlight { opacity: 1 !important; filter: drop-shadow(0 0 4px currentColor); }
        .port-highlight { filter: drop-shadow(0 0 8px #FFD700) brightness(1.2); }
    </style>
</head>
<body>
    <div id="svg-container">%s</div>
    <script>
        const connections = %s;
        window.onload = function () {
            const svg = document.querySelector('#svg-container svg');
            if (!svg) return;
            svg.setAttribute('width', '100%%');
            svg.setAttribute('height', '100%%');
            
            const panZoomInstance = svgPanZoom(svg, {
                zoomEnabled: true, controlIconsEnabled: true, fit: true, center: true
            });
            panZoomInstance.zoom(panZoomInstance.getZoom() * 0.95);
            
            try {
                const edges = svg.querySelectorAll('g.edge');
                edges.forEach(edge => {
                    const title = edge.querySelector('title');
                    if (!title) return;
                    const match = title.textContent.trim().match(/^(.+?)->(.+?)$/);
                    if (!match) return;
                    const [, srcNode, dstNode] = match;
                    const conn = connections.find(c => 
                        (c.src_dev === srcNode.trim() || srcNode.includes(c.src_dev)) &&
                        (c.dst_dev === dstNode.trim() || dstNode.includes(c.dst_dev))
                    );
                    if (!conn) return;
                    
                    edge.style.cursor = 'pointer';
                    edge.addEventListener('mouseenter', () => {
                        edges.forEach(e => e.classList.add('edge-dimmed'));
                        edge.classList.remove('edge-dimmed');
                        edge.classList.add('edge-highlight');
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
                        edges.forEach(e => {
                            e.classList.remove('edge-dimmed');
                            e.classList.remove('edge-highlight');
                        });
                        svg.querySelectorAll('.port-highlight').forEach(el => el.classList.remove('port-highlight'));
                    });
                });
            } catch (e) { console.error(e); }
        };
    </script>
</body>
</html>`, string(svgData), string(connJSON))

	return os.WriteFile(outFilename, []byte(htmlContent), 0644)
}
