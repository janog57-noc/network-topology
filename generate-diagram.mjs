#!/usr/bin/env node

/**
 * Generate network topology diagram from OCX and NetBox data
 *
 * Usage:
 *   node generate-diagram.mjs -o diagram.html
 */

import 'dotenv/config';
import { execSync } from 'child_process';
import { readFileSync, writeFileSync, existsSync } from 'fs';

const args = process.argv.slice(2);
let outputFile = 'diagram.html';

for (let i = 0; i < args.length; i++) {
    if (args[i] === '-o' || args[i] === '--output') {
        outputFile = args[++i];
    }
}

console.log('=== Network Topology Diagram Generator ===\n');

// Step 1: Fetch OCX data
console.log('Step 1: Fetching OCX data...');
try {
    execSync('node src/ocx-integration/cli.mjs -o ocx-topology.json', {
        stdio: 'inherit',
        env: process.env
    });
} catch (e) {
    console.error('Failed to fetch OCX data');
    process.exit(1);
}

// Step 2: Fetch NetBox data
console.log('\nStep 2: Fetching NetBox data...');
const netboxUrl = process.env.NETBOX_URL;
const netboxToken = process.env.NETBOX_TOKEN;

if (!netboxUrl || !netboxToken) {
    console.log('  NetBox not configured, skipping...');
} else {
    try {
        execSync(`npx netbox-to-shumoku -u "${netboxUrl}" -t "${netboxToken}" -f json -o netbox.json`, {
            stdio: 'inherit'
        });
    } catch (e) {
        console.error('Failed to fetch NetBox data, continuing with OCX only...');
    }
}

// Step 3: Merge data
console.log('\nStep 3: Merging data...');
const filesToMerge = [];

if (existsSync('ocx-topology.json')) {
    filesToMerge.push('ocx-topology.json');
}
if (existsSync('netbox.json')) {
    filesToMerge.push('netbox.json');
}

if (filesToMerge.length === 0) {
    console.error('No data files to merge');
    process.exit(1);
}

if (filesToMerge.length === 1) {
    // Only one file, just copy it
    const content = readFileSync(filesToMerge[0], 'utf8');
    writeFileSync('merged.json', content);
    console.log(`  Using ${filesToMerge[0]} directly`);
} else {
    // Merge multiple files
    execSync(`node src/merge-json.js ${filesToMerge.join(' ')} -o merged.json`, {
        stdio: 'inherit'
    });
}

// Step 4: Generate HTML
console.log('\nStep 4: Generating HTML...');
execSync(`npx shumoku render merged.json -f html -o "${outputFile}"`, {
    stdio: 'inherit'
});

console.log(`\n=== Done! Output: ${outputFile} ===`);
