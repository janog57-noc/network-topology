/**
 * OCX API Client
 * 
 * This module provides functions to interact with the OCX API and retrieve
 * network resource information for integration with Shumoku diagrams.
 */

class OCXClient {
  /**
   * Constructor for OCXClient
   * @param {Object} options - Configuration options
   * @param {string} options.baseUrl - Base URL for the OCX API
   * @param {string} options.token - API token for authentication
   * @param {string} options.tokenId - Token ID for authentication
   */
  constructor(options) {
    this.baseUrl = options.baseUrl;
    this.token = options.token;
    this.tokenId = options.tokenId;
  }

  /**
   * Make an authenticated request to the OCX API
   * @param {string} endpoint - API endpoint to call
   * @returns {Promise<Object>} Response data from the API
   */
  async _request(endpoint) {
    const url = `${this.baseUrl}${endpoint}`;
    
    const headers = {
      'Authorization': `Bearer ${this.token}`,
      'X-Token-Id': this.tokenId,
      'Content-Type': 'application/json'
    };

    try {
      const response = await fetch(url, {
        method: 'GET',
        headers: headers
      });

      if (!response.ok) {
        throw new Error(`OCX API request failed: ${response.status} ${response.statusText}`);
      }

      return await response.json();
    } catch (error) {
      console.error(`Error requesting ${url}:`, error);
      throw error;
    }
  }

  /**
   * Get all physical ports
   * @returns {Promise<Array>} Array of physical port objects
   */
  async getPhysicalPorts() {
    try {
      const response = await this._request('/api/v1/physicalPorts');
      return response.items || response.physicalPorts || [];
    } catch (error) {
      if (error.message.includes('404')) {
        console.warn('Physical ports endpoint not found, returning empty array');
        return [];
      }
      throw error;
    }
  }

  /**
   * Get all LAG ports
   * @returns {Promise<Array>} Array of LAG port objects
   */
  async getLAGs() {
    try {
      const response = await this._request('/api/v1/lags');
      return response.items || response.lagPorts || [];
    } catch (error) {
      if (error.message.includes('404')) {
        console.warn('LAGs endpoint not found, returning empty array');
        return [];
      }
      throw error;
    }
  }

  /**
   * Get all virtual circuits
   * @returns {Promise<Array>} Array of virtual circuit objects
   */
  async getVirtualCircuits() {
    try {
      const response = await this._request('/api/v1/vcs');
      return response.items || response.vcs || [];
    } catch (error) {
      if (error.message.includes('404')) {
        console.warn('Virtual circuits endpoint not found, returning empty array');
        return [];
      }
      throw error;
    }
  }

  /**
   * Get all virtual circuit interfaces
   * @returns {Promise<Array>} Array of VCI objects
   */
  async getVCIs() {
    try {
      const response = await this._request('/api/v1/vcis');
      console.log(response)
      return response.items || response.vcis || [];
    } catch (error) {
      if (error.message.includes('404')) {
        console.warn('VCIs endpoint not found, returning empty array');
        return [];
      }
      throw error;
    }
  }

  /**
   * Get all cloud connections
   * @returns {Promise<Array>} Array of cloud connection objects
   */
  async getCloudConnections() {
    try {
      const response = await this._request('/api/v1/cloudConnections');
      return response.items || response.cloudConnections || [];
    } catch (error) {
      if (error.message.includes('404')) {
        console.warn('Cloud connections endpoint not found, returning empty array');
        return [];
      }
      throw error;
    }
  }

  /**
   * Get all internet connections
   * @returns {Promise<Array>} Array of internet gateway objects
   */
  async getInternetConnections() {
    try {
      const response = await this._request('/api/v1/internets');
      return response.items || response.internetConnections || [];
    } catch (error) {
      if (error.message.includes('404')) {
        console.warn('Internet connections endpoint not found, returning empty array');
        return [];
      }
      throw error;
    }
  }

  /**
   * Get all internet gateways
   * @returns {Promise<Array>} Array of internet gateway objects
   */
  async getInternetGateways() {
    try {
      const response = await this._request('/api/v2/internets');
      return response.items || response.internetGateways || [];
    } catch (error) {
      if (error.message.includes('404')) {
        console.warn('Internet gateways endpoint not found, returning empty array');
        return [];
      }
      throw error;
    }
  }


  /**
   * Get all tunnel gateways
   * @returns {Promise<Array>} Array of tunnel gateway objects
   */
  async getTunnelGateways() {
    try {
      const response = await this._request('/api/v1/tunnelGateways');
      return response.items || response.tunnelGateways || [];
    } catch (error) {
      if (error.message.includes('404')) {
        console.warn('Tunnel gateways endpoint not found, returning empty array');
        return [];
      }
      throw error;
    }
  }

  /**
   * Get all OCX-Router(v1)
   * @returns {Promise<Array>} Array of OCX-Router objects
   */
  async getRouters() {
    try {
      const response = await this._request('/api/v1/routers');
      return response.items || response.virtualRouters || [];
    } catch (error) {
      if (error.message.includes('404')) {
        console.warn('Tunnel gateways endpoint not found, returning empty array');
        return [];
      }
      throw error;
    }
  }

  /**
   * Get all SaaS connections
   * @returns {Promise<Array>} Array of SaaS connection objects
   */
  async getSaaSConnections() {
    const response = await this._request('/api/v1/saasconnections');
    return response.saasConnections || [];
  }

  /**
   * Get all sites
   * @returns {Promise<Array>} Array of site objects
   */
  async getSites() {
    const response = await this._request('/api/v1/sites');
    return response.sites || [];
  }
}

export default OCXClient;