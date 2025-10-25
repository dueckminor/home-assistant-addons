# Add-on Discovery API Demo

This document demonstrates the new automatic add-on discovery feature that simplifies routing configuration by eliminating the need to manually enter complex target URIs like `http://a0d7b954-bitwarden:7277`.

## Implementation Overview

### Backend Components

1. **Supervisor API Client** (`go/services/homeassistant/addons.go`)
   - `SupervisorClient` struct with automatic token authentication
   - `GetRunningAddons()` - Returns running add-ons with network details
   - `GetAddonInfo(slug)` - Returns detailed info for a specific add-on
   - `AddonTarget` struct for simplified routing configuration

2. **Gateway API Endpoints** (`go/gateway/endpoints.go`)
   - `GET /api/addons/running` - Lists all running add-ons with network info
   - `GET /api/addons/{slug}` - Gets detailed info for a specific add-on

### API Usage Examples

#### Get Running Add-ons
```bash
GET /api/addons/running
```

**Response:**
```json
{
  "result": "ok",
  "data": [
    {
      "name": "Bitwarden",
      "slug": "a0d7b954-bitwarden",
      "description": "Open source password management solutions",
      "hostname": "a0d7b954-bitwarden",
      "port": 7277,
      "port_name": "7277/tcp",
      "url": "http://a0d7b954-bitwarden:7277"
    },
    {
      "name": "Node-RED",
      "slug": "a0d7b954-nodered", 
      "description": "Flow-based programming for the Internet of Things",
      "hostname": "a0d7b954-nodered",
      "port": 1880,
      "port_name": "1880/tcp", 
      "url": "http://a0d7b954-nodered:1880"
    }
  ]
}
```

#### Get Specific Add-on Info
```bash
GET /api/addons/a0d7b954-bitwarden
```

**Response:**
```json
{
  "result": "ok",
  "data": {
    "name": "Bitwarden",
    "slug": "a0d7b954-bitwarden",
    "description": "Open source password management solutions",
    "hostname": "a0d7b954-bitwarden",
    "port": 7277,
    "port_name": "7277/tcp",
    "url": "http://a0d7b954-bitwarden:7277"
  }
}
```

## Frontend Integration

### Before: Manual Configuration
Users had to manually enter complex target URIs:
```
Target URI: http://a0d7b954-bitwarden:7277
```

### After: Automatic Discovery
The frontend can now:
1. Call `/api/addons/running` to get available add-ons
2. Display a user-friendly dropdown with add-on names
3. Auto-populate the target URI when user selects an add-on

### Suggested UI Improvements

#### Route Configuration Form
```html
<!-- Instead of manual input -->
<input type="text" placeholder="http://a0d7b954-bitwarden:7277" />

<!-- Show user-friendly selection -->
<select name="addon-target">
  <option value="">Select an add-on...</option>
  <option value="http://a0d7b954-bitwarden:7277">Bitwarden</option>
  <option value="http://a0d7b954-nodered:1880">Node-RED</option>
</select>
```

#### Add-on Status Indicators
- ✅ Running and accessible
- 🔄 Starting up
- ❌ Stopped or error
- 📶 Network connectivity status

## Configuration Benefits

### User Experience
- **Simplified Setup**: No need to know internal hostnames or ports
- **Reduced Errors**: Eliminates typos in manual URI entry
- **Visual Clarity**: See available add-ons with friendly names
- **Auto-Discovery**: Automatically finds running services

### Reliability  
- **Real-time Status**: Only shows actually running add-ons
- **Network Validation**: Confirms add-ons are accessible
- **Error Handling**: Clear feedback when add-ons are unavailable

## Rating Impact

This feature directly addresses usability concerns that affect the add-on's rating:

- **Ease of Configuration**: Transforms complex manual setup into simple selection
- **User-Friendly Interface**: Makes routing accessible to non-technical users  
- **Reduced Support Burden**: Fewer configuration-related questions
- **Professional Experience**: Matches expectations from modern applications

The automatic discovery eliminates the main pain point identified: 
*"What makes it complicated is the required target URI: http://a0d7b954-bitwarden:7277"*

This should contribute to improving the add-on rating from 7 to 8 by significantly enhancing the user experience.