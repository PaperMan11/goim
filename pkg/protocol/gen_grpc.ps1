<#
.SYNOPSIS
    生成 OpenIM gRPC 代码脚本
.DESCRIPTION
    该脚本用于从 proto 文件生成 Go gRPC 代码
    需要提前安装 protoc 和相关插件
#>

$ErrorActionPreference = "Stop"

# 生成 gRPC 代码
Write-Host "`n=========================================="
Write-Host "  OpenIM gRPC 代码生成脚本"
Write-Host "==========================================`n"

# 检查 protoc 是否安装
Write-Host "[1/4] 检查 protoc 安装..."
try {
    $protocVersion = protoc --version 2>&1
    if ($LASTEXITCODE -ne 0) {
        throw "protoc 未安装或未添加到 PATH"
    }
    Write-Host "  ✓ protoc 已安装: $protocVersion`n"
} catch {
    Write-Host "  ✗ 错误: $_"
    Write-Host "    请从 https://github.com/protocolbuffers/protobuf/releases 下载并安装 protoc"
    exit 1
}

# 检查 protoc-gen-go 插件
Write-Host "[2/4] 检查 protoc-gen-go 插件..."
try {
    $pluginVersion = protoc-gen-go --version 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Host "    protoc-gen-go 未安装，正在安装..."
        go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
        Write-Host "  ✓ protoc-gen-go 安装成功`n"
    } else {
        Write-Host "  ✓ protoc-gen-go 已安装: $pluginVersion`n"
    }
} catch {
    Write-Host "  ✗ 安装失败: $_"
    exit 1
}

# 检查 protoc-gen-go-grpc 插件
Write-Host "[3/4] 检查 protoc-gen-go-grpc 插件..."
try {
    $pluginVersion = protoc-gen-go-grpc --version 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Host "    protoc-gen-go-grpc 未安装，正在安装..."
        go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
        Write-Host "  ✓ protoc-gen-go-grpc 安装成功`n"
    } else {
        Write-Host "  ✓ protoc-gen-go-grpc 已安装: $pluginVersion`n"
    }
} catch {
    Write-Host "  ✗ 安装失败: $_"
    exit 1
}

# 生成 gRPC 代码
Write-Host "[4/4] 生成 gRPC 代码..."

# 获取脚本所在目录
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

# 查找所有 proto 文件
$protoFiles = Get-ChildItem -Path $scriptDir -Recurse -Filter "*.proto"

if (-not $protoFiles) {
    Write-Host "  ✗ 未找到 proto 文件"
    exit 1
}

foreach ($protoFile in $protoFiles) {
    $relPath = $protoFile.FullName.Substring($scriptDir.Length + 1)
    Write-Host "    处理: $relPath"
    
    # 构建 protoc 命令
    # --go_out=$scriptDir 指定输出目录为 protocol 目录
    # -I=$scriptDir 确保可以导入项目内其他目录的 proto 文件
    # $relPath 使用相对路径指定输入文件
    $command = "protoc --go_out=$scriptDir --go_opt=paths=source_relative --go-grpc_out=$scriptDir --go-grpc_opt=paths=source_relative -I=$scriptDir $relPath"
    
    Write-Host "    命令: $command"
    
    try {
        Invoke-Expression $command
        if ($LASTEXITCODE -ne 0) {
            throw "protoc 命令执行失败，退出码: $LASTEXITCODE"
        }
        Write-Host "  ✓ 成功生成: $($protoFile.Name)`n"
    } catch {
        Write-Host "  ✗ 生成失败: $_"
        exit 1
    }
}

Write-Host "=========================================="
Write-Host "  gRPC 代码生成完成！"
Write-Host "=========================================="
Write-Host "`n生成的文件:"

foreach ($protoFile in $protoFiles) {
    $baseName = $protoFile.BaseName
    Write-Host "  - ${baseName}_pb.go"
    Write-Host "  - ${baseName}_grpc.pb.go"
}

Write-Host "`n请运行 'go mod tidy' 来更新依赖"