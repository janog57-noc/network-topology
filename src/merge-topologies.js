#!/usr/bin/env node

/**
 * Merge OCX and NetBox topology data into a single Shumoku NetworkGraph
 *
 * Output format: NetworkGraph JSON (compatible with `npx shumoku render`)
 */

import { readFileSync, writeFileSync } from 'fs';
import { convertToNetworkGraph } from '@shumoku/netbox';


/**
 * Load NetBox data from JSON files
 * @param {string} devicesPath - Path to devices JSON file
 * @param {string} interfacesPath - Path to interfaces JSON file
 * @param {string} cablesPath - Path to cables JSON file
 * @returns {Object} NetBox data objects
 */
function loadNetBoxData(devicesPath, interfacesPath, cablesPath) {
    const devices = JSON.parse(readFileSync(devicesPath, 'utf8'));
    const interfaces = JSON.parse(readFileSync(interfacesPath, 'utf8'));
    const cables = JSON.parse(readFileSync(cablesPath, 'utf8'));
    return { devices, interfaces, cables };
}

/**
 * Merge OCX and NetBox graphs
 * @param {Object} ocxGraph - OCX NetworkGraph
 * @param {Object} netboxGraph - NetBox NetworkGraph
 * @returns {Object} Merged NetworkGraph
 */
function mergeGraphs(ocxGraph, netboxGraph) {
    // Combine nodes from both graphs
    const nodes = [...netboxGraph.nodes, ...ocxGraph.nodes];
    
    // Combine links from both graphs
    const links = [...netboxGraph.links, ...ocxGraph.links];
    
    // Combine subgraphs if they exist
    const subgraphs = [];
    if (netboxGraph.subgraphs) {
        subgraphs.push(...netboxGraph.subgraphs);
    }
    if (ocxGraph.subgraphs) {
        subgraphs.push(...ocxGraph.subgraphs);
    }
    
    // Create merged graph
    return {
        version: '1.0.0',
        name: 'Merged Network Topology',
        description: 'Combined OCX and NetBox network topology',
        nodes,
        links,
        subgraphs: subgraphs.length > 0 ? subgraphs : undefined,
        settings: {
            direction: 'TB',
            theme: 'light'
        }
    };
}

/**
 * Save graph to JSON file (NetworkGraph format for shumoku render)
 * @param {Object} graph - NetworkGraph object
 * @param {string} outputPath - Output file path
 */
function saveGraph(graph, outputPath) {
    // Remove any non-serializable properties (like toYAML function)
    const { toYAML, ...cleanGraph } = graph;
    writeFileSync(outputPath, JSON.stringify(cleanGraph, null, 2), 'utf8');
    console.log(`Merged topology saved to ${outputPath}`);
}

/**
 * Main function
 */
async function main() {
    try {
        // Check command line arguments
        const args = process.argv.slice(2);
        if (args.length < 4) {
            console.log('Usage: node merge-topologies.js <netbox-devices.json> <netbox-interfaces.json> <netbox-cables.json> <output.json>');
            console.log('Example: node merge-topologies.js netbox-devices.json netbox-interfaces.json netbox-cables.json merged-topology.json');
            console.log('');
            console.log('Then render HTML with: npx shumoku render merged-topology.json -f html -o diagram.html');
            process.exit(1);
        }

        const [devicesPath, interfacesPath, cablesPath, outputPath] = args;
        
        // Load NetBox data
        console.log('Loading NetBox data...');
        const netboxData = loadNetBoxData(devicesPath, interfacesPath, cablesPath);
        
        // Convert NetBox data to graph
        console.log('Converting NetBox data to graph...');
        const netboxGraph = convertToNetworkGraph(
            netboxData.devices,
            netboxData.interfaces,
            netboxData.cables
        );
        
        // Generate OCX graph
        console.log('Generating OCX graph...');
        // Load OCX data from the existing OCX client
        const ocxGraph = JSON.parse(readFileSync('ocx-topology.json', 'utf8'));
        
        // Merge graphs
        console.log('Merging graphs...');
        const mergedGraph = mergeGraphs(ocxGraph, netboxGraph);
        
        // Save merged graph
        console.log('Saving merged graph...');
        saveGraph(mergedGraph, outputPath);
        
        console.log('Merge completed successfully!');
    } catch (error) {
        console.error('Error:', error.message);
        process.exit(1);
    }
}

// Run main function if this script is executed directly
if (import.meta.url === `file://${process.argv[1]}`) {
    main();
}

export { loadNetBoxData, mergeGraphs, saveGraph };