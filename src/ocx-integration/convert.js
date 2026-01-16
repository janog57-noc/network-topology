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
  const [physicalPorts, lags, vcs, vcis, cloudConns, internetGws, tunnelGws] = await Promise.all([
    client.getPhysicalPorts(),
    client.getLAGs(),
    client.getVirtualCircuits(),
    client.getVCIs(),
    client.getCloudConnections(),
    client.getInternetGateways(),
    client.getTunnelGateways(),
  ]);

  /** @type {Array<any>} */
  const nodes = [];
  /** @type {Array<any>} */
  const links = [];

  // --- Physical ports -------------------------------------------------
  // Each physical port becomes an l3‑switch node.  The node ID is the
  // port "id" if available, otherwise we fall back to a composite of the
  // device name and the port name.
  for (const port of physicalPorts) {
    const id = `${port.id}`;
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
    nodes.push({
      id,
      label: [`<b>${safeString(lag.name)}</b>`, safeString(lag.device?.name)],
      shape: 'rounded',
      type: 'l2-switch',
    });
  }

  // --- Cloud, Internet & VPN nodes ------------------------------------
  // Track created node IDs to avoid duplicates
  const createdNodeIds = new Set();

  for (const cc of cloudConns) {
    const id = `cloud-${safeString(cc.name)}`;
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
    const id = `internet-${safeString(gw.name)}`;
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
    if (vc.cloudConnection) {
      for (const cc of vc.cloudConnection) {
        const id = `cloud-${safeString(cc.name)}`;
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
      const id = `internet-${safeString(vc.internetGateway.name)}`;
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
    // Cloud Connections
    if (vc.cloudConnection && vc.cloudConnection.length > 0) {
      for (const cc of vc.cloudConnection) {
        const cloudNodeId = `cloud-${safeString(cc.name)}`;
        // Check VRI name for Primary/Secondary hint
        const vris = vc.virtualRouterInterface || [];
        const isPrimary = vris.some(v =>
          safeString(v.name).toLowerCase().includes('primary') ||
          safeString(v.name).includes('01')
        );
        const portId = isPrimary ? primaryPortId : secondaryPortId;

        if (portId) {
          links.push({
            from: { node: portId },
            to: { node: cloudNodeId },
            label: [safeString(vc.name)],
            type: 'dashed',
          });
        }
      }
    }

    // Internet Gateway
    if (vc.internetGateway) {
      const igwNodeId = `internet-${safeString(vc.internetGateway.name)}`;
      // Connect to both ports if multiple VRIs, otherwise primary
      const vris = vc.virtualRouterInterface || [];
      if (vris.length >= 2 && primaryPortId && secondaryPortId) {
        links.push({
          from: { node: primaryPortId },
          to: { node: igwNodeId },
          label: [safeString(vc.name)],
          type: 'dashed',
        });
        links.push({
          from: { node: secondaryPortId },
          to: { node: igwNodeId },
          type: 'dashed',
        });
      } else if (primaryPortId) {
        links.push({
          from: { node: primaryPortId },
          to: { node: igwNodeId },
          label: [safeString(vc.name)],
          type: 'dashed',
        });
      }
    }

    // Tunnel Gateway
    if (vc.tunnelGateway) {
      const tgwNodeId = `vpn-${safeString(vc.tunnelGateway.name)}`;
      // Connect to both ports if multiple VRIs
      const vris = vc.virtualRouterInterface || [];
      if (vris.length >= 2 && primaryPortId && secondaryPortId) {
        links.push({
          from: { node: primaryPortId },
          to: { node: tgwNodeId },
          label: [safeString(vc.name)],
          type: 'dashed',
        });
        links.push({
          from: { node: secondaryPortId },
          to: { node: tgwNodeId },
          type: 'dashed',
        });
      } else if (primaryPortId) {
        links.push({
          from: { node: primaryPortId },
          to: { node: tgwNodeId },
          label: [safeString(vc.name)],
          type: 'dashed',
        });
      }
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
