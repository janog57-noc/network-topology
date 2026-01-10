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
		SrcDev    string `json:"src_dev"`
		SrcPort   string `json:"src_port"`
		SrcPortID string `json:"src_port_id"`
		DstDev    string `json:"dst_dev"`
		DstPort   string `json:"dst_port"`
		DstPortID string `json:"dst_port_id"`
	}
	var connList []ConnJSON
	for _, conn := range data.Connections {
		connList = append(connList, ConnJSON{
			SrcDev:    conn.SrcDev,
			SrcPort:   conn.SrcPort,
			SrcPortID: utils.Escape(conn.SrcPort),
			DstDev:    conn.DstDev,
			DstPort:   conn.DstPort,
			DstPortID: utils.Escape(conn.DstPort),
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
        body, html { margin: 0; padding: 0; width: 100%%; height: 100%%; overflow: hidden; font-family: Arial, sans-serif; }
        .tab-bar {
            display: flex;
            background: #f5f5f5;
            border-bottom: 1px solid #ddd;
            padding: 0;
            margin: 0;
        }
        .tab-btn {
            padding: 12px 24px;
            border: none;
            background: transparent;
            cursor: pointer;
            font-size: 14px;
            font-weight: 500;
            color: #666;
            border-bottom: 2px solid transparent;
            transition: all 0.2s;
        }
        .tab-btn:hover { background: #eee; }
        .tab-btn.active {
            color: #1976d2;
            border-bottom-color: #1976d2;
            background: #fff;
        }
        .tab-content {
            display: none;
            width: 100%%;
            height: calc(100%% - 45px);
        }
        .tab-content.active { display: block; }
        #svg-container { width: 100%%; height: 100%%; }
        #shumoku-container { width: 100%%; height: 100%%; }
        #shumoku-container object { width: 100%%; height: 100%%; }
        .edge-dimmed { opacity: 0.15 !important; }
        .edge-highlight { opacity: 1 !important; filter: drop-shadow(0 0 4px currentColor); }
        .port-highlight { filter: drop-shadow(0 0 8px #FFD700) brightness(1.2); }
        #tooltip {
            position: fixed;
            background: #333;
            color: #fff;
            padding: 8px 12px;
            border-radius: 4px;
            font-size: 12px;
            pointer-events: none;
            z-index: 1000;
            display: none;
            white-space: nowrap;
            box-shadow: 0 2px 8px rgba(0,0,0,0.3);
        }
    </style>
</head>
<body>
    <div class="tab-bar">
        <button class="tab-btn active" data-tab="graphviz">Graphviz</button>
        <button class="tab-btn" data-tab="shumoku">Shumoku</button>
    </div>
    <div id="graphviz-tab" class="tab-content active">
        <div id="svg-container">%s</div>
    </div>
    <div id="shumoku-tab" class="tab-content">
        <div id="shumoku-container"></div>
    </div>
    <div id="tooltip"></div>
    <script>
        const connections = %s;
        const tooltip = document.getElementById('tooltip');
        let shumokuPanZoom = null;
        let shumokuLoaded = false;

        // Tab switching
        document.querySelectorAll('.tab-btn').forEach(btn => {
            btn.addEventListener('click', () => {
                document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
                document.querySelectorAll('.tab-content').forEach(c => c.classList.remove('active'));
                btn.classList.add('active');
                document.getElementById(btn.dataset.tab + '-tab').classList.add('active');

                // Initialize Shumoku pan-zoom on first view
                if (btn.dataset.tab === 'shumoku' && !shumokuLoaded) {
                    loadShumokuSvg();
                }
            });
        });

        function loadShumokuSvg() {
            fetch('topology-shumoku.svg')
                .then(res => res.text())
                .then(svgText => {
                    const container = document.getElementById('shumoku-container');
                    container.innerHTML = svgText;
                    const svg = container.querySelector('svg');
                    if (svg) {
                        svg.removeAttribute('width');
                        svg.removeAttribute('height');
                        svg.style.width = '100%%';
                        svg.style.height = '100%%';
                        svg.style.maxWidth = '100%%';
                        svg.style.maxHeight = '100%%';
                        // Wait for DOM update before initializing pan-zoom
                        requestAnimationFrame(() => {
                            shumokuPanZoom = svgPanZoom(svg, {
                                zoomEnabled: true,
                                controlIconsEnabled: true,
                                fit: true,
                                center: true,
                                minZoom: 0.1,
                                maxZoom: 20
                            });
                        });
                    }
                    shumokuLoaded = true;
                })
                .catch(err => console.error('Failed to load Shumoku SVG:', err));
        }

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

                    // Disable default SVG tooltip after parsing to avoid double popups
                    title.textContent = '';

                    edge.style.cursor = 'pointer';
                    const tooltipText = conn.src_dev + ':' + conn.src_port + ' → ' + conn.dst_dev + ':' + conn.dst_port;

                    edge.addEventListener('mouseenter', (e) => {
                        edges.forEach(el => el.classList.add('edge-dimmed'));
                        edge.classList.remove('edge-dimmed');
                        edge.classList.add('edge-highlight');
                        const srcPortId = conn.src_dev + ':' + conn.src_port_id;
                        const dstPortId = conn.dst_dev + ':' + conn.dst_port_id;
                        svg.querySelectorAll('[id]').forEach(el => {
                            const id = el.getAttribute('id');
                            if (id === srcPortId || id === dstPortId) {
                                el.classList.add('port-highlight');
                            }
                        });

                        tooltip.textContent = tooltipText;
                        tooltip.style.display = 'block';
                    });

                    edge.addEventListener('mousemove', (e) => {
                        tooltip.style.left = (e.clientX + 10) + 'px';
                        tooltip.style.top = (e.clientY + 10) + 'px';
                    });

                    edge.addEventListener('mouseleave', () => {
                        edges.forEach(e => {
                            e.classList.remove('edge-dimmed');
                            e.classList.remove('edge-highlight');
                        });
                        svg.querySelectorAll('.port-highlight').forEach(el => el.classList.remove('port-highlight'));
                        tooltip.style.display = 'none';
                    });
                });
            } catch (e) { console.error(e); }
        };
    </script>
</body>
</html>`, string(svgData), string(connJSON))

	return os.WriteFile(outFilename, []byte(htmlContent), 0644)
}
