#!/usr/bin/env node

/**
 * Merge OCX and NetBox NetworkGraph JSON files
 *
 * Features:
 * - Maps OCX node IDs to NetBox node IDs (via mapping file)
 * - Merges node properties (OCX overwrites NetBox)
 * - Adds OCX-only nodes to "ocx" subgraph
 * - Preserves all links with ID translation
 *
 * Usage:
 *   node merge-json.js <netbox.json> <ocx.json> -o <output.json> [-m <mapping.json>]
 */

import { readFileSync, writeFileSync, existsSync } from 'fs';

/**
 * Load mapping file
 */
function loadMapping(mappingPath) {
    if (!mappingPath || !existsSync(mappingPath)) {
        return { nodes: {} };
    }
    console.log(`  Loading mapping: ${mappingPath}`);
    return JSON.parse(readFileSync(mappingPath, 'utf8'));
}

/**
 * Merge OCX graph into NetBox graph
 * @param {Object} netboxGraph - NetBox NetworkGraph (base)
 * @param {Object} ocxGraph - OCX NetworkGraph (overlay)
 * @param {Object} mapping - ID mapping { nodes: { ocxId: netboxId } }
 * @returns {Object} Merged NetworkGraph
 */
function mergeOcxIntoNetbox(netboxGraph, ocxGraph, mapping) {
    const nodeMap = new Map();  // id -> node
    const links = [];
    const linkKeys = new Set();
    const subgraphs = [...(netboxGraph.subgraphs || [])];

    // Step 1: Index all NetBox nodes
    for (const node of (netboxGraph.nodes || [])) {
        nodeMap.set(node.id, { ...node });
    }

    // Step 2: Process OCX nodes
    for (const ocxNode of (ocxGraph.nodes || [])) {
        const mappedId = mapping.nodes[ocxNode.id] || ocxNode.id;

        if (nodeMap.has(mappedId)) {
            // Merge: OCX overwrites NetBox properties
            const existing = nodeMap.get(mappedId);
            nodeMap.set(mappedId, {
                ...existing,
                ...ocxNode,
                id: mappedId,  // Keep NetBox ID
                label: ocxNode.label || existing.label,
                parent: existing.parent || 'ocx',  // Keep NetBox parent
            });
            console.log(`    Merged: ${ocxNode.id} -> ${mappedId}`);
        } else {
            // New OCX-only node: add to "ocx" subgraph
            nodeMap.set(mappedId, {
                ...ocxNode,
                id: mappedId,
                parent: ocxNode.parent || 'ocx',
            });
            console.log(`    Added: ${ocxNode.id} -> ${mappedId} (parent: ocx)`);
        }
    }

    // Step 3: Process NetBox links
    for (const link of (netboxGraph.links || [])) {
        const key = makeLinkKey(link);
        if (!linkKeys.has(key)) {
            linkKeys.add(key);
            links.push(link);
        }
    }

    // Step 4: Process OCX links (with ID translation)
    for (const link of (ocxGraph.links || [])) {
        const translatedLink = translateLinkIds(link, mapping);
        const key = makeLinkKey(translatedLink);
        if (!linkKeys.has(key)) {
            linkKeys.add(key);
            links.push(translatedLink);
        }
    }

    return {
        version: '1.0.0',
        name: netboxGraph.name || 'Merged Network Topology',
        description: 'Merged OCX and NetBox topology',
        nodes: Array.from(nodeMap.values()),
        links,
        subgraphs: subgraphs.length > 0 ? subgraphs : undefined,
        settings: netboxGraph.settings || {
            direction: 'TB',
            theme: 'light'
        }
    };
}

/**
 * Translate node IDs in a link using mapping
 */
function translateLinkIds(link, mapping) {
    const translateEndpoint = (endpoint) => {
        if (typeof endpoint === 'string') {
            return mapping.nodes[endpoint] || endpoint;
        }
        if (endpoint && endpoint.node) {
            return {
                ...endpoint,
                node: mapping.nodes[endpoint.node] || endpoint.node
            };
        }
        return endpoint;
    };

    return {
        ...link,
        from: translateEndpoint(link.from),
        to: translateEndpoint(link.to)
    };
}

/**
 * Create a unique key for link deduplication
 */
function makeLinkKey(link) {
    const fromNode = typeof link.from === 'string' ? link.from : link.from?.node;
    const fromPort = typeof link.from === 'object' ? link.from?.port : '';
    const toNode = typeof link.to === 'string' ? link.to : link.to?.node;
    const toPort = typeof link.to === 'object' ? link.to?.port : '';
    return `${fromNode}:${fromPort}->${toNode}:${toPort}`;
}

/**
 * Parse command line arguments
 */
function parseArgs() {
    const args = process.argv.slice(2);
    const inputs = [];
    let output = 'merged.json';
    let mapping = 'ocx-netbox-mapping.json';

    for (let i = 0; i < args.length; i++) {
        if (args[i] === '-o' || args[i] === '--output') {
            output = args[++i];
        } else if (args[i] === '-m' || args[i] === '--mapping') {
            mapping = args[++i];
        } else if (!args[i].startsWith('-')) {
            inputs.push(args[i]);
        }
    }

    return { inputs, output, mapping };
}

function main() {
    const { inputs, output, mapping } = parseArgs();

    if (inputs.length < 2) {
        console.log('Usage: node merge-json.js <netbox.json> <ocx.json> -o <output.json> [-m <mapping.json>]');
        console.log('');
        console.log('Options:');
        console.log('  -o, --output   Output file (default: merged.json)');
        console.log('  -m, --mapping  ID mapping file (default: ocx-netbox-mapping.json)');
        console.log('');
        console.log('Example:');
        console.log('  node src/merge-json.js netbox.json ocx-topology.json -o merged.json');
        console.log('  npx shumoku render merged.json -f html -o diagram.html');
        process.exit(1);
    }

    console.log('Merging NetBox + OCX...');

    // Load files
    console.log(`  Loading NetBox: ${inputs[0]}`);
    const netboxGraph = JSON.parse(readFileSync(inputs[0], 'utf8'));

    console.log(`  Loading OCX: ${inputs[1]}`);
    const ocxGraph = JSON.parse(readFileSync(inputs[1], 'utf8'));

    const mappingData = loadMapping(mapping);

    // Merge
    console.log('  Processing nodes...');
    const merged = mergeOcxIntoNetbox(netboxGraph, ocxGraph, mappingData);

    console.log(`  Result: ${merged.nodes.length} nodes, ${merged.links.length} links`);

    // Write output
    writeFileSync(output, JSON.stringify(merged, null, 2), 'utf8');
    console.log(`Output written to: ${output}`);
    console.log('');
    console.log('To generate HTML:');
    console.log(`  npx shumoku render ${output} -f html -o diagram.html`);
}

main();

export { mergeOcxIntoNetbox };
