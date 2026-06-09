#!/bin/bash
# ==========================================
#  OpenIM gRPC 代码生成脚本 (Linux/macOS)
# ==========================================
# 该脚本用于从 proto 文件生成 Go gRPC 代码
# 需要提前安装 protoc 和相关插件

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "\n=========================================="
echo -e "  OpenIM gRPC 代码生成脚本"
echo -e "==========================================\n"

# 检查 protoc 是否安装
echo -e "${YELLOW}[1/4] 检查 protoc 安装...${NC}"
if ! command -v protoc &> /dev/null; then
    echo -e "${RED}  ✗ protoc 未安装或未添加到 PATH${NC}"
    echo -e "    请从 https://github.com/protocolbuffers/protobuf/releases 下载并安装 protoc"
    exit 1
else
    PROTOC_VERSION=$(protoc --version)
    echo -e "${GREEN}  ✓ protoc 已安装: $PROTOC_VERSION${NC}"
fi

# 检查 protoc-gen-go 插件
echo -e "\n${YELLOW}[2/4] 检查 protoc-gen-go 插件...${NC}"
if ! command -v protoc-gen-go &> /dev/null; then
    echo -e "    protoc-gen-go 未安装，正在安装..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
    echo -e "${GREEN}  ✓ protoc-gen-go 安装成功${NC}"
else
    PLUGIN_VERSION=$(protoc-gen-go --version)
    echo -e "${GREEN}  ✓ protoc-gen-go 已安装: $PLUGIN_VERSION${NC}"
fi

# 检查 protoc-gen-go-grpc 插件
echo -e "\n${YELLOW}[3/4] 检查 protoc-gen-go-grpc 插件...${NC}"
if ! command -v protoc-gen-go-grpc &> /dev/null; then
    echo -e "    protoc-gen-go-grpc 未安装，正在安装..."
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
    echo -e "${GREEN}  ✓ protoc-gen-go-grpc 安装成功${NC}"
else
    PLUGIN_VERSION=$(protoc-gen-go-grpc --version)
    echo -e "${GREEN}  ✓ protoc-gen-go-grpc 已安装: $PLUGIN_VERSION${NC}"
fi

# 生成 gRPC 代码
echo -e "\n${YELLOW}[4/4] 生成 gRPC 代码...${NC}"

# 获取脚本所在目录的绝对路径（即 protocol 目录）
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 查找所有 proto 文件（使用绝对路径）
PROTO_FILES=$(find "$SCRIPT_DIR" -name "*.proto" -type f)

if [ -z "$PROTO_FILES" ]; then
    echo -e "${RED}  ✗ 未找到 proto 文件${NC}"
    exit 1
fi

for PROTO_FILE in $PROTO_FILES; do
    # 转换为相对于脚本目录的路径，便于显示
    REL_PATH="${PROTO_FILE#$SCRIPT_DIR/}"
    echo -e "    处理: $REL_PATH"
    
    PROTO_DIR=$(dirname "$PROTO_FILE")
    OUTPUT_DIR="$PROTO_DIR"
    
    # 构建 protoc 命令
    # -I=$SCRIPT_DIR 确保可以导入项目内其他目录的 proto 文件
    # 使用绝对路径指定输入文件，确保 paths=source_relative 正常工作
    COMMAND="protoc --go_out=$SCRIPT_DIR --go_opt=paths=source_relative --go-grpc_out=$SCRIPT_DIR --go-grpc_opt=paths=source_relative -I=$SCRIPT_DIR $REL_PATH"
    
    echo -e "    命令: $COMMAND"
    
    $COMMAND
    
    echo -e "${GREEN}  ✓ 成功生成: $(basename "$PROTO_FILE")${NC}"
done

echo -e "\n=========================================="
echo -e "  gRPC 代码生成完成！"
echo -e "=========================================="
echo -e "\n生成的文件:"

for PROTO_FILE in $PROTO_FILES; do
    BASE_NAME=$(basename "$PROTO_FILE" .proto)
    echo -e "  - ${BASE_NAME}_pb.go"
    echo -e "  - ${BASE_NAME}_grpc.pb.go"
done

echo -e "\n请运行 'go mod tidy' 来更新依赖"