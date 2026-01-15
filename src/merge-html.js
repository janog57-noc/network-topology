#!/usr/bin/env node

/**
 * Generate HTML output from merged topology YAML
 */

import { readFileSync, writeFileSync } from 'fs';
console.log('FS module imported');

import { html } from '@shumoku/renderer';
console.log('Renderer module imported');

import { INTERACTIVE_IIFE } from '@shumoku/renderer/iife-string';
console.log('IIFE module imported');

import yaml from 'js-yaml';
console.log('YAML module imported');

console.log('Starting HTML conversion script...');

// Add immediate execution
console.log('About to define functions...');

/**
 * Convert YAML to HTML
 * @param {string} yamlPath - Path to YAML file
 * @param {string} outputPath - Output HTML file path
 */
function convertYamlToHtml(yamlPath, outputPath) {
    try {
        console.log(`Reading YAML file: ${yamlPath}`);
        // Read YAML file
        const yamlContent = readFileSync(yamlPath, 'utf8');
        console.log('YAML content loaded');
        
        // Parse YAML
        const graph = yaml.load(yamlContent);
        console.log('YAML parsed successfully');
        console.log('Graph keys:', Object.keys(graph));
        
        // Set up HTML renderer
        html.setIIFE(INTERACTIVE_IIFE);
        console.log('HTML renderer configured');
        
        // Render HTML
        const htmlContent = html.render(graph);
        console.log('HTML content rendered');
        
        // Write HTML file
        const fullHtml = `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>Merged Network Topology</title>
    <style>
        body { margin: 0; padding: 0; }
        #shumoku-container { width: 100vw; height: 100vh; }
    </style>
</head>
<body>
    <div id="shumoku-container">${htmlContent}</div>
</body>
</html>`;
        
        console.log(`Writing HTML file: ${outputPath}`);
        writeFileSync(outputPath, fullHtml, 'utf8');
        console.log(`HTML output saved to ${outputPath}`);
    } catch (error) {
        console.error('Error:', error.message);
        console.error('Stack trace:', error.stack);
        process.exit(1);
    }
}

/**
 * Main function
 */
function main() {
    // Check command line arguments
    const args = process.argv.slice(2);
    if (args.length < 2) {
        console.log('Usage: node merge-html.js <input.yaml> <output.html>');
        console.log('Example: node merge-html.js merged-topology.yaml merged-topology.html');
        process.exit(1);
    }
    
    const [inputPath, outputPath] = args;
    
    console.log('Converting YAML to HTML...');
    console.log(`Input: ${inputPath}`);
    console.log(`Output: ${outputPath}`);
    convertYamlToHtml(inputPath, outputPath);
    
    console.log('Conversion completed successfully!');
}

export { convertYamlToHtml };

// Add immediate execution
console.log('About to check if main should run...');
if (import.meta.url === `file://${process.argv[1]}`) {
    console.log('Running main function...');
    main();
}
console.log('Script finished');