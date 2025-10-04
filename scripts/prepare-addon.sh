#!/bin/bash
# Build script for local development with pre-compiled binaries

set -e

# Default to local architecture if not specified
TARGET_ARCH=${1:-$(uname -m)}
case $TARGET_ARCH in
  x86_64) TARGET_ARCH="amd64" ;;
  aarch64) TARGET_ARCH="arm64" ;;
  amd64|arm64) ;; # already correct
  *) echo "Unsupported architecture: $TARGET_ARCH"; exit 1 ;;
esac

echo "Building for architecture: $TARGET_ARCH"

# Clean up old binaries
rm -f addons/gateway/gateway addons/gateway/gateway-*
rm -f addons/mqtt-bridge/mqtt-bridge addons/mqtt-bridge/mqtt-bridge-*
rm -rf addons/gateway/dist

# Build Go binaries with final names
echo "Building Go binaries..."
CGO_ENABLED=0 GOOS=linux GOARCH=$TARGET_ARCH go build -o addons/gateway/gateway ./go/tools/gateway/
CGO_ENABLED=0 GOOS=linux GOARCH=$TARGET_ARCH go build -o addons/mqtt-bridge/mqtt-bridge ./go/tools/mqtt-bridge/

# Build web assets for gateway
echo "Building web assets..."
cd web/auth
npm ci
npm run build
cp -r dist ../../addons/gateway/
cd ../..

echo "Build completed successfully for $TARGET_ARCH!"
echo ""
echo "To test locally:"
echo "  docker build -t test-gateway ./addons/gateway/"
echo "  docker build -t test-mqtt-bridge ./addons/mqtt-bridge/"