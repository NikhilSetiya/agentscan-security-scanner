#!/bin/bash
# Clean go.mod for DigitalOcean deployment

# Remove toolchain directive and fix version format
sed -i '' '/^toolchain/d' go.mod
sed -i '' 's/go 1.23.0/go 1.23/g' go.mod
sed -i '' 's/go 1.23.1/go 1.23/g' go.mod

echo "go.mod cleaned for DigitalOcean deployment"