#!/usr/bin/env node

/**
 * CLI to fetch OCX data and convert it to a Shumoku NetworkGraph JSON.
 *
 * Setup:
 *   1. Copy .env.example to .env
 *   2. Fill in your OCX credentials
 *
 * Usage:
 *   node src/ocx-integration/cli.mjs -o ocx-topology.json
 */

import 'dotenv/config';
import fs from 'fs/promises';
import { Command } from 'commander';
import OCXClient from './ocx-client.js';
import { generateShumokuGraph } from './convert.js';

const program = new Command();

program
  .option('--url <url>', 'OCX API base URL (or OCX_URL env)')
  .option('--token <token>', 'Bearer token (or OCX_TOKEN env)')
  .option('--token-id <tokenId>', 'X-Token-Id (or OCX_TOKEN_ID env)')
  .option('-o, --output <file>', 'Output JSON file path (default: stdout)')
  .parse();

const options = program.opts();

// Use environment variables as fallback
const config = {
  url: options.url || process.env.OCX_URL,
  token: options.token || process.env.OCX_TOKEN,
  tokenId: options.tokenId || process.env.OCX_TOKEN_ID
};

// Validate required inputs
if (!config.url || !config.token || !config.tokenId) {
  console.error('Missing required configuration.');
  console.error('Provide via CLI options or environment variables:');
  console.error('  --url / OCX_URL');
  console.error('  --token / OCX_TOKEN');
  console.error('  --token-id / OCX_TOKEN_ID');
  process.exit(1);
}

async function main() {
  try {
    const client = new OCXClient({
      baseUrl: config.url,
      token: config.token,
      tokenId: config.tokenId
    });

    console.error('Fetching OCX data...');
    const graph = await generateShumokuGraph(client);

    // Remove non-serializable properties
    const { toYAML, ...cleanGraph } = graph;
    const json = JSON.stringify(cleanGraph, null, 2);

    if (options.output) {
      await fs.writeFile(options.output, json, 'utf-8');
      console.error(`Wrote NetworkGraph JSON to ${options.output}`);
      console.error(`  Nodes: ${cleanGraph.nodes.length}`);
      console.error(`  Links: ${cleanGraph.links.length}`);
    } else {
      // Output to stdout for piping
      console.log(json);
    }
  } catch (error) {
    console.error('Error:', error.message);
    process.exit(1);
  }
}

main();