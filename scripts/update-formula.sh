#!/bin/bash
set -euo pipefail

# This script updates the Homebrew formula with the latest release information

# Get the latest release tag
echo "Fetching latest release information..."
LATEST_TAG=$(gh release view --json tagName -q .tagName)
VERSION=${LATEST_TAG#v}

echo "Latest version: $VERSION"

# Create temp directory
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

# Download release artifacts
echo "Downloading release artifacts..."
gh release download "$LATEST_TAG" --pattern "*.tar.gz" --pattern "checksums.txt" --dir "$TEMP_DIR"

# Extract checksums
echo "Extracting checksums..."
ARM64_CHECKSUM=$(grep "darwin_arm64.tar.gz" "$TEMP_DIR/checksums.txt" | cut -d' ' -f1)
AMD64_CHECKSUM=$(grep "darwin_x86_64.tar.gz" "$TEMP_DIR/checksums.txt" | cut -d' ' -f1)

echo "ARM64 checksum: $ARM64_CHECKSUM"
echo "AMD64 checksum: $AMD64_CHECKSUM"

# Update the formula using the template
echo "Updating Formula..."
sed -e "s/{{ .Version }}/$VERSION/g" \
    -e "s/{{ .DarwinArm64Checksum }}/$ARM64_CHECKSUM/g" \
    -e "s/{{ .DarwinAmd64Checksum }}/$AMD64_CHECKSUM/g" \
    Formula/macos-notify-bridge.rb.tmpl > Formula/macos-notify-bridge.rb

echo "Formula updated successfully!"