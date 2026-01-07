/**
 * Shared utilities for Home Assistant addon integration
 * Used by gateway, mqtt-bridge, and security web components
 */

/**
 * Get the base URL for API calls, handling Home Assistant ingress paths
 * 
 * When running as a Home Assistant addon with ingress enabled, the URL will include
 * a path like /api/hassio_ingress/<token>/. This function detects that pattern and
 * returns the correct base URL.
 * 
 * @returns {string} The base URL to use for API calls
 * 
 * @example
 * // In Home Assistant: https://homeassistant.local/api/hassio_ingress/abc123/
 * // Returns: https://homeassistant.local/api/hassio_ingress/abc123
 * 
 * // In development: http://localhost:3000/
 * // Returns: http://localhost:3000
 */
export function getBaseUrl() {
  // Detect if we're running in Home Assistant with ingress
  if (window.location.pathname.includes('/api/hassio_ingress/')) {
    // Extract the ingress path
    const pathParts = window.location.pathname.split('/')
    const ingressIndex = pathParts.indexOf('api')
    if (ingressIndex >= 0 && pathParts[ingressIndex + 1] === 'hassio_ingress') {
      const ingressPath = pathParts.slice(0, ingressIndex + 3).join('/')
      return window.location.origin + ingressPath
    }
  }
  
  // Fallback to current origin
  return window.location.origin
}

/**
 * Get the WebSocket URL for real-time connections, handling Home Assistant ingress paths
 * 
 * @param {string} path - The WebSocket path (e.g., '/api/topics?stream=true')
 * @returns {string} The complete WebSocket URL
 * 
 * @example
 * // In Home Assistant with HTTPS
 * getWebSocketUrl('/api/topics?stream=true')
 * // Returns: wss://homeassistant.local/api/hassio_ingress/abc123/api/topics?stream=true
 * 
 * // In development with HTTP
 * getWebSocketUrl('/api/topics?stream=true')
 * // Returns: ws://localhost:3000/api/topics?stream=true
 */
export function getWebSocketUrl(path) {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const baseUrl = getBaseUrl()
  
  // Extract pathname from base URL
  const baseUrlObj = new URL(baseUrl, window.location.origin)
  const basePath = baseUrlObj.pathname === '/' ? '' : baseUrlObj.pathname
  
  // Ensure path starts with /
  const normalizedPath = path.startsWith('/') ? path : `/${path}`
  
  return `${protocol}//${window.location.host}${basePath}${normalizedPath}`
}

/**
 * Make an API request with proper path handling for Home Assistant ingress
 * 
 * @param {string} endpoint - The API endpoint (e.g., 'dns/external/ipv4' or '/api/dns/external/ipv4')
 * @param {RequestInit} options - Fetch options (method, headers, body, etc.)
 * @returns {Promise<Response>} Fetch response
 */
export async function apiRequest(endpoint, options = {}) {
  const baseUrl = getBaseUrl()
  
  // Normalize endpoint - remove leading slash and 'api/' prefix if present
  let normalizedEndpoint = endpoint.replace(/^\/+/, '')
  if (!normalizedEndpoint.startsWith('api/')) {
    normalizedEndpoint = `api/${normalizedEndpoint}`
  }
  
  const url = `${baseUrl}/${normalizedEndpoint}`
  
  return await fetch(url, {
    headers: {
      'Content-Type': 'application/json',
      ...options.headers
    },
    ...options
  })
}

/**
 * Make a GET request to an API endpoint
 * 
 * @param {string} endpoint - The API endpoint
 * @returns {Promise<any>} Parsed JSON response
 */
export async function apiGet(endpoint) {
  const response = await apiRequest(endpoint)
  
  if (!response.ok) {
    throw new Error(`API request failed: ${response.status} ${response.statusText}`)
  }
  
  const contentType = response.headers.get('content-type')
  if (contentType && contentType.includes('application/json')) {
    return await response.json()
  } else {
    const text = await response.text()
    throw new Error(`Expected JSON response but got ${contentType}: ${text.substring(0, 100)}...`)
  }
}

/**
 * Make a POST request to an API endpoint
 * 
 * @param {string} endpoint - The API endpoint
 * @param {any} data - Data to send in the request body
 * @returns {Promise<any>} Parsed JSON response or response object
 */
export async function apiPost(endpoint, data) {
  const response = await apiRequest(endpoint, {
    method: 'POST',
    body: JSON.stringify(data)
  })
  
  if (!response.ok) {
    throw new Error(`API request failed: ${response.status} ${response.statusText}`)
  }
  
  const contentType = response.headers.get('content-type')
  if (contentType && contentType.includes('application/json')) {
    return await response.json()
  }
  
  return response
}

/**
 * Make a PUT request to an API endpoint
 * 
 * @param {string} endpoint - The API endpoint
 * @param {any} data - Data to send in the request body
 * @returns {Promise<any>} Parsed JSON response or response object
 */
export async function apiPut(endpoint, data) {
  const response = await apiRequest(endpoint, {
    method: 'PUT',
    body: JSON.stringify(data)
  })
  
  if (!response.ok) {
    throw new Error(`API request failed: ${response.status} ${response.statusText}`)
  }
  
  const contentType = response.headers.get('content-type')
  if (contentType && contentType.includes('application/json')) {
    return await response.json()
  }
  
  return response
}

/**
 * Make a DELETE request to an API endpoint
 * 
 * @param {string} endpoint - The API endpoint
 * @returns {Promise<any>} Parsed JSON response or response object
 */
export async function apiDelete(endpoint) {
  const response = await apiRequest(endpoint, {
    method: 'DELETE'
  })
  
  if (!response.ok) {
    throw new Error(`API request failed: ${response.status} ${response.statusText}`)
  }
  
  const contentType = response.headers.get('content-type')
  if (contentType && contentType.includes('application/json')) {
    return await response.json()
  }
  
  return response
}
