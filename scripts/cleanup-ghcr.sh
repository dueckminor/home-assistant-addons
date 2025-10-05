#!/bin/bash
# Script to clean up unreferenced untagged images from GitHub Container Registry

set -euo pipefail

PACKAGE_NAME=${1:-}
if [ -z "$PACKAGE_NAME" ]; then
    echo "Usage: $0 <package-name>"
    echo "Example: $0 home-assistant-addons-gateway"
    exit 1
fi

echo "Cleaning up package: $PACKAGE_NAME"

# Get GitHub username
GITHUB_USER=$(gh api user --jq '.login')
echo "GitHub user: $GITHUB_USER"

# Get all tagged versions
TAGGED_VERSIONS=$(gh api --paginate "/user/packages/container/$PACKAGE_NAME/versions" --jq '.[] | select(.metadata.container.tags | length > 0) | .id')

# Get all referenced digests from tagged versions
echo "Finding referenced digests from tagged versions..."
REFERENCED_DIGESTS=""
for version_id in $TAGGED_VERSIONS; do
    echo "Checking tagged version: $version_id"
    
    # Get the tags for this version
    TAGS=$(gh api "/user/packages/container/$PACKAGE_NAME/versions/$version_id" --jq '.metadata.container.tags[]')
    
    for tag in $TAGS; do
        echo "  Inspecting tag: $tag"
        
        # Get manifest and extract referenced digests
        MANIFEST=$(docker manifest inspect "ghcr.io/$GITHUB_USER/$PACKAGE_NAME:$tag" 2>/dev/null || echo "{}")
        
        if [ "$MANIFEST" != "{}" ]; then
            # Extract digests from multi-arch manifest or single manifest
            DIGESTS=$(echo "$MANIFEST" | jq -r '
                if .manifests then 
                    .manifests[].digest 
                else 
                    .config.digest // empty,
                    (.layers[]?.digest // empty)
                end' 2>/dev/null || echo "")
            
            REFERENCED_DIGESTS="$REFERENCED_DIGESTS $DIGESTS"
            echo "    Found digests: $(echo $DIGESTS | tr '\n' ' ')"
        fi
    done
done

# Remove duplicates and clean up
REFERENCED_DIGESTS=$(echo $REFERENCED_DIGESTS | tr ' ' '\n' | sort -u | grep -v '^$' || echo "")

echo ""
echo "All referenced digests:"
echo "$REFERENCED_DIGESTS"
echo ""

# Find untagged versions that are not referenced
echo "Finding untagged versions to delete..."
UNTAGGED_VERSIONS=$(gh api --paginate "/user/packages/container/$PACKAGE_NAME/versions" --jq '.[] | select(.metadata.container.tags | length == 0) | .id')

SAFE_TO_DELETE=""
for version_id in $UNTAGGED_VERSIONS; do
    # Get the digest for this version
    VERSION_DIGEST=$(gh api "/user/packages/container/$PACKAGE_NAME/versions/$version_id" --jq '.name')
    
    # Check if this digest is referenced by any tagged version
    if echo "$REFERENCED_DIGESTS" | grep -qF "$VERSION_DIGEST"; then
        echo "KEEP: Version $version_id (digest: $VERSION_DIGEST) - referenced by tagged version"
    else
        echo "DELETE: Version $version_id (digest: $VERSION_DIGEST) - not referenced"
        SAFE_TO_DELETE="$SAFE_TO_DELETE $version_id"
    fi
done

echo ""
if [ -n "$SAFE_TO_DELETE" ]; then
    echo "Versions safe to delete: $SAFE_TO_DELETE"
    echo ""
    read -p "Do you want to delete these unreferenced versions? (y/N): " -n 1 -r
    echo
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        for version_id in $SAFE_TO_DELETE; do
            echo "Deleting version: $version_id"
            gh api --method DELETE "/user/packages/container/$PACKAGE_NAME/versions/$version_id"
        done
        echo "Cleanup completed!"
    else
        echo "Cleanup cancelled."
    fi
else
    echo "No unreferenced versions found - nothing to delete!"
fi