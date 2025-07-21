#!/usr/bin/env bash
set -euo pipefail

APP_NAME="MacOS Notify Bridge"
# First argument is icon path, second (optional) is app directory location
ICON_SOURCE="${1}"
DEST_DIR="${2}"
BUNDLE_ID="com.ahacop.macos-notify-bridge"

# Create app bundle in a temporary directory first
TEMP_DIR=$(mktemp -d)
APP_DIR="${TEMP_DIR}/${APP_NAME}.app"

echo "Setting up ${APP_NAME}.app bundle..."

# Create app bundle structure
mkdir -p "${APP_DIR}/Contents/"{MacOS,Resources}

# Create dummy executable
cat >"${APP_DIR}/Contents/MacOS/${APP_NAME}" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
chmod +x "${APP_DIR}/Contents/MacOS/${APP_NAME}"

# Create Info.plist
cat >"${APP_DIR}/Contents/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleName</key>
    <string>${APP_NAME}</string>
    <key>CFBundleDisplayName</key>
    <string>${APP_NAME}</string>
    <key>CFBundleIdentifier</key>
    <string>${BUNDLE_ID}</string>
    <key>CFBundleVersion</key>
    <string>1.0</string>
    <key>CFBundleShortVersionString</key>
    <string>1.0</string>
    <key>CFBundleExecutable</key>
    <string>${APP_NAME}</string>
    <key>CFBundleIconFile</key>
    <string>AppIcon</string>
    <key>LSUIElement</key>
    <true/>
    <key>NSHighResolutionCapable</key>
    <true/>
</dict>
</plist>
EOF

# Convert PNG to ICNS if source icon exists
if [ -f "$ICON_SOURCE" ]; then
	echo "Creating icon from ${ICON_SOURCE}..."

	# Create iconset directory
	ICONSET_DIR=$(mktemp -d)
	ICONSET="${ICONSET_DIR}/AppIcon.iconset"
	mkdir -p "$ICONSET"

	# Generate various icon sizes
	sips -z 16 16 "$ICON_SOURCE" --out "${ICONSET}/icon_16x16.png" >/dev/null 2>&1
	sips -z 32 32 "$ICON_SOURCE" --out "${ICONSET}/icon_16x16@2x.png" >/dev/null 2>&1
	sips -z 32 32 "$ICON_SOURCE" --out "${ICONSET}/icon_32x32.png" >/dev/null 2>&1
	sips -z 64 64 "$ICON_SOURCE" --out "${ICONSET}/icon_32x32@2x.png" >/dev/null 2>&1
	sips -z 128 128 "$ICON_SOURCE" --out "${ICONSET}/icon_128x128.png" >/dev/null 2>&1
	sips -z 256 256 "$ICON_SOURCE" --out "${ICONSET}/icon_128x128@2x.png" >/dev/null 2>&1
	sips -z 256 256 "$ICON_SOURCE" --out "${ICONSET}/icon_256x256.png" >/dev/null 2>&1
	sips -z 512 512 "$ICON_SOURCE" --out "${ICONSET}/icon_256x256@2x.png" >/dev/null 2>&1
	sips -z 512 512 "$ICON_SOURCE" --out "${ICONSET}/icon_512x512.png" >/dev/null 2>&1
	sips -z 1024 1024 "$ICON_SOURCE" --out "${ICONSET}/icon_512x512@2x.png" >/dev/null 2>&1

	# Convert to icns
	iconutil -c icns "$ICONSET" -o "${APP_DIR}/Contents/Resources/AppIcon.icns"

	# Cleanup
	rm -rf "$ICONSET_DIR"

	echo "Icon created successfully"
else
	echo "Warning: Icon source '${ICON_SOURCE}' not found"
fi

# Move the app bundle to the destination directory
if [ -n "${DEST_DIR}" ]; then
	echo "Moving app bundle to ${DEST_DIR}..."
	if ! mv "${APP_DIR}" "${DEST_DIR}/" 2>/dev/null; then
		# Try with sudo if regular move fails
		if [ -t 0 ]; then
			echo "Permission denied. Trying with sudo..."
			sudo mv "${APP_DIR}" "${DEST_DIR}/"
		else
			echo "Error: Could not move app bundle to ${DEST_DIR} - insufficient permissions"
			echo "The app bundle remains at: ${APP_DIR}"
			exit 1
		fi
	fi
	FINAL_APP_PATH="${DEST_DIR}/${APP_NAME}.app"
else
	# No destination specified, leave in temp directory
	FINAL_APP_PATH="${APP_DIR}"
	echo "Warning: No destination directory specified, app bundle created at ${APP_DIR}"
fi

# Cleanup temp directory if we successfully moved the app
if [ "${FINAL_APP_PATH}" != "${APP_DIR}" ]; then
	rm -rf "${TEMP_DIR}"
fi

echo "${APP_NAME}.app bundle created at ${FINAL_APP_PATH}"
echo "You can now use -sender ${BUNDLE_ID} with terminal-notifier"
