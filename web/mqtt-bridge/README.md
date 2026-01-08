# MQTT Bridge Web UI

A Vue.js web interface for configuring and managing the MQTT Bridge addon.

## Development

```bash
# Install dependencies
npm install

# Start development server
npm run dev

# Build for production
npm run build
```

## Features

- MQTT configuration management (coming soon)
- Real-time monitoring (coming soon)
- Bridge status and diagnostics (coming soon)

## API Endpoints

The web UI communicates with the MQTT Bridge backend via REST API endpoints:

- `GET /api/status` - Get bridge status
- `GET /api/config` - Get current configuration  
- `PUT /api/config` - Update configuration
- `GET /api/connections` - Get active connections
- `POST /api/restart` - Restart bridge services

(Note: API endpoints are planned and not yet implemented)