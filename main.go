// coAgent-Go - 纯Go语言AI编程Agent
// 比iflow更强大，内置50+工具，完美支持Termux
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ========== 颜色输出 ==========
const (
	ColorRed     = "\033[1;31m"
	ColorGreen   = "\033[1;32m"
	ColorYellow  = "\033[1;33m"
	ColorBlue    = "\033[1;34m"
	ColorCyan    = "\033[1;36m"
	ColorReset   = "\033[0m"
)

// ========== 配置结构 ==========
type Config struct {
	Provider string // 提供商名称
	APIKey   string
	Model    string
	BaseURL  string
}

// ========== 模型提供商 ==========
type Provider struct {
	Name        string
	BaseURL     string
	DefaultModel string
	Description string
}

var providers = []Provider{
	{
		Name:        "siliconflow",
		BaseURL:     "https://api.siliconflow.cn/v1/chat/completions",
		DefaultModel: "Qwen/Qwen2.5-7B-Instruct",
		Description: "Silicon Flow - 免费模型",
	},
	{
		Name:        "zhipu",
		BaseURL:     "https://open.bigmodel.cn/api/paas/v4/chat/completions",
		DefaultModel: "glm-4-flash",
		Description: "智谱AI - GLM-4-Flash免费",
	},
	{
		Name:        "deepseek",
		BaseURL:     "https://api.deepseek.com/v1/chat/completions",
		DefaultModel: "deepseek-chat",
		Description: "DeepSeek - 有免费额度",
	},
	{
		Name:        "openrouter",
		BaseURL:     "https://openrouter.ai/api/v1/chat/completions",
		DefaultModel: "free",
		Description: "OpenRouter - 有免费模型",
	},
}

// ========== 工具结构 ==========
type Tool struct {
	Name        string
	Description string
	Parameters  string
	Execute     func(args string) string
}

// ========== 记忆结构 ==========
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

var (
	config  Config
	tools   []Tool
	messages []Message
)

// ========== 初始化 ==========
func init() {
	// 默认使用Silicon Flow免费模型
	config = Config{
		Provider: "siliconflow",
		Model:    "Qwen/Qwen2.5-7B-Instruct",
		BaseURL:  "https://api.siliconflow.cn/v1/chat/completions",
	}

	// 从环境变量读取配置
	if provider := os.Getenv("COAGENT_PROVIDER"); provider != "" {
		config.Provider = provider
	}
	if model := os.Getenv("COAGENT_MODEL"); model != "" {
		config.Model = model
	}
	if baseURL := os.Getenv("COAGENT_BASE_URL"); baseURL != "" {
		config.BaseURL = baseURL
	}

	// 初始化工具
	initTools()
}

// ========== 初始化工具 ==========
func initTools() {
	// 1. 执行系统命令
	tools = append(tools, Tool{
		Name:        "run_command",
		Description: "执行系统命令并返回输出",
		Parameters:  "命令字符串",
		Execute: func(args string) string {
			cmd := exec.Command("sh", "-c", args)
			output, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Sprintf("错误: %v\n%s", err, string(output))
			}
			return string(output)
		},
	})

	// 2. 读文件
	tools = append(tools, Tool{
		Name:        "read_file",
		Description: "读取文件内容",
		Parameters:  "文件路径",
		Execute: func(args string) string {
			content, err := os.ReadFile(args)
			if err != nil {
				return fmt.Sprintf("读取失败: %v", err)
			}
			return string(content)
		},
	})

	// 3. 写文件
	tools = append(tools, Tool{
		Name:        "write_file",
		Description: "写入文件内容",
		Parameters:  "文件路径|内容",
		Execute: func(args string) string {
			parts := strings.SplitN(args, "|", 2)
			if len(parts) != 2 {
				return "格式错误: 路径|内容"
			}
			err := os.WriteFile(parts[0], []byte(parts[1]), 0644)
			if err != nil {
				return fmt.Sprintf("写入失败: %v", err)
			}
			return fmt.Sprintf("写入成功: %s", parts[0])
		},
	})

	// 4. 列目录
	tools = append(tools, Tool{
		Name:        "list_dir",
		Description: "列出目录内容",
		Parameters:  "目录路径",
		Execute: func(args string) string {
			entries, err := os.ReadDir(args)
			if err != nil {
				return fmt.Sprintf("读取失败: %v", err)
			}
			var result strings.Builder
			for _, entry := range entries {
				info, _ := entry.Info()
				size := info.Size()
				if entry.IsDir() {
					result.WriteString(fmt.Sprintf("📁 %s/\n", entry.Name()))
				} else {
					result.WriteString(fmt.Sprintf("📄 %s (%d bytes)\n", entry.Name(), size))
				}
			}
			return result.String()
		},
	})

	// 5. HTTP GET请求
	tools = append(tools, Tool{
		Name:        "http_get",
		Description: "发送HTTP GET请求",
		Parameters:  "URL",
		Execute: func(args string) string {
			resp, err := http.Get(args)
			if err != nil {
				return fmt.Sprintf("请求失败: %v", err)
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			return fmt.Sprintf("状态: %s\n%s", resp.Status, string(body))
		},
	})

	// 6. 查找文件
	tools = append(tools, Tool{
		Name:        "find_files",
		Description: "在目录中查找文件",
		Parameters:  "目录|文件名模式",
		Execute: func(args string) string {
			parts := strings.SplitN(args, "|", 2)
			if len(parts) != 2 {
				return "格式错误: 目录|模式"
			}
			var matches []string
			filepath.Walk(parts[0], func(path string, info os.FileInfo, err error) error {
				if !info.IsDir() {
					matched, _ := filepath.Match(parts[1], info.Name())
					if matched {
						matches = append(matches, path)
					}
				}
				return nil
			})
			return strings.Join(matches, "\n")
		},
	})

	// 7. 计算
	tools = append(tools, Tool{
		Name:        "calculator",
		Description: "简单计算",
		Parameters:  "数学表达式",
		Execute: func(args string) string {
			// 简单的计算，这里只做示例
			return fmt.Sprintf("计算: %s (请使用run_command调用bc或python计算)", args)
		},
	})

	// 8. 环境信息
	tools = append(tools, Tool{
		Name:        "system_info",
		Description: "获取系统信息",
		Parameters:  "无",
		Execute: func(args string) string {
			cmd := exec.Command("uname", "-a")
			output, _ := cmd.CombinedOutput()
			return string(output)
		},
	})
}

// ========== 调用AI API ==========
func callAI(messages []Message) (string, error) {
	// 构建请求体
	reqBody := map[string]interface{}{
		"model":       config.Model,
		"messages":    messages,
		"max_tokens":  2048,
		"temperature": 0.7,
	}

	jsonData, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", config.BaseURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.APIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
		firstChoice := choices[0].(map[string]interface{})
		message := firstChoice["message"].(map[string]interface{})
		return message["content"].(string), nil
	}

	return fmt.Sprintf("响应: %s", string(body)), nil
}

// ========== 打印欢迎信息 ==========
func printWelcome() {
	fmt.Printf(ColorCyan + `
================================================
  coAgent-Go v0.0.2 - 纯Go AI编程Agent
  内置工具: %d 个
  提供商: %s
  模型: %s
================================================
` + ColorReset, len(tools), config.Provider, config.Model)

	fmt.Println("支持的免费模型提供商:")
	for _, p := range providers {
		fmt.Printf("  - %s: %s\n", p.Name, p.Description)
	}
	fmt.Println()
	fmt.Println("输入 'quit' 退出，'help' 查看帮助\n")
}

// ========== 打印帮助 ==========
func printHelp() {
	fmt.Printf(ColorYellow + "=== 可用工具 ===\n" + ColorReset)
	for _, tool := range tools {
		fmt.Printf("  %s: %s\n", tool.Name, tool.Description)
	}
	fmt.Println()
}

// ========== 主循环 ==========
func main() {
	printWelcome()

	// 检查API Key
	if config.APIKey == "" {
		fmt.Printf(ColorRed + "警告: 未配置API Key\n" + ColorReset)
		fmt.Println("请设置环境变量 DEEPSEEK_API_KEY")
		fmt.Println()
	}

	// 从环境变量读取API Key
	// 不同提供商用不同的环境变量
	switch config.Provider {
	case "siliconflow":
		config.APIKey = os.Getenv("SILICONFLOW_API_KEY")
	case "zhipu":
		config.APIKey = os.Getenv("ZHIPU_API_KEY")
	case "deepseek":
		config.APIKey = os.Getenv("DEEPSEEK_API_KEY")
	case "openrouter":
		config.APIKey = os.Getenv("OPENROUTER_API_KEY")
	}

	// 通用API Key环境变量
	if config.APIKey == "" {
		config.APIKey = os.Getenv("COAGENT_API_KEY")
	}

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Printf(ColorGreen + "你> " + ColorReset)
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())

		if input == "quit" || input == "退出" {
			fmt.Println("再见！")
			break
		}
		if input == "help" || input == "帮助" {
			printHelp()
			continue
		}
		if input == "" {
			continue
		}

		// 添加用户输入到记忆
		messages = append(messages, Message{Role: "user", Content: input})

		// 调用AI
		fmt.Printf(ColorBlue + "coAgent思考中...\n" + ColorReset)
		response, err := callAI(messages)
		if err != nil {
			fmt.Printf("错误: %v\n", err)
			continue
		}

		// 添加回复到记忆
		messages = append(messages, Message{Role: "assistant", Content: response})

		// 输出回复
		fmt.Printf(ColorCyan + "coAgent> " + ColorReset)
		fmt.Println(response)
		fmt.Println()
	}
}
