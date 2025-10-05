#!/bin/bash
# Build script for local development with multi-arch binaries

set -e

echo "Building binaries for all supported architectures..."

# Clean up old binaries
rm -f addons/gateway/gateway addons/gateway/gateway-*
rm -f addons/mqtt-bridge/mqtt-bridge addons/mqtt-bridge/mqtt-bridge-*
rm -rf addons/gateway/dist

# Build Go binaries for all architectures (for platform selector)
echo "Building AMD64 binaries..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o addons/gateway/gateway-amd64 ./go/tools/gateway/
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o addons/mqtt-bridge/mqtt-bridge-amd64 ./go/tools/mqtt-bridge/

echo "Building ARM64 binaries..."
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o addons/gateway/gateway-arm64 ./go/tools/gateway/
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o addons/mqtt-bridge/mqtt-bridge-arm64 ./go/tools/mqtt-bridge/

# Build web assets for gateway
echo "Building web assets..."
cd web/auth
npm ci
npm run build
cp -r dist ../../addons/gateway/
cd ../..

echo "Build completed successfully!"
echo ""
echo "Built binaries:"
echo "  Gateway: gateway-amd64, gateway-arm64, gateway (default)"
echo "  MQTT Bridge: mqtt-bridge-amd64, mqtt-bridge-arm64, mqtt-bridge (default)"
echo ""
echo "To test locally:"
echo "  docker build -t test-gateway ./addons/gateway/"
echo "  docker build -t test-mqtt-bridge ./addons/mqtt-bridge/"
echo ""
echo "The Docker build will automatically select the correct binary for the target architecture."