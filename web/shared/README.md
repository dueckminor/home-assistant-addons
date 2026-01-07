# Shared Web Utilities

This directory contains shared utilities used across all Home Assistant addon web interfaces (gateway, mqtt-bridge, security, and auth).

## Home Assistant API Utilities

Location: `utils/homeassistant.js`

These utilities handle the complexities of making API calls and WebSocket connections when running as a Home Assistant addon with ingress enabled.

### Problem

When a web application runs as a Home Assistant addon with ingress enabled, all URLs are prefixed with `/api/hassio_ingress/<token>/`. For example:

```
Normal URL:        https://homeassistant.local/api/topics
Ingress URL:       https://homeassistant.local/api/hassio_ingress/abc123/api/topics
```

Without proper handling, API calls and WebSocket connections will fail because they're trying to connect to the wrong path.

### Solution

The utilities automatically detect the ingress path and construct the correct URLs.

### Functions

#### `getBaseUrl()`

Returns the base URL for API calls, automatically detecting Home Assistant ingress paths.

```javascript
import { getBaseUrl } from '../../../shared/utils/homeassistant.js'

const baseUrl = getBaseUrl()
// Development: http://localhost:3000
// HA Ingress: https://homeassistant.local/api/hassio_ingress/abc123
```

#### `getWebSocketUrl(path)`

Returns the complete WebSocket URL, handling ingress paths and protocol selection (ws/wss).

```javascript
import { getWebSocketUrl } from '../../../shared/utils/homeassistant.js'

const wsUrl = getWebSocketUrl('/api/topics?stream=true')
// Development: ws://localhost:3000/api/topics?stream=true
// HA Ingress: wss://homeassistant.local/api/hassio_ingress/abc123/api/topics?stream=true
```

#### `apiRequest(endpoint, options)`

Make a fetch request with proper path handling.

```javascript
import { apiRequest } from '../../../shared/utils/homeassistant.js'

const response = await apiRequest('topics', {
  method: 'GET',
  headers: { 'Custom-Header': 'value' }
})
```

#### `apiGet(endpoint)`, `apiPost(endpoint, data)`, `apiPut(endpoint, data)`, `apiDelete(endpoint)`

Convenience wrappers for common HTTP methods that automatically parse JSON responses.

```javascript
import { apiGet, apiPost } from '../../../shared/utils/homeassistant.js'

// GET request
const topics = await apiGet('topics')

// POST request
const result = await apiPost('domains', { name: 'example.com' })
```

## Usage in Each Addon

### Gateway

The gateway already had its own `utils/api.js` which now re-exports the shared utilities for backward compatibility:

```javascript
import { apiGet, apiPost } from '../utils/api.js'
```

### MQTT Bridge

Import directly from the composables:

```javascript
import { getBaseUrl, getWebSocketUrl } from '../../../shared/utils/homeassistant.js'
```

### Security

Import in Vue components:

```javascript
import { getBaseUrl } from '../../shared/utils/homeassistant.js'
```

## Development

When adding new shared utilities:

1. Add them to the appropriate file in `web/shared/utils/`
2. Export them with proper JSDoc documentation
3. Update this README with usage examples
4. Consider backward compatibility with existing code

## Testing

Test with both development and Home Assistant ingress environments:

- **Development**: Access via `http://localhost:<port>`
- **Home Assistant**: Access via the addon's ingress URL in Home Assistant

The utilities should work correctly in both environments without code changes.
