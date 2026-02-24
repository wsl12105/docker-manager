#!/bin/bash

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 获取版本信息
VERSION=$(grep 'Version =' internal/version/version.go | awk -F'"' '{print $2}')
BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

echo -e "${GREEN}Building Docker Manager ${VERSION}${NC}"
echo -e "Build date: ${BUILD_DATE}"
echo -e "Git commit: ${GIT_COMMIT}"
echo ""

# 创建输出目录
mkdir -p dist

# 编译选项
LDFLAGS="-s -w \
-X github.com/wsl12105/docker-manager/internal/version.BuildDate=${BUILD_DATE} \
-X github.com/wsl12105/docker-manager/internal/version.GitCommit=${GIT_COMMIT}"

# 常用平台列表（不含Windows）
PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "linux/386"
    "linux/arm"
    "darwin/amd64"
    "darwin/arm64"
    "freebsd/amd64"
    "freebsd/arm64"
)

# 构建函数
build() {
    local GOOS=$1
    local GOARCH=$2
    local OUTPUT="dist/dm-${VERSION}-${GOOS}-${GOARCH}"
    
    echo -e "${YELLOW}Building for ${GOOS}/${GOARCH}...${NC}"
    
    GOOS=$GOOS GOARCH=$GOARCH CGO_ENABLED=0 go build \
        -ldflags="$LDFLAGS" \
        -trimpath \
        -o "$OUTPUT" \
        ./cmd
    
    if [ -f "$OUTPUT" ]; then
        SIZE=$(ls -lh "$OUTPUT" | awk '{print $5}')
        echo -e "${GREEN}  ✅ ${OUTPUT} (${SIZE})${NC}"
        
        # 如果是当前平台，创建软链接
        if [ "$GOOS" = "$(go env GOOS)" ] && [ "$GOARCH" = "$(go env GOARCH)" ]; then
            cd dist
            ln -sf "$(basename "${OUTPUT}")" dm-latest 2>/dev/null || cp "$(basename "${OUTPUT}")" dm-latest
            cd ..
        fi
    else
        echo -e "${RED}  ❌ Build failed${NC}"
    fi
}

# 显示用法
show_usage() {
    echo "Usage: $0 [platform]"
    echo ""
    echo "Platforms:"
    echo "  all       - Build all platforms (Linux/macOS/BSD)"
    echo "  local     - Build current platform (default)"
    echo "  linux     - Build all Linux platforms"
    echo "  darwin    - Build all macOS platforms"
    echo "  bsd       - Build all BSD platforms"
    echo "  <os/arch> - Build specific platform (e.g., linux/arm64)"
    echo ""
    echo "Examples:"
    echo "  $0                 # Build current platform"
    echo "  $0 local           # Build current platform"
    echo "  $0 all             # Build all platforms"
    echo "  $0 linux           # Build all Linux platforms"
    echo "  $0 linux/arm64     # Build Linux ARM64"
    echo "  $0 darwin/arm64    # Build macOS Apple Silicon"
    echo ""
    echo "Note: Windows platform is not supported"
}

# 根据参数执行
case $1 in
    ""|local)
        # 构建当前平台
        GOOS=$(go env GOOS)
        GOARCH=$(go env GOARCH)
        
        # 检查是否尝试构建Windows
        if [ "$GOOS" = "windows" ]; then
            echo -e "${RED}Error: Windows is not supported${NC}"
            echo "Please run on Linux or macOS"
            exit 1
        fi
        
        build "$GOOS" "$GOARCH"
        echo ""
        echo -e "${GREEN}✅ Build completed!${NC}"
        echo -e "Run: ${BLUE}./dist/dm-latest${NC}"
        ;;
    all)
        # 构建所有平台
        for platform in "${PLATFORMS[@]}"; do
            GOOS=${platform%/*}
            GOARCH=${platform#*/}
            build "$GOOS" "$GOARCH"
        done
        echo ""
        echo -e "${GREEN}✅ All builds completed!${NC}"
        ;;
    linux)
        # 构建所有Linux平台
        build "linux" "amd64"
        build "linux" "arm64"
        build "linux" "386"
        build "linux" "arm"
        build "linux" "riscv64" 2>/dev/null || true
        build "linux" "ppc64le" 2>/dev/null || true
        build "linux" "s390x" 2>/dev/null || true
        echo ""
        echo -e "${GREEN}✅ Linux builds completed!${NC}"
        ;;
    darwin|macos)
        # 构建所有macOS平台
        build "darwin" "amd64"
        build "darwin" "arm64"
        echo ""
        echo -e "${GREEN}✅ macOS builds completed!${NC}"
        ;;
    bsd|freebsd)
        # 构建所有BSD平台
        build "freebsd" "amd64"
        build "freebsd" "arm64"
        build "openbsd" "amd64"
        build "openbsd" "arm64"
        build "netbsd" "amd64"
        build "netbsd" "arm64"
        echo ""
        echo -e "${GREEN}✅ BSD builds completed!${NC}"
        ;;
    */*)
        # 构建指定平台
        if [[ $1 =~ ^[^/]+/[^/]+$ ]]; then
            GOOS=${1%/*}
            GOARCH=${1#*/}
            
            # 检查是否尝试构建Windows
            if [ "$GOOS" = "windows" ]; then
                echo -e "${RED}Error: Windows platform is not supported${NC}"
                exit 1
            fi
            
            build "$GOOS" "$GOARCH"
            echo ""
            echo -e "${GREEN}✅ Build completed!${NC}"
        else
            echo -e "${RED}Error: Invalid platform format${NC}"
            show_usage
            exit 1
        fi
        ;;
    -h|--help)
        show_usage
        ;;
    *)
        echo -e "${RED}Unknown option: $1${NC}"
        show_usage
        exit 1
        ;;
esac
