/**
 * OCX Plugin for Shumoku
 *
 * Provides topology data from Open Cloud Exchange API.
 */

import OCXClient from './client.js'
import { generateShumokuGraph } from './converter.js'

export class OCXPlugin {
  type = 'ocx'
  displayName = 'Open Cloud Exchange'
  capabilities = ['topology']

  /** @type {OCXClient | null} */
  client = null

  /**
   * Initialize the plugin with configuration
   * @param {Object} config
   * @param {string} config.url - OCX API URL
   * @param {string} config.token - Bearer token
   * @param {string} config.tokenId - X-Token-Id
   */
  initialize(config) {
    if (!config.url || !config.token || !config.tokenId) {
      throw new Error('OCX plugin requires url, token, and tokenId')
    }

    this.client = new OCXClient({
      baseUrl: config.url,
      token: config.token,
      tokenId: config.tokenId,
    })
  }

  /**
   * Test connection to OCX API
   * @returns {Promise<{success: boolean, message: string}>}
   */
  async testConnection() {
    if (!this.client) {
      return { success: false, message: 'Plugin not initialized' }
    }

    try {
      // Try to fetch physical ports as a connection test
      await this.client.getPhysicalPorts()
      return { success: true, message: 'Connected to OCX API' }
    } catch (err) {
      return {
        success: false,
        message: err instanceof Error ? err.message : 'Connection failed',
      }
    }
  }

  /**
   * Fetch topology from OCX API
   * @returns {Promise<Object>} NetworkGraph
   */
  async fetchTopology() {
    if (!this.client) {
      throw new Error('Plugin not initialized')
    }

    const graph = await generateShumokuGraph(this.client)

    // Remove the toYAML helper (not serializable)
    delete graph.toYAML

    return graph
  }

  /**
   * Cleanup resources
   */
  dispose() {
    this.client = null
  }
}

export default OCXPlugin
