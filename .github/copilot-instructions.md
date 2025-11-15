# Architecture

The repo implements two addons for Home Assistant: the Gateway and the 
MQTT Bridge. The Gateway addon serves as a reverse proxy and DNS server, 
while the MQTT Bridge facilitates communication between Home Assistant and
MQTT brokers.

## Gateway Addon

The gateway addon listens on ports 25, 80 and 443 and normally the router
(like a Fritz!Box) forwards these ports from the public internet to the
gateway addon. The gateway addon then inspects incoming requests and redirects
all HTTP request (port 80) to HTTPS (port 443). For HTTPS requests the
gateway addon checks the SNI header (TLS Server Name Indication) to determine
the requested hostname. If the hostname is known, the gateway addon routes the
request to the appropriate internal service based on the configured routes.
If the hostname is unknown, the gateway simply closes the connection
(without any response).

Port 25 is used for DNS-01 challenge responses to automatically issue and
renew SSL certificates via Let's Encrypt. Port 80 might also used for the
HTTP-01 challenge responses. But as only wildcard certificates are requested,
currently only DNS-01 is supported.

The gateway addon also provides a web interface to manage domains, routes,
DNS settings, and users.

On port 443 the gateway addon implements different kind of routing:

- it terminates HTTPS connections for configured domains and routes them to
  internal services. It is possible to configure additional authentication
  and also modify request headers.
- it forwards HTTPS connections directly to another target which is responsible
  for terminating the HTTPS connection (TCP passthrough)

To be able to answer DNS requests for the configured domains, the gateway addon
also implements a DNS server. The DNS server is used to resolve the configured
domains to the public IP address of the gateway addon. It is also used to
respond to DNS-01 challenge requests from Let's Encrypt.

To ensure that all requests reach the gateway addon, the DNS server has to
return the public IPv4 address of the router and the IPv6 address of the
home-assistant instance.

For example. If you own the example.com domain and want to use the gateway addon
for routing, you have to add DNS records for subdomains like `home.example.com`
and let them point to the public IP address of your internet router.

The gateway will then answer all DNS requests for `home.example.com` and all
hosts in this domain with the public IP address of your router.

### Gateway Addon Components

#### Backend (`go/gateway`)

The backend of the gateway addon is implemented in Go. It implements the 
reverse proxy, DNS server, SSL certificate management, OAuth authentication,
and the REST API for the web interface.

#### Configuration Frontend (`web/gateway`)

The dist files are embedded into the Go binary and served by the backend.
The frontend is implemented in Vue.js and provides the web interface to manage
the gateway addon.

#### Authentication Frontend (`web/auth`)

The dist files are embedded into the Go binary and served by the backend.
The frontend is implemented in Vue.js and provides the web interface to
authenticate users via OAuth.

## MQTT Bridge Addon

TBD

# Testing

## Gateway Addon

The gateway addon can be tested on a MacOS or Linux development machine.
To do this, the gateway addon running on the Home Assistant instance has to be
configured to forward all requests for a specific test domain 
(like `dev.example.com`) to the development machine. This will forward
HTTP/HTTPS/DNS requests to the development machine.

### Development Environment Setup

To test the gateway addon, use the development server setup:

```bash
# Start both frontend development servers
./scripts/serve start

# This starts:
# - Gateway frontend: http://localhost:3001 (Vue.js + Vuetify)
# - Auth frontend: http://localhost:3000 (Vue.js + Vuetify)
```

Alternatively, the `.vscode/launch.json` configuration `Launch Gateway (serve)`
can be used. This configuration will start the gateway addon in debug mode and
attach the debugger. The frontend (both `gateway` and `auth`) is started in
watch mode to automatically rebuild the frontend on changes. Instead of serving
the embedded dist files, the backend serves the frontend from the local 
development server.

The gateway listens on the following ports:

- 8099: Management API/UI
- 10025: DNS server
- 10080: HTTP server
- 10443: HTTPS server

The frontend components on port 3000 and 3001 should not be directly accessed, 
as the backend serves the frontend on port 8099. If you would use them directly, 
it would not be possible to call the backend API.

To access the authentication interface, you have to use the configured
development domain. For example: https://auth.dev.example.com

### Testing Frontend Changes

1. **Live Development**: While the development server is running, changes to 
   frontend code will automatically be reflected in the browser via hot-reload
2. **Manual Testing**: Access the gateway interface at `http://localhost:8099`
   to test UI changes
3. **Build Testing**: Run `npm run build` in the respective web directory to 
   test production builds
4. **Backend Changes**: Require a restart of the gateway addon 
   (backend changes are not hot-reloaded)

### Frontend Structure

- **Gateway Frontend** (`web/gateway/`): Main configuration interface with 
  domain/route management, DNS settings, user management
- **Auth Frontend** (`web/auth/`): OAuth authentication interface for user login
- Both use Vue.js 3 + Vuetify 3 + Vite for optimal development experience


