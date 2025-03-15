#!/bin/bash

# 设置颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # 无颜色

# 版本号
VERSION="1.0.0"

# 检查前端文件是否存在
check_frontend_files() {
    echo -e "${BLUE}检查前端文件...${NC}"
    if [ ! -d "web/out" ]; then
        echo -e "${RED}错误: 前端文件目录 'web/out' 不存在!${NC}"
        echo -e "${YELLOW}提示: 请确保前端已构建并放置在正确位置，以便嵌入到二进制文件中。${NC}"
        return 1
    fi
    
    if [ ! -f "web/out/index.html" ]; then
        echo -e "${RED}错误: 前端主页文件 'web/out/index.html' 不存在!${NC}"
        echo -e "${YELLOW}提示: 请确保前端已正确构建。${NC}"
        return 1
    fi
    
    echo -e "${GREEN}前端文件检查通过。${NC}"
    return 0
}

# 创建输出目录
mkdir -p build

echo -e "${BLUE}开始编译 GhostEye v${VERSION}...${NC}"

# 默认编译当前系统架构
build_current() {
    echo -e "${GREEN}编译当前系统架构的二进制文件...${NC}"
    
    # 检查前端文件
    check_frontend_files || return 1
    
    # 执行go mod tidy整理依赖
    echo -e "${BLUE}执行 go mod tidy 整理依赖...${NC}"
    go mod tidy
    if [ $? -ne 0 ]; then
        echo -e "${RED}go mod tidy 执行失败！${NC}"
        return 1
    fi
    echo -e "${GREEN}依赖整理完成。${NC}"
    
    output_name="GhostEye"
    
    # 如果是Windows，添加.exe后缀
    if [[ "$OSTYPE" == "msys" || "$OSTYPE" == "win32" ]]; then
        output_name="GhostEye.exe"
    fi
    
    echo -e "${BLUE}正在编译...${NC}"
    go build -o "build/$output_name" -ldflags "-s -w" .
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}编译成功！二进制文件位于: build/$output_name${NC}"
        echo -e "${BLUE}提示: 直接运行 build/$output_name 即可启动服务，所有前端资源已嵌入。${NC}"
    else
        echo -e "${RED}编译失败！${NC}"
        exit 1
    fi
}

# 编译所有支持的平台
build_all() {
    echo -e "${BLUE}开始为所有支持的平台编译GhostEye...${NC}"
    
    # 检查前端文件
    check_frontend_files || return 1
    
    # 执行go mod tidy整理依赖
    echo -e "${BLUE}执行 go mod tidy 整理依赖...${NC}"
    go mod tidy
    if [ $? -ne 0 ]; then
        echo -e "${RED}go mod tidy 执行失败！${NC}"
        return 1
    fi
    echo -e "${GREEN}依赖整理完成。${NC}"
    
    # 定义要编译的平台列表
    platforms=("linux/amd64" "linux/386" "linux/arm64" "linux/arm")
    
    for platform in "${platforms[@]}"
    do
        platform_split=(${platform//\// })
        GOOS=${platform_split[0]}
        GOARCH=${platform_split[1]}
        
        output_name="ghosteye-$GOOS-$GOARCH"
        if [ $GOOS = "windows" ]; then
            output_name+=".exe"
        fi
        
        echo -e "${YELLOW}编译 $GOOS/$GOARCH...${NC}"
        env GOOS=$GOOS GOARCH=$GOARCH go build -o "build/$output_name" -ldflags "-s -w" .
        
        if [ $? -ne 0 ]; then
            echo -e "${RED}编译 $GOOS/$GOARCH 失败${NC}"
        else
            echo -e "${GREEN}编译 $GOOS/$GOARCH 成功${NC}"
        fi
    done
    
    echo -e "${GREEN}所有平台编译完成！${NC}"
    echo -e "${BLUE}提示: 所有二进制文件已包含嵌入式前端资源，可直接运行。${NC}"
}

# 清理编译文件
clean() {
    echo -e "${BLUE}清理编译文件...${NC}"
    rm -rf build
    echo -e "${GREEN}清理完成${NC}"
}

# 显示帮助信息
show_help() {
    echo "GhostEye 编译脚本 v${VERSION}"
    echo "用法: ./build.sh [选项]"
    echo ""
    echo "选项:"
    echo "  -c, --current   仅编译当前系统架构的二进制文件"
    echo "  -a, --all       编译所有支持的平台"
    echo "      --clean     清理编译文件"
    echo "  -h, --help      显示此帮助信息"
    echo ""
    echo "示例:"
    echo "  ./build.sh              # 默认编译当前系统架构"
    echo "  ./build.sh --all        # 编译所有支持的平台"
    echo "  ./build.sh --clean      # 清理编译文件"
}

# 解析命令行参数
if [[ $# -eq 0 ]]; then
    build_current
else
    case "$1" in
        -c|--current)
            build_current
            ;;
        -a|--all)
            build_all
            ;;
        --clean)
            clean
            ;;
        -h|--help)
            show_help
            ;;
        *)
            echo -e "${RED}未知选项: $1${NC}"
            show_help
            exit 1
            ;;
    esac
fi

exit 0 
