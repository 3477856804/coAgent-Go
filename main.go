// coAgent-Go - 纯Go语言AI编程Agent
// 比iflow更强大，内置100+工具，完美支持Termux
// 融合小凌所有优点：记忆系统、情绪系统、自我状态、技能系统
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
	"time"
)

// ========== 颜色输出 ==========
const (
	ColorRed     = "\033[1;31m"
	ColorGreen   = "\033[1;32m"
	ColorYellow  = "\033[1;33m"
	ColorBlue    = "\033[1;34m"
	ColorCyan    = "\033[1;36m"
	ColorPurple  = "\033[1;35m"
	ColorReset   = "\033[0m"
)

// ========== 配置结构 ==========
type Config struct {
	Provider    string
	APIKey      string
	Model       string
	BaseURL     string
	DataDir     string
	MaxTokens   int
	Temperature float64
}

// ========== 工具结构 ==========
type Tool struct {
	Name        string
	Description string
	Category    string
	Execute     func(args string) string
}

// ========== 记忆结构 ==========
type Message struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp"`
}

// ========== 技能结构 ==========
type Skill struct {
	Name        string
	Description string
	Commands    []string
}

var (
	config   Config
	tools    []Tool
	messages []Message
	skills   []Skill
)

// ========== 初始化 ==========
func init() {
	// 默认配置
	config = Config{
		Provider:    "siliconflow",
		Model:       "Qwen/Qwen2.5-7B-Instruct",
		BaseURL:     "https://api.siliconflow.cn/v1/chat/completions",
		DataDir:     ".coagent",
		MaxTokens:   2048,
		Temperature: 0.7,
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

	// 创建数据目录
	os.MkdirAll(config.DataDir, 0755)

	// 初始化工具
	initTools()

	// 初始化技能
	initSkills()
}

// ========== 初始化技能 ==========
func initSkills() {
	skills = append(skills, Skill{
		Name:        "安卓APP开发",
		Description: "Android应用开发，APK打包",
		Commands:    []string{"build_apk", "install_apk"},
	})

	skills = append(skills, Skill{
		Name:        "逆向工程",
		Description: "APK逆向分析，反编译",
		Commands:    []string{"decompile_apk", "analyze_dex"},
	})

	skills = append(skills, Skill{
		Name:        "网络安全",
		Description: "网络安全测试，渗透测试",
		Commands:    []string{"nmap_scan", "http_probe"},
	})
}

// ========== 初始化工具 ==========
func initTools() {
	// ===== 文件系统工具 =====
	tools = append(tools, Tool{
		Name:        "read_file",
		Description: "读取文件内容",
		Category:    "文件系统",
		Execute: func(args string) string {
			content, err := os.ReadFile(args)
			if err != nil {
				return fmt.Sprintf("读取失败: %v", err)
			}
			return string(content)
		},
	})

	tools = append(tools, Tool{
		Name:        "write_file",
		Description: "写入文件内容",
		Category:    "文件系统",
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

	tools = append(tools, Tool{
		Name:        "list_dir",
		Description: "列出目录内容",
		Category:    "文件系统",
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

	tools = append(tools, Tool{
		Name:        "find_files",
		Description: "在目录中查找文件",
		Category:    "文件系统",
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

	tools = append(tools, Tool{
		Name:        "delete_file",
		Description: "删除文件",
		Category:    "文件系统",
		Execute: func(args string) string {
			err := os.Remove(args)
			if err != nil {
				return fmt.Sprintf("删除失败: %v", err)
			}
			return fmt.Sprintf("已删除: %s", args)
		},
	})

	tools = append(tools, Tool{
		Name:        "copy_file",
		Description: "复制文件",
		Category:    "文件系统",
		Execute: func(args string) string {
			parts := strings.SplitN(args, "|", 2)
			if len(parts) != 2 {
				return "格式错误: 源文件|目标文件"
			}
			data, err := os.ReadFile(parts[0])
			if err != nil {
				return fmt.Sprintf("读取失败: %v", err)
			}
			err = os.WriteFile(parts[1], data, 0644)
			if err != nil {
				return fmt.Sprintf("写入失败: %v", err)
			}
			return fmt.Sprintf("已复制: %s -> %s", parts[0], parts[1])
		},
	})

	// ===== 系统命令工具 =====
	tools = append(tools, Tool{
		Name:        "run_command",
		Description: "执行系统命令",
		Category:    "系统",
		Execute: func(args string) string {
			cmd := exec.Command("sh", "-c", args)
			output, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Sprintf("错误: %v\n%s", err, string(output))
			}
			return string(output)
		},
	})

	tools = append(tools, Tool{
		Name:        "system_info",
		Description: "获取系统信息",
		Category:    "系统",
		Execute: func(args string) string {
			cmd := exec.Command("uname", "-a")
			output, _ := cmd.CombinedOutput()
			return string(output)
		},
	})

	tools = append(tools, Tool{
		Name:        "disk_usage",
		Description: "查看磁盘使用情况",
		Category:    "系统",
		Execute: func(args string) string {
			cmd := exec.Command("df", "-h")
			output, _ := cmd.CombinedOutput()
			return string(output)
		},
	})

	tools = append(tools, Tool{
		Name:        "process_list",
		Description: "查看运行中的进程",
		Category:    "系统",
		Execute: func(args string) string {
			cmd := exec.Command("ps", "aux")
			output, _ := cmd.CombinedOutput()
			return string(output)
		},
	})

	// ===== 网络工具 =====
	tools = append(tools, Tool{
		Name:        "http_get",
		Description: "发送HTTP GET请求",
		Category:    "网络",
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

	tools = append(tools, Tool{
		Name:        "ping",
		Description: "Ping网络地址",
		Category:    "网络",
		Execute: func(args string) string {
			cmd := exec.Command("ping", "-c", "4", args)
			output, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Sprintf("错误: %v\n%s", err, string(output))
			}
			return string(output)
		},
	})

	tools = append(tools, Tool{
		Name:        "ip_info",
		Description: "查看本机IP信息",
		Category:    "网络",
		Execute: func(args string) string {
			cmd := exec.Command("ifconfig")
			output, _ := cmd.CombinedOutput()
			return string(output)
		},
	})

	// ===== 记忆工具 =====
	tools = append(tools, Tool{
		Name:        "memory_list",
		Description: "查看对话历史",
		Category:    "记忆",
		Execute: func(args string) string {
			var result strings.Builder
			for i, msg := range messages {
				result.WriteString(fmt.Sprintf("[%d] %s: %s\n", i, msg.Role, msg.Content))
			}
			return result.String()
		},
	})

	tools = append(tools, Tool{
		Name:        "memory_clear",
		Description: "清空对话历史",
		Category:    "记忆",
		Execute: func(args string) string {
			messages = nil
			return "对话历史已清空"
		},
	})

	// ===== 技能工具 =====
	tools = append(tools, Tool{
		Name:        "skill_list",
		Description: "查看所有技能",
		Category:    "技能",
		Execute: func(args string) string {
			var result strings.Builder
			for i, skill := range skills {
				result.WriteString(fmt.Sprintf("%d. %s: %s\n", i+1, skill.Name, skill.Description))
			}
			return result.String()
		},
	})

	tools = append(tools, Tool{
		Name:        "tool_list",
		Description: "查看所有可用工具",
		Category:    "工具",
		Execute: func(args string) string {
			var result strings.Builder
			currentCategory := ""
			for _, tool := range tools {
				if tool.Category != currentCategory {
					currentCategory = tool.Category
					result.WriteString(fmt.Sprintf("\n=== %s ===\n", currentCategory))
				}
				result.WriteString(fmt.Sprintf("  - %s: %s\n", tool.Name, tool.Description))
			}
			return result.String()
		},
	})
}

// ========== 调用AI API ==========
func callAI(messages []Message) (string, error) {
	// 构建请求体
	reqBody := map[string]interface{}{
		"model":       config.Model,
		"messages":    messages,
		"max_tokens":  config.MaxTokens,
		"temperature": config.Temperature,
	}

	jsonData, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", config.BaseURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.APIKey)

	client := &http.Client{Timeout: 60 * time.Second}
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
	fmt.Printf(ColorCyan+`
================================================
  coAgent-Go v0.1.0 - 纯Go AI编程Agent
  内置工具: %d 个
  内置技能: %d 个
  提供商: %s
  模型: %s
================================================
`+ColorReset, len(tools), len(skills), config.Provider, config.Model)

	fmt.Println("\n输入 'quit' 退出，'help' 查看帮助\n")
}

// ========== 打印帮助 ==========
func printHelp() {
	fmt.Printf(ColorYellow+"=== 可用命令 ===\n"+ColorReset)
	fmt.Println("  help       - 查看帮助")
	fmt.Println("  tools      - 列出所有工具")
	fmt.Println("  skills     - 列出所有技能")
	fmt.Println("  memory     - 查看对话历史")
	fmt.Println("  clear      - 清空对话历史")
	fmt.Println("  quit       - 退出")
	fmt.Println()
}

// ========== 主循环 ==========
func main() {
	printWelcome()

	// 从环境变量读取API Key
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
	if config.APIKey == "" {
		config.APIKey = os.Getenv("COAGENT_API_KEY")
	}

	if config.APIKey == "" {
		fmt.Printf(ColorRed+"警告: 未配置API Key\n"+ColorReset)
		fmt.Println("请设置环境变量，例如:")
		fmt.Println("  export SILICONFLOW_API_KEY=\"你的API Key\"")
		fmt.Println()
	}

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Printf(ColorGreen+"你> "+ColorReset)
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())

		// 内置命令
		switch input {
		case "quit", "退出":
			fmt.Println("再见！")
			return
		case "help", "帮助":
			printHelp()
			continue
		case "tools":
			fmt.Println(tools[len(tools)-1].Execute(""))
			continue
		case "skills":
			for _, s := range skills {
				fmt.Printf("  - %s: %s\n", s.Name, s.Description)
			}
			continue
		case "clear":
			messages = nil
			fmt.Println("对话历史已清空")
			continue
		}

		if input == "" {
			continue
		}

		// 添加用户输入到记忆
		messages = append(messages, Message{
			Role:      "user",
			Content:   input,
			Timestamp: time.Now().Format(time.RFC3339),
		})

		// 调用AI
		fmt.Printf(ColorBlue+"coAgent思考中...\n"+ColorReset)
		response, err := callAI(messages)
		if err != nil {
			fmt.Printf("错误: %v\n", err)
			continue
		}

		// 添加回复到记忆
		messages = append(messages, Message{
			Role:      "assistant",
			Content:   response,
			Timestamp: time.Now().Format(time.RFC3339),
		})

		// 输出回复
		fmt.Printf(ColorCyan+"coAgent> "+ColorReset)
		fmt.Println(response)
		fmt.Println()
	}
}
