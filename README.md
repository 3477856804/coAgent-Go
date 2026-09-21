# coAgent-Go

纯Go语言开发的AI编程Agent，比iflow更强大，完美支持Termux安卓环境。

## 特性

- 🚀 **纯Go开发**：单二进制文件，无依赖，直接运行
- 🛠️ **内置8+核心工具**：执行命令、读写文件、列目录、HTTP请求、查找文件等
- 🧠 **内置记忆系统**：自动记住对话历史
- 📱 **完美支持Termux**：安卓手机直接运行
- ⚡ **轻量快速**：启动快，内存占用低
- 🔌 **支持DeepSeek API**：国内访问快，成本低

## 安装

### Termux / Linux / macOS

```bash
# 克隆仓库
git clone https://github.com/3477856804/coAgent-Go.git
cd coAgent-Go

# 编译
go build -o coAgent main.go

# 运行
./coAgent
```

### 直接下载二进制

Releases页面下载对应平台的二进制文件，直接运行。

## 配置

### 免费模型API

**Silicon Flow（推荐，完全免费）：**
```bash
# 注册获取免费API Key: https://siliconflow.cn
export SILICONFLOW_API_KEY="你的API Key"
export COAGENT_PROVIDER="siliconflow"
export COAGENT_MODEL="Qwen/Qwen2.5-7B-Instruct"
```

**智谱AI（GLM-4-Flash免费）：**
```bash
# 注册获取免费API Key: https://open.bigmodel.cn
export ZHIPU_API_KEY="你的API Key"
export COAGENT_PROVIDER="zhipu"
export COAGENT_MODEL="glm-4-flash"
```

**DeepSeek（有免费额度）：**
```bash
# 注册获取API Key: https://platform.deepseek.com
export DEEPSEEK_API_KEY="你的API Key"
export COAGENT_PROVIDER="deepseek"
```

**OpenRouter（有免费模型）：**
```bash
# 注册获取API Key: https://openrouter.ai
export OPENROUTER_API_KEY="你的API Key"
export COAGENT_PROVIDER="openrouter"
```

## 使用

```bash
# 启动
./coAgent

# 命令
help    # 查看帮助
quit    # 退出
```

## 内置工具

| 工具名 | 功能 |
|--------|------|
| run_command | 执行系统命令 |
| read_file | 读取文件内容 |
| write_file | 写入文件内容 |
| list_dir | 列出目录内容 |
| http_get | 发送HTTP GET请求 |
| find_files | 查找文件 |
| calculator | 简单计算 |
| system_info | 获取系统信息 |

## 路线图

- [ ] 增加更多工具（蓝牙控制、文件搜索、代码执行等）
- [ ] 增加向量记忆、长期记忆
- [ ] 增加工具自动调用（AI自己决定用哪个工具）
- [ ] 增加Termux专属工具（短信、电话、传感器等）
- [ ] 支持更多模型（Ollama、本地模型等）

## 许可证

MIT
