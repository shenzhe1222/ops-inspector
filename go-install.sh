#!/usr/bin/env bash
set -e

# ===== 基本配置 =====
GO_VERSION="1.26.9"
GO_ARCHIVE="go${GO_VERSION}.linux-amd64.tar.gz"
GO_URL="https://go.dev/dl/${GO_ARCHIVE}"
GO_DIR="/usr/local/go"

# ===== 检查环境 =====
if [ "$(id -u)" -ne 0 ]; then
    echo "错误：请使用 root 用户执行"
    exit 1
fi

if [ "$(uname -s)" != "Linux" ] || [ "$(uname -m)" != "x86_64" ]; then
    echo "错误：仅支持 Linux x86_64"
    exit 1
fi

if [ -e "$GO_DIR" ]; then
    echo "错误：$GO_DIR 已存在，请检查现有安装"
    exit 1
fi

# ===== 安装下载工具 =====
if ! command -v curl >/dev/null 2>&1; then
    if command -v apt-get >/dev/null 2>&1; then
        apt-get update
        apt-get install -y curl ca-certificates
    elif command -v dnf >/dev/null 2>&1; then
        dnf install -y curl ca-certificates
    elif command -v yum >/dev/null 2>&1; then
        yum install -y curl ca-certificates
    else
        echo "错误：请手动安装 curl 和 ca-certificates"
        exit 1
    fi
fi

# ===== 下载 Go =====
cd /tmp
echo "下载 Go ${GO_VERSION}..."
curl -fL --retry 3 "$GO_URL" -o "$GO_ARCHIVE"

# ===== 获取官方 SHA-256 =====
echo "获取官方校验值..."
CHECKSUM=$(curl -fsSL \
    "https://go.dev/dl/?mode=json&include=all" |
    python3 -c '
import json, sys
for release in json.load(sys.stdin):
    if release["version"] == "go1.26.9":
        for item in release["files"]:
            if item["filename"] == "go1.26.9.linux-amd64.tar.gz":
                print(item["sha256"])
                sys.exit(0)
sys.exit("未找到官方 SHA-256")
') || {
    echo "错误：无法获取官方校验值，请检查 python3 和网络"
    exit 1
}

# ===== 校验文件 =====
echo "${CHECKSUM}  ${GO_ARCHIVE}" | sha256sum -c - || {
    echo "错误：文件校验失败"
    exit 1
}

# ===== 安装 Go =====
tar -C /usr/local -xzf "$GO_ARCHIVE"

# ===== 配置环境变量 =====
cat > /etc/profile.d/go.sh <<'EOF'
export GOROOT=/usr/local/go
export GOPATH="$HOME/go"
export PATH="$GOROOT/bin:$GOPATH/bin:$PATH"
EOF

chmod 644 /etc/profile.d/go.sh

# 当前终端生效
export GOROOT="$GO_DIR"
export GOPATH="$HOME/go"
export PATH="$GOROOT/bin:$GOPATH/bin:$PATH"

# ===== 验证安装 =====
echo "安装完成："
go version
go env GOROOT GOPATH GOOS GOARCH

echo "请注意：其他终端重新登录后加载环境变量。"