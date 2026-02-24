#!/bin/bash


VERSION=$(grep 'Version =' internal/version/version.go | awk -F'"' '{print $2}')
BUILD_DATE=$(date +%Y-%m-%dT%H:%M:%S%z)
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

echo "Building Docker Manager ${VERSION}..."
echo "Build date: ${BUILD_DATE}"
echo "Git commit: ${GIT_COMMIT}"


go build -ldflags "\
-s -w \
-X github.com/wsl12105/docker-manager/internal/version.BuildDate=${BUILD_DATE} \
-X github.com/wsl12105/docker-manager/internal/version.GitCommit=${GIT_COMMIT}" \
-o dm ./cmd

if [ $? -eq 0 ]; then
    SIZE=$(ls -lh dm | awk '{print $5}')
    echo "✅ Build successful! (Size: ${SIZE})"
    echo "Run ./dm to start"
else
    echo "❌ Build failed"
    exit 1
fi
