// src/ocx-integration/convert.js
/**
 * Convert OCX API data into a Shumoku NetworkGraph.
 *
 * The OCX API provides a set of resources (physical ports, LAG ports,
 * virtual circuits, cloud connections, internet gateways, tunnel gateways, etc.).
 * This module fetches those resources via an OCXClient instance and builds a
 * {@link NetworkGraph} compatible with @shumoku/netbox.
 *
 * The conversion is intentionally lightweight – it creates one node per
 * physical‑port/LAG‑port and one link per virtual‑circuit.  Cloud‑related
 * resources become dedicated nodes of type "cloud", "internet" or "vpn".
 *
 * The output can be serialized to YAML/HTML/SVG using the existing Shumoku
 * helpers (e.g. `toYaml(graph)`).
 */

import { toYaml } from '@shumoku/netbox';

/**
 * Helper to safely extract a string value.
 */
function safeString(value) {
  return typeof value === 'string' ? value : '';
}

/**
 * Main entry point.
 * @param {Object} client - An instance of OCXClient.
 * @param {Object} [options] - Conversion options (currently unused).
 * @returns {Promise<Object>} A Shumoku NetworkGraph.
 */
export async function generateShumokuGraph(client, options = {}) {
  // Fetch all required OCX resources in parallel.
  const [physicalPorts, lags, vcs, vcis, cloudConns, internetConnections, internetGws, tunnelGws, routers] = await Promise.all([
    client.getPhysicalPorts(),
    client.getLAGs(),
    client.getVirtualCircuits(),
    client.getVCIs(),
    client.getCloudConnections(),
    client.getInternetConnections(),
    client.getInternetGateways(),
    client.getTunnelGateways(),
    client.getRouters(),
  ]);

  /** @type {Array<any>} */
  const nodes = [];
  /** @type {Array<any>} */
  const links = [];
  /** @type {Map<number, number>} */
  const virtualRouterInterfaceIdToRouterId = new Map();
  // Track created node IDs to avoid duplicates
  const createdNodeIds = new Set();

  // --- Physical ports -------------------------------------------------
  // Each physical port becomes an l3‑switch node.  The node ID is the
  // port "id" if available, otherwise we fall back to a composite of the
  // device name and the port name.
  for (const port of physicalPorts) {
    const id = `${port.id}`;
    createdNodeIds.add(id);
    nodes.push({
      id,
      label: [`<b>${safeString(port.name)}</b>`, safeString(port.device)],
      shape: 'rounded',
      type: 'l3-switch',
    });
  }

  // --- LAG ports ------------------------------------------------------
  // LAG ports become l2‑switch nodes.
  for (const lag of lags) {
    const id = safeString(lag.id) || `${safeString(lag.device?.name)}-${safeString(lag.name)}`;
    createdNodeIds.add(id);
    nodes.push({
      id,
      label: [`<b>${safeString(lag.name)}</b>`, safeString(lag.device?.name)],
      shape: 'rounded',
      type: 'l2-switch',
    });
  }

  // --- Routers --------------------------------------------------------
  // Routers become router nodes.
  for (const router of routers) {
    const id = `router-${router.id}`;
    createdNodeIds.add(id);
    nodes.push({
      id,
      label: [`<b>${safeString(router.name)}</b>`],
      shape: 'rounded',
      type: 'router',
    });

    // Collect VRI IDs for link generation later.
    if (router.virtualRouterInterfaces && Array.isArray(router.virtualRouterInterfaces)) {
      for (const vri of router.virtualRouterInterfaces) {
        if (vri.id) {
          virtualRouterInterfaceIdToRouterId.set(vri.id, router.id);
        }
      }
    }
  }


  // --- Internet Connections ---------------------------------------------
  // (Currently not used in graph generation, but logged for debugging)
  for (const ic of internetConnections) {
    const id = `ic-${ic.id}`;
    createdNodeIds.add(id);
    nodes.push({
      id,
      label: [`<b>${safeString(ic.name)}</b>`],
      shape: 'rounded',
      type: 'internet',
    });
  }

  // --- Cloud, Internet & VPN nodes ------------------------------------

  for (const cc of cloudConns) {
    const id = `cloud-${cc.id}`;
    if (!createdNodeIds.has(id)) {
      createdNodeIds.add(id);
      nodes.push({
        id,
        label: [`<b>${safeString(cc.name)}</b>`],
        shape: 'rounded',
        type: 'cloud',
      });
    }
  }
  for (const gw of internetGws) {
    const id = `igw-${gw.id}`;
    if (!createdNodeIds.has(id)) {
      createdNodeIds.add(id);
      nodes.push({
        id,
        label: [`<b>${safeString(gw.name)}</b>`],
        shape: 'rounded',
        type: 'internet',
      });
    }
  }
  for (const tg of tunnelGws) {
    const id = `vpn-${safeString(tg.name)}`;
    if (!createdNodeIds.has(id)) {
      createdNodeIds.add(id);
      nodes.push({
        id,
        label: [`<b>${safeString(tg.name)}</b>`],
        shape: 'rounded',
        type: 'vpn',
      });
    }
  }

  // Also create nodes from VCs (in case API endpoints return empty)
  for (const vc of vcs) {
    if (vc.vci) {
      for (const vci of vc.vci) {
        const physicalPortId = `${vci.physicalPort.id}`;
        if (!createdNodeIds.has(physicalPortId)) {
          createdNodeIds.add(physicalPortId);
          nodes.push({
            id: physicalPortId,
            label: [`<b>${safeString(vci.physicalPort.name)}</b>`, safeString(vci.physicalPort.device)],
            shape: 'rounded',
            type: 'l3-switch',
          });
        }
      }
    }

    if (vc.cloudConnection) {
      for (const cc of vc.cloudConnection) {
        const id = `cloud-${cc.id}`;
        if (!createdNodeIds.has(id)) {
          createdNodeIds.add(id);
          nodes.push({
            id,
            label: [`<b>${safeString(cc.name)}</b>`],
            shape: 'rounded',
            type: 'cloud',
          });
        }
      }
    }
    if (vc.internetGateway) {
      const id = `igw-${vc.internetGateway.id}`;
      if (!createdNodeIds.has(id)) {
        createdNodeIds.add(id);
        nodes.push({
          id,
          label: [`<b>${safeString(vc.internetGateway.name)}</b>`],
          shape: 'rounded',
          type: 'internet',
        });
      }
    }
    if (vc.tunnelGateway) {
      const id = `vpn-${safeString(vc.tunnelGateway.name)}`;
      if (!createdNodeIds.has(id)) {
        createdNodeIds.add(id);
        nodes.push({
          id,
          label: [`<b>${safeString(vc.tunnelGateway.name)}</b>`],
          shape: 'rounded',
          type: 'vpn',
        });
      }
    }
  }

  // --- Build Physical Port lookup (Primary/Secondary) -------------------
  // Assume first port is Primary, second is Secondary (based on pairedPortId)
  const sortedPorts = [...physicalPorts].sort((a, b) => a.id - b.id);
  const primaryPortId = sortedPorts[0]?.id ? `${sortedPorts[0].id}` : null;
  const secondaryPortId = sortedPorts[1]?.id ? `${sortedPorts[1].id}` : null;

  // --- Generate links from VCs to Cloud/Internet/Tunnel -----------------
  for (const vc of vcs) {

    const vc_id = `vc-${vc.id}`;
    const vcnode = {
      id: vc_id,
      label: [`<b>${safeString(vc.name)}</b>`],
      shape: 'rounded',
      type: 'l2-switch',
    };
    const vcLinks = [];

    // OCX-Router(v1) virtualRouterInterface
    if (vc.virtualRouterInterface && vc.virtualRouterInterface.length > 0) {
      for (const vri of vc.virtualRouterInterface) {
        if (vri.id) {
          const routerId = virtualRouterInterfaceIdToRouterId.get(vri.id);
          if (routerId) {
            // Link from Router to VC
            vcLinks.push({
              from: { node: vc_id },
              to: { node: `router-${routerId}`, port: `${vri.id}` },
              label: safeString(vri.name),
            });
          }
        }
      }
    }

    // Cloud Connections
    if (vc.cloudConnection && vc.cloudConnection.length > 0) {
      for (const cc of vc.cloudConnection) {
        const cloudNodeId = `cloud-${cc.id}`;
        // Connect to VC to Cloud Connection
        vcLinks.push({
          from: { node: vc_id },
          to: { node: cloudNodeId, port: "port-1"},
        });

      }
    }

    // Internet Connection
    if (vc.internetConnection) {
      console.log(vc.internetConnection)
      vcLinks.push({
        from: { node: vc_id },
        to: { node: `ic-${vc.internetConnection.id}`, port: "port-1"},
      });
    }

    // Internet Gateway
    if (vc.internetGateway) {
      const igwNodeId = `igw-${vc.internetGateway.id}`;
      vcLinks.push({
        from: { node: vc_id },
        to: { node: igwNodeId, port: "port-1"},
      });
    }

    // Tunnel Gateway
    if (vc.tunnelGateway) {
      const tgNodeId = `vpn-${safeString(vc.tunnelGateway.name)}`;
      vcLinks.push({
        from: { node: vc_id },
        to: { node: tgNodeId, port: "port-1"},
      });
    }

    // VCI (Physical Port)
    if (vc.vci && vc.vci.length > 0) {
      for (const vci of vc.vci) {
        console.log(vci)
        const physicalPortId = `${vci.physicalPort.id}`;
        // Connect VC to Phyical Port
        vcLinks.push({
          from: { node: vc_id },
          to: { node: physicalPortId, port: `VLAN${vci.vlanId}`},
          label: safeString(vci.name),
        });
      }
    }

    // Create links
    if (vcLinks.length == 2) {
      // 接続が2つだけの場合、VCノードを作らずに直接リンクを作成
      const label = vcLinks[0].label || vcLinks[1].label || [safeString(vc.name)];
      links.push({
        from: vcLinks[0].to,
        to: vcLinks[1].to,
        label,
      });
    }else if (vcLinks.length > 0) {
      nodes.push(vcnode);
      links.push(...vcLinks);
    }
  }

  // Assemble final graph.
  const graph = {
    version: '1.0.0',
    name: 'OCX Topology',
    description: 'Generated from OCX API',
    nodes,
    links,
    settings: {
      direction: 'TB',
      theme: 'light',
    },
  };

  // Attach a helper to serialize to YAML if desired.
  graph.toYAML = () => toYaml(graph);
  return graph;
}

export default generateShumokuGraph;
