# Docker Manager (DM)

一个基于终端的 Docker 管理工具，提供直观的 TUI 界面来管理 Docker 容器和镜像。

![Docker Manager Screenshot](screenshot.png)

## 功能特性

### 容器管理
- 📋 查看所有容器列表（运行中+已停止）
- 📊 实时监控容器 CPU 和内存使用情况
- 🔍 查看容器详细信息（inspect）
- 📜 查看容器日志
- ▶️ 启动/停止容器
- 🗑️ 删除容器
- 💻 进入容器执行命令（exec）

### 镜像管理
- 🖼️ 查看所有镜像列表
- 🏷️ 给镜像添加标签
- 🗑️ 删除镜像

### 界面特性
- 🎨 彩色终端界面
- ⌨️ 完整的键盘快捷键支持
- 🔄 自动刷新（2秒间隔）
- 📱 响应式布局

## 安装

### 前置要求
- Go 1.21 或更高版本
- Docker 服务正在运行
- 当前用户有权限访问 Docker（或者在 sudo 下运行）

### 从源码安装

```bash
# 克隆仓库
git clone https://github.com/yourusername/docker-manager.git
cd docker-manager

# 编译
bash build.sh

# 运行
./dist/dm
