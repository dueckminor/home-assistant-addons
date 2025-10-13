// API utility functions for Home Assistant ingress compatibility

/**
 * Makes an API request with proper path handling for Home Assistant ingress
 * @param {string} endpoint - The API endpoint (e.g., 'dns/external/ipv4')
 * @param {object} options - Fetch options (method, headers, body, etc.)
 * @returns {Promise<Response>} - Fetch response
 */
export async function apiRequest(endpoint, options = {}) {
  // Use relative path - this works both in development and Home Assistant ingress
  const url = `api/${endpoint}`
  
  return await fetch(url, {
    headers: {
      'Content-Type': 'application/json',
      ...options.headers
    },
    ...options
  })
}

/**
 * Makes a GET request to an API endpoint
 * @param {string} endpoint - The API endpoint
 * @returns {Promise<any>} - Parsed JSON response
 */
export async function apiGet(endpoint) {
  const response = await apiRequest(endpoint)
  
  if (!response.ok) {
    throw new Error(`API request failed: ${response.status} ${response.statusText}`)
  }
  
  // Check if response is actually JSON
  const contentType = response.headers.get('content-type')
  if (contentType && contentType.includes('application/json')) {
    return await response.json()
  } else {
    // If not JSON, throw an error with more context
    const text = await response.text()
    throw new Error(`Expected JSON response but got ${contentType}: ${text.substring(0, 100)}...`)
  }
}

/**
 * Makes a POST request to an API endpoint
 * @param {string} endpoint - The API endpoint
 * @param {any} data - Data to send in the request body
 * @returns {Promise<any>} - Parsed JSON response or response object
 */
export async function apiPost(endpoint, data) {
  const response = await apiRequest(endpoint, {
    method: 'POST',
    body: JSON.stringify(data)
  })
  
  if (!response.ok) {
    throw new Error(`API request failed: ${response.status} ${response.statusText}`)
  }
  
  // Some endpoints might not return JSON
  const contentType = response.headers.get('content-type')
  if (contentType && contentType.includes('application/json')) {
    return await response.json()
  }
  
  return response
}

/**
 * Makes a PUT request to an API endpoint
 * @param {string} endpoint - The API endpoint
 * @param {any} data - Data to send in the request body
 * @returns {Promise<any>} - Parsed JSON response or response object
 */
export async function apiPut(endpoint, data) {
  const response = await apiRequest(endpoint, {
    method: 'PUT',
    body: JSON.stringify(data)
  })
  
  if (!response.ok) {
    throw new Error(`API request failed: ${response.status} ${response.statusText}`)
  }
  
  // Some endpoints might not return JSON
  const contentType = response.headers.get('content-type')
  if (contentType && contentType.includes('application/json')) {
    return await response.json()
  }
  
  return response
}