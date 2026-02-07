/**
 * Shumoku Plugin Entry Point
 *
 * Registers the OCX plugin with the Shumoku plugin registry.
 */

import { OCXPlugin } from './plugin.js'

export function register(pluginRegistry) {
  pluginRegistry.register(
    'ocx',
    'Open Cloud Exchange',
    ['topology'],
    (config) => {
      const plugin = new OCXPlugin()
      plugin.initialize(config)
      return plugin
    }
  )
}

export { OCXPlugin }
