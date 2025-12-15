# InfluxDB Auto-Detection Implementation

## Overview

The gateway now automatically detects if an InfluxDB add-on is installed and running in Home Assistant during startup. This eliminates the need for manual configuration.

## Configuration

### InfluxDB Credentials

The InfluxDB add-on requires authentication, but credentials are not exposed through the Supervisor API for security reasons. You need to configure them manually in the **Gateway add-on settings**:

1. Go to **Settings** → **Add-ons** → **Home Assistant Gateway**
2. Click on the **Configuration** tab
3. Enter your InfluxDB credentials:
   ```yaml
   influx_db_user: your_username
   influx_db_password: your_password
   ```
4. Click **Save** and restart the gateway add-on

The gateway will automatically detect the running InfluxDB add-on and use these credentials to send metrics.

### How It Works

### 1. Startup Detection

When the gateway starts, it:
- Checks if running in Home Assistant (via `SUPERVISOR_TOKEN` environment variable)
- Queries the Supervisor API for all installed add-ons
- Looks for known InfluxDB add-ons by:
  - Checking against known slugs (`a0d7b954_influxdb`, `influxdb`, `local_influxdb`, etc.)
  - Searching for "influx" in add-on names (case-insensitive)

### 2. Configuration Retrieval

If an InfluxDB add-on is found and running, the gateway:
- Retrieves network configuration (hostname and port)
- Extracts credentials from the add-on's options:
  - Username (`username` or `user`)
  - Password (`password` or `pass`)
  - Database name (`database` or `db`, defaults to `homeassistant`)

### 3. Status Exposure

A new API endpoint provides the InfluxDB integration status:
```
GET /api/influxdb/status
```

Response when InfluxDB is detected:
```json
{
  "result": "ok",
  "data": {
    "enabled": true,
    "name": "InfluxDB",
    "slug": "a0d7b954_influxdb",
    "url": "http://a0d7b954-influxdb:8086",
    "database": "homeassistant",
    "username": "homeassistant",
    "message": "InfluxDB integration active"
  }
}
```

Response when not detected:
```json
{
  "result": "ok",
  "data": {
    "enabled": false,
    "message": "No InfluxDB add-on detected"
  }
}
```

## Implementation Details

### New Files

1. **`go/services/homeassistant/influxdb.go`**
   - `InfluxDBConfig` struct to store detected configuration
   - `DetectInfluxDB()` method on `SupervisorClient`
   - Helper functions for slug-to-hostname conversion
   - Known InfluxDB add-on slug list

2. **`examples/test-influxdb-detection.go`**
   - Standalone test program to verify detection logic
   - Shows detection results with masked password
   - Explains how the detection works

### Modified Files

1. **`go/gateway/gateway.go`**
   - Added `influxDBConfig` field to `Gateway` struct
   - Added `detectInfluxDB()` method
   - Integrated detection into `Start()` method
   - Logs detection results at startup

2. **`go/gateway/endpoints.go`**
   - Added `/api/influxdb/status` endpoint
   - `GET_InfluxDBStatus()` handler returns sanitized config

## Console Output

When the gateway starts:

### With InfluxDB detected:
```
✅ InfluxDB detected: InfluxDB
   URL: http://a0d7b954-influxdb:8086
   Database: homeassistant
   Username: homeassistant
```

### Without InfluxDB:
```
ℹ️  No InfluxDB add-on detected
```

### Outside Home Assistant:
```
InfluxDB detection skipped: not running in Home Assistant environment
```

## Testing

Run the detection test program:
```bash
go run examples/test-influxdb-detection.go
```

Run the startup metric test program:
```bash
go run examples/test-startup-metric.go
```

## Metrics Collected

### 1. Startup Events

**Measurement**: `gateway_events`  
**Tags**: `service=gateway`, `event=startup`  
**Value**: `1` (counter)

This metric is sent every time the gateway starts up, allowing you to:
- Track how often the gateway restarts
- Identify stability issues
- Monitor deployment frequency

**InfluxDB Queries**:
```sql
-- View all startup events
SELECT * FROM gateway_events WHERE event='startup'

-- Count restarts per hour
SELECT COUNT(value) FROM gateway_events 
  WHERE event='startup' 
  GROUP BY time(1h)

-- Count restarts per day
SELECT COUNT(value) FROM gateway_events 
  WHERE event='startup' 
  GROUP BY time(1d)
```

## Next Steps

The foundation is now in place. Additional metrics to implement:

1. **Metrics Collection**: Define what metrics to track (requests, response times, errors, etc.)
2. **InfluxDB Client**: Initialize the InfluxDB client with detected credentials
3. **Metric Writing**: Implement background goroutine to batch and send metrics
4. **Configuration Options**: Add optional user overrides in case auto-detection doesn't work
5. **Frontend UI**: Display InfluxDB status and metrics configuration in the gateway UI

## Security Considerations

- The gateway requires `hassio_role: manager` (already configured) to access add-on information
- Passwords are never logged or exposed via API endpoints
- Only sanitized configuration is returned to clients
- Detection only runs when `SUPERVISOR_TOKEN` is present (Home Assistant environment)
