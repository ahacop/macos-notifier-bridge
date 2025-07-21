# Release Process

This document describes how to create a new release for macOS Notify Bridge.

## Automated Release Process

The release process is fully automated using GitHub Actions and goreleaser. When you push a version tag, the following happens automatically:

1. Tests are run to ensure code quality
2. Goreleaser creates binaries for Intel and Apple Silicon Macs
3. A GitHub release is created with the binaries
4. The Homebrew formula is automatically updated with the new version and checksums
5. The updated formula is committed back to the main branch

## Creating a Release

1. **Update version in code** (if needed):
   - The version is automatically set from git tags during build
   - No manual version updates needed in code

2. **Create and push a version tag**:
   ```bash
   # Create an annotated tag
   git tag -a v0.2.0 -m "Release v0.2.0: Add feature X"
   
   # Push the tag to GitHub
   git push origin v0.2.0
   ```

3. **Monitor the release**:
   - Go to the [Actions tab](https://github.com/ahacop/macos-notify-bridge/actions) on GitHub
   - Watch the "Release" workflow progress
   - The workflow will:
     - Build and test the code
     - Create binaries for both architectures
     - Create a GitHub release
     - Update the Homebrew formula
     - Commit the updated formula to main

4. **Verify the release**:
   - Check the [Releases page](https://github.com/ahacop/macos-notify-bridge/releases)
   - Verify the Homebrew formula was updated in `Formula/macos-notify-bridge.rb`
   - Test installation: `brew install ./Formula/macos-notify-bridge.rb`

## Manual Formula Update (if needed)

If the automatic update fails, you can manually update the formula:

```bash
# Run the update script
./scripts/update-formula.sh

# Commit the changes
git add Formula/macos-notify-bridge.rb
git commit -m "Update Homebrew Formula to version vX.Y.Z"
git push origin main
```

## Version Naming

Follow semantic versioning:
- `vX.Y.Z` format (e.g., v0.2.0, v1.0.0)
- Major version (X): Breaking changes
- Minor version (Y): New features, backwards compatible
- Patch version (Z): Bug fixes, backwards compatible

## Troubleshooting

If the release workflow fails:

1. **Check GitHub Actions logs** for specific error messages
2. **Verify the tag format** - must match `v*.*.*`
3. **Ensure goreleaser config is valid**: `make release-dry`
4. **Check permissions** - the workflow needs write access to the repo

## Local Testing

To test the release process locally without publishing:

```bash
# Test goreleaser configuration
make release-dry

# This creates a local release in dist/ directory
```