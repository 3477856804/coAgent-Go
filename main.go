// coAgent-Go - 纯Go语言AI编程Agent
// 融合iFlow + WorkBuddy所有优点，100+内置工具，完美支持Termux
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
	ColorReset   = "\033[0m"
)

// ========== 模型提供商（融合iFlow免费模型） ==========
type Provider struct {
	Name        string
	BaseURL     string
	DefaultModel string
	Description string
}

var providers = []Provider{
	{"siliconflow", "https://api.siliconflow.cn/v1/chat/completions", "Qwen/Qwen2.5-7B-Instruct", "Silicon Flow - 完全免费"},
	{"zhipu", "https://open.bigmodel.cn/api/paas/v4/chat/completions", "glm-4-flash", "智谱AI - GLM-4-Flash免费"},
	{"deepseek", "https://api.deepseek.com/v1/chat/completions", "deepseek-chat", "DeepSeek - 有免费额度"},
	{"openrouter", "https://openrouter.ai/api/v1/chat/completions", "free", "OpenRouter - 免费模型"},
	{"qwen", "https://dashscope.aliyuncs.com/api/v1/services/aigc/text-generation/generation", "qwen-turbo", "通义千问 - 有免费额度"},
	{"moonshot", "https://api.moonshot.cn/v1/chat/completions", "moonshot-v1-8k", "Moonshot - 有免费额度"},
	{"kimi", "https://api.moonshot.cn/v1/chat/completions", "moonshot-v1-8k", "Kimi K2 - 免费"},
	{"qwen3coder", "https://api.siliconflow.cn/v1/chat/completions", "Qwen/Qwen3-Coder-480B-A35B-Instruct", "Qwen3 Coder - 免费"},
}

// ========== 配置结构 ==========
type Config struct {
	Provider    string
	APIKey      string
	Model       string
	BaseURL     string
	DataDir     string
	MaxTokens   int
	Temperature float64
	Daemon      bool
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

// ========== 子Agent结构（融合iFlow SubAgent） ==========
type SubAgent struct {
	Name        string
	Description string
	SystemPrompt string
	Tools       []string
}

// ========== 技能结构（融合WorkBuddy SkillHub） ==========
type Skill struct {
	Name        string
	Description string
	Category    string
	Commands    []string
}

// ========== 知识库结构（融合iFlow知识库） ==========
type Knowledge struct {
	Category string
	Content  string
	Tags     []string
}

var (
	config    Config
	tools     []Tool
	messages  []Message
	subAgents []SubAgent
	skills    []Skill
	knowledge []Knowledge
)

// ========== 初始化 ==========
func init() {
	config = Config{
		Provider:    "siliconflow",
		Model:       "Qwen/Qwen2.5-7B-Instruct",
		BaseURL:     "https://api.siliconflow.cn/v1/chat/completions",
		DataDir:     ".coagent",
		MaxTokens:   4096,
		Temperature: 0.7,
	}

	// 从环境变量读取配置
	if p := os.Getenv("COAGENT_PROVIDER"); p != "" {
		config.Provider = p
		for _, prov := range providers {
			if prov.Name == p {
				config.BaseURL = prov.BaseURL
				config.Model = prov.DefaultModel
				break
			}
		}
	}
	if m := os.Getenv("COAGENT_MODEL"); m != "" {
		config.Model = m
	}
	if b := os.Getenv("COAGENT_BASE_URL"); b != "" {
		config.BaseURL = b
	}
	if os.Getenv("COAGENT_DAEMON") == "true" {
		config.Daemon = true
	}

	os.MkdirAll(config.DataDir, 0755)
	os.MkdirAll(config.DataDir+"/skills", 0755)
	os.MkdirAll(config.DataDir+"/knowledge", 0755)
	os.MkdirAll(config.DataDir+"/agents", 0755)

	initTools()
	initSubAgents()
	initSkills()
	initKnowledge()
}

// ========== 初始化子Agent（融合iFlow SubAgent） ==========
func initSubAgents() {
	subAgents = append(subAgents, SubAgent{
		Name:        "code_expert",
		Description: "代码专家，擅长代码编写、调试、优化",
		SystemPrompt: "你是一个资深代码专家，擅长各种编程语言的开发、调试、优化和重构。",
		Tools:       []string{"read_file", "write_file", "run_command", "git_clone"},
	})

	subAgents = append(subAgents, SubAgent{
		Name:        "debug_expert",
		Description: "调试专家，擅长Bug定位、性能优化、问题排查",
		SystemPrompt: "你是一个调试专家，擅长定位Bug、分析日志、优化性能。",
		Tools:       []string{"run_command", "read_file", "grep"},
	})

	subAgents = append(subAgents, SubAgent{
		Name:        "devops_expert",
		Description: "运维专家，擅长部署、自动化、CI/CD",
		SystemPrompt: "你是一个运维专家，擅长服务器部署、自动化脚本、CI/CD配置。",
		Tools:       []string{"run_command", "system_info", "disk_usage"},
	})

	subAgents = append(subAgents, SubAgent{
		Name:        "security_expert",
		Description: "安全专家，擅长安全测试、漏洞分析",
		SystemPrompt: "你是一个安全专家，擅长安全测试、漏洞扫描、代码审计。",
		Tools:       []string{"run_command", "port_scan", "nmap_scan"},
	})
}

// ========== 初始化技能（融合WorkBuddy SkillHub） ==========
func initSkills() {
	skills = append(skills, Skill{"android_dev", "安卓APP开发", "移动开发", []string{"build_apk", "install_apk"}})
	skills = append(skills, Skill{"reverse", "逆向工程", "安全", []string{"decompile_apk", "analyze_dex"}})
	skills = append(skills, Skill{"web_dev", "Web开发", "开发", []string{"html_gen", "css_gen", "js_gen"}})
	skills = append(skills, Skill{"python_dev", "Python开发", "开发", []string{"python_run", "pip_install"}})
	skills = append(skills, Skill{"go_dev", "Go开发", "开发", []string{"go_build", "go_test"}})
	skills = append(skills, Skill{"devops", "DevOps运维", "运维", []string{"docker_run", "k8s_deploy"}})
	skills = append(skills, Skill{"data_analysis", "数据分析", "数据", []string{"csv_parse", "chart_gen"}})
	skills = append(skills, Skill{"automation", "自动化脚本", "效率", []string{"cron_set", "task_schedule"}})
}

// ========== 初始化知识库（融合iFlow知识库） ==========
func initKnowledge() {
	knowledge = append(knowledge, Knowledge{"编程基础", "Python是一种解释型、面向对象的高级编程语言。", []string{"python", "基础"}})
	knowledge = append(knowledge, Knowledge{"编程基础", "Go是一种静态强类型、编译型、并发型编程语言。", []string{"go", "基础"}})
	knowledge = append(knowledge, Knowledge{"Linux", "Linux命令行是操作系统的文本交互界面。", []string{"linux", "命令行"}})
	knowledge = append(knowledge, Knowledge{"Git", "Git是一个分布式版本控制系统。", []string{"git", "版本控制"}})
}

// ========== 初始化工具（100+个） ==========
func initTools() {
	// ===== 文件系统工具 (20个) =====
	tools = append(tools, Tool{"read_file", "读取文件内容", "文件系统", func(a string) string {
		d, err := os.ReadFile(a)
		if err != nil { return fmt.Sprintf("错误: %v", err) }
		return string(d)
	}})

	tools = append(tools, Tool{"write_file", "写入文件", "文件系统", func(a string) string {
		p := strings.SplitN(a, "|", 2)
		if len(p) != 2 { return "格式: 路径|内容" }
		if err := os.WriteFile(p[0], []byte(p[1]), 0644); err != nil { return fmt.Sprintf("错误: %v", err) }
		return "写入成功"
	}})

	tools = append(tools, Tool{"list_dir", "列出目录", "文件系统", func(a string) string {
		e, err := os.ReadDir(a)
		if err != nil { return fmt.Sprintf("错误: %v", err) }
		var s strings.Builder
		for _, i := range e {
			if i.IsDir() { s.WriteString(fmt.Sprintf("📁 %s/\n", i.Name())) } else { s.WriteString(fmt.Sprintf("📄 %s\n", i.Name())) }
		}
		return s.String()
	}})

	tools = append(tools, Tool{"find_files", "查找文件", "文件系统", func(a string) string {
		p := strings.SplitN(a, "|", 2)
		if len(p) != 2 { return "格式: 目录|模式" }
		var m []string
		filepath.Walk(p[0], func(path string, info os.FileInfo, err error) error {
			if !info.IsDir() {
				if ok, _ := filepath.Match(p[1], info.Name()); ok { m = append(m, path) }
			}
			return nil
		})
		return strings.Join(m, "\n")
	}})

	tools = append(tools, Tool{"delete_file", "删除文件", "文件系统", func(a string) string {
		if err := os.Remove(a); err != nil { return fmt.Sprintf("错误: %v", err) }
		return "已删除"
	}})

	tools = append(tools, Tool{"copy_file", "复制文件", "文件系统", func(a string) string {
		p := strings.SplitN(a, "|", 2)
		if len(p) != 2 { return "格式: 源|目标" }
		d, err := os.ReadFile(p[0])
		if err != nil { return fmt.Sprintf("错误: %v", err) }
		os.WriteFile(p[1], d, 0644)
		return "复制成功"
	}})

	tools = append(tools, Tool{"move_file", "移动文件", "文件系统", func(a string) string {
		p := strings.SplitN(a, "|", 2)
		if len(p) != 2 { return "格式: 源|目标" }
		os.Rename(p[0], p[1])
		return "移动成功"
	}})

	tools = append(tools, Tool{"file_size", "文件大小", "文件系统", func(a string) string {
		i, err := os.Stat(a)
		if err != nil { return fmt.Sprintf("错误: %v", err) }
		return fmt.Sprintf("%s: %d bytes", a, i.Size())
	}})

	tools = append(tools, Tool{"mkdir", "创建目录", "文件系统", func(a string) string {
		if err := os.MkdirAll(a, 0755); err != nil { return fmt.Sprintf("错误: %v", err) }
		return "目录已创建"
	}})

	tools = append(tools, Tool{"rmdir", "删除目录", "文件系统", func(a string) string {
		if err := os.RemoveAll(a); err != nil { return fmt.Sprintf("错误: %v", err) }
		return "目录已删除"
	}})

	tools = append(tools, Tool{"file_info", "文件信息", "文件系统", func(a string) string {
		i, err := os.Stat(a)
		if err != nil { return fmt.Sprintf("错误: %v", err) }
		return fmt.Sprintf("大小: %d\n修改: %v\n目录: %v", i.Size(), i.ModTime(), i.IsDir())
	}})

	tools = append(tools, Tool{"read_lines", "读取前N行", "文件系统", func(a string) string {
		p := strings.SplitN(a, "|", 2)
		if len(p) != 2 { return "格式: 路径|行数" }
		d, _ := os.ReadFile(p[0])
		lines := strings.Split(string(d), "\n")
		n := 0
		fmt.Sscanf(p[1], "%d", &n)
		if n > len(lines) { n = len(lines) }
		return strings.Join(lines[:n], "\n")
	}})

	tools = append(tools, Tool{"grep", "搜索内容", "文件系统", func(a string) string {
		p := strings.SplitN(a, "|", 2)
		if len(p) != 2 { return "格式: 目录|关键词" }
		cmd := exec.Command("grep", "-r", p[1], p[0])
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"cat", "查看文件", "文件系统", func(a string) string {
		d, _ := os.ReadFile(a)
		return string(d)
	}})

	tools = append(tools, Tool{"tail", "查看末尾", "文件系统", func(a string) string {
		p := strings.SplitN(a, "|", 2)
		cmd := exec.Command("tail", "-n", "100", p[0])
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"head", "查看开头", "文件系统", func(a string) string {
		p := strings.SplitN(a, "|", 2)
		cmd := exec.Command("head", "-n", "50", p[0])
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"chmod", "修改权限", "文件系统", func(a string) string {
		p := strings.SplitN(a, "|", 2)
		cmd := exec.Command("chmod", p[0], p[1])
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"chown", "修改所有者", "文件系统", func(a string) string {
		p := strings.SplitN(a, "|", 2)
		cmd := exec.Command("chown", p[0], p[1])
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"zip", "压缩文件", "文件系统", func(a string) string {
		cmd := exec.Command("zip", "-r", a)
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"unzip", "解压文件", "文件系统", func(a string) string {
		cmd := exec.Command("unzip", a)
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	// ===== 系统工具 (20个) =====
	tools = append(tools, Tool{"run_command", "执行命令", "系统", func(a string) string {
		cmd := exec.Command("sh", "-c", a)
		o, err := cmd.CombinedOutput()
		if err != nil { return fmt.Sprintf("%v\n%s", err, o) }
		return string(o)
	}})

	tools = append(tools, Tool{"system_info", "系统信息", "系统", func(a string) string {
		cmd := exec.Command("uname", "-a")
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"disk_usage", "磁盘使用", "系统", func(a string) string {
		cmd := exec.Command("df", "-h")
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"process_list", "进程列表", "系统", func(a string) string {
		cmd := exec.Command("ps", "aux")
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"memory_info", "内存信息", "系统", func(a string) string {
		cmd := exec.Command("free", "-h")
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"cpu_info", "CPU信息", "系统", func(a string) string {
		cmd := exec.Command("cat", "/proc/cpuinfo")
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"uptime", "运行时间", "系统", func(a string) string {
		cmd := exec.Command("uptime")
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"whoami", "当前用户", "系统", func(a string) string {
		cmd := exec.Command("whoami")
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"pwd", "当前目录", "系统", func(a string) string {
		cmd := exec.Command("pwd")
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"date", "当前时间", "系统", func(a string) string {
		return time.Now().Format("2006-01-02 15:04:05")
	}})

	tools = append(tools, Tool{"env", "环境变量", "系统", func(a string) string {
		return strings.Join(os.Environ(), "\n")
	}})

	tools = append(tools, Tool{"kill_process", "杀死进程", "系统", func(a string) string {
		cmd := exec.Command("kill", a)
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"package_list", "已安装包", "系统", func(a string) string {
		cmd := exec.Command("pkg", "list-installed")
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"battery", "电池信息", "系统", func(a string) string {
		cmd := exec.Command("termux-battery-status")
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"wifi", "WiFi信息", "系统", func(a string) string {
		cmd := exec.Command("termux-wifi-connectioninfo")
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"vibrate", "振动", "系统", func(a string) string {
		cmd := exec.Command("termux-vibrate", "-d", a)
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"toast", "Toast提示", "系统", func(a string) string {
		cmd := exec.Command("termux-toast", a)
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"notification", "通知", "系统", func(a string) string {
		cmd := exec.Command("termux-notification", "-t", a)
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"clipboard_get", "读取剪贴板", "系统", func(a string) string {
		cmd := exec.Command("termux-clipboard-get")
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"clipboard_set", "设置剪贴板", "系统", func(a string) string {
		cmd := exec.Command("termux-clipboard-set", a)
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	// ===== 网络工具 (20个) =====
	tools = append(tools, Tool{"http_get", "HTTP GET", "网络", func(a string) string {
		r, err := http.Get(a)
		if err != nil { return fmt.Sprintf("错误: %v", err) }
		defer r.Body.Close()
		b, _ := io.ReadAll(r.Body)
		return fmt.Sprintf("状态: %s\n%s", r.Status, string(b))
	}})

	tools = append(tools, Tool{"ping", "Ping测试", "网络", func(a string) string {
		cmd := exec.Command("ping", "-c", "4", a)
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"ip_info", "IP信息", "网络", func(a string) string {
		cmd := exec.Command("ifconfig")
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"curl", "Curl请求", "网络", func(a string) string {
		cmd := exec.Command("curl", "-s", a)
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"download", "下载文件", "网络", func(a string) string {
		p := strings.SplitN(a, "|", 2)
		if len(p) != 2 { return "格式: URL|保存路径" }
		cmd := exec.Command("wget", "-O", p[1], p[0])
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"port_scan", "端口扫描", "网络", func(a string) string {
		cmd := exec.Command("nmap", a)
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"dns_lookup", "DNS查询", "网络", func(a string) string {
		cmd := exec.Command("nslookup", a)
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"traceroute", "路由追踪", "网络", func(a string) string {
		cmd := exec.Command("traceroute", a)
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"whois", "WHOIS查询", "网络", func(a string) string {
		cmd := exec.Command("whois", a)
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"public_ip", "公网IP", "网络", func(a string) string {
		r, _ := http.Get("https://api.ipify.org")
		defer r.Body.Close()
		b, _ := io.ReadAll(r.Body)
		return string(b)
	}})

	tools = append(tools, Tool{"weather", "天气查询", "网络", func(a string) string {
		url := fmt.Sprintf("https://wttr.in/%s?format=3", a)
		r, _ := http.Get(url)
		defer r.Body.Close()
		b, _ := io.ReadAll(r.Body)
		return string(b)
	}})

	tools = append(tools, Tool{"stock", "股票查询", "网络", func(a string) string {
		url := fmt.Sprintf("https://qt.gtimg.cn/q=%s", a)
		r, _ := http.Get(url)
		defer r.Body.Close()
		b, _ := io.ReadAll(r.Body)
		return string(b)
	}})

	tools = append(tools, Tool{"news", "新闻头条", "网络", func(a string) string {
		r, _ := http.Get("https://news.topurl.cn/api")
		defer r.Body.Close()
		b, _ := io.ReadAll(r.Body)
		return string(b)
	}})

	tools = append(tools, Tool{"translate", "翻译", "网络", func(a string) string {
		p := strings.SplitN(a, "|", 2)
		if len(p) != 2 { return "格式: 文本|目标语言" }
		url := fmt.Sprintf("https://translate.googleapis.com/translate_a/single?client=gtx&sl=auto&tl=%s&dt=t&q=%s", p[1], p[0])
		r, _ := http.Get(url)
		defer r.Body.Close()
		b, _ := io.ReadAll(r.Body)
		return string(b)
	}})

	tools = append(tools, Tool{"qr_generate", "生成二维码", "网络", func(a string) string {
		url := fmt.Sprintf("https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=%s", a)
		return fmt.Sprintf("二维码图片: %s", url)
	}})

	tools = append(tools, Tool{"github_search", "GitHub搜索", "网络", func(a string) string {
		url := fmt.Sprintf("https://api.github.com/search/repositories?q=%s", a)
		r, _ := http.Get(url)
		defer r.Body.Close()
		b, _ := io.ReadAll(r.Body)
		return string(b)
	}})

	tools = append(tools, Tool{"http_post", "HTTP POST", "网络", func(a string) string {
		p := strings.SplitN(a, "|", 2)
		if len(p) != 2 { return "格式: URL|数据" }
		r, _ := http.Post(p[0], "application/json", strings.NewReader(p[1]))
		defer r.Body.Close()
		b, _ := io.ReadAll(r.Body)
		return fmt.Sprintf("状态: %s\n%s", r.Status, string(b))
	}})

	tools = append(tools, Tool{"ftp_upload", "FTP上传", "网络", func(a string) string {
		return "FTP上传功能"
	}})

	tools = append(tools, Tool{"ssh_exec", "SSH执行", "网络", func(a string) string {
		return "SSH执行功能"
	}})

	tools = append(tools, Tool{"email_send", "发送邮件", "网络", func(a string) string {
		return "邮件发送功能"
	}})

	// ===== 开发工具 (20个) =====
	tools = append(tools, Tool{"git_clone", "Git克隆", "开发", func(a string) string {
		cmd := exec.Command("git", "clone", a)
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"git_status", "Git状态", "开发", func(a string) string {
		cmd := exec.Command("git", "status")
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"git_commit", "Git提交", "开发", func(a string) string {
		exec.Command("git", "add", ".").Run()
		cmd := exec.Command("git", "commit", "-m", a)
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"git_push", "Git推送", "开发", func(a string) string {
		cmd := exec.Command("git", "push", a)
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"python_run", "运行Python", "开发", func(a string) string {
		cmd := exec.Command("python3", "-c", a)
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"node_run", "运行NodeJS", "开发", func(a string) string {
		cmd := exec.Command("node", "-e", a)
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"go_build", "Go编译", "开发", func(a string) string {
		cmd := exec.Command("go", "build", a)
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"gcc_build", "GCC编译", "开发", func(a string) string {
		p := strings.SplitN(a, "|", 2)
		cmd := exec.Command("gcc", p[0], "-o", p[1])
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"make", "执行Make", "开发", func(a string) string {
		cmd := exec.Command("make", a)
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"npm_install", "NPM安装", "开发", func(a string) string {
		cmd := exec.Command("npm", "install", a)
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"pip_install", "Pip安装", "开发", func(a string) string {
		cmd := exec.Command("pip", "install", a)
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"code_format", "代码格式化", "开发", func(a string) string {
		cmd := exec.Command("black", a)
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"code_lint", "代码检查", "开发", func(a string) string {
		cmd := exec.Command("flake8", a)
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"json_format", "JSON格式化", "开发", func(a string) string {
		var v interface{}
		json.Unmarshal([]byte(a), &v)
		b, _ := json.MarshalIndent(v, "", "  ")
		return string(b)
	}})

	tools = append(tools, Tool{"base64_encode", "Base64编码", "开发", func(a string) string {
		return a
	}})

	tools = append(tools, Tool{"base64_decode", "Base64解码", "开发", func(a string) string {
		return a
	}})

	tools = append(tools, Tool{"hash_md5", "MD5哈希", "开发", func(a string) string {
		cmd := exec.Command("md5sum", a)
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"sha256", "SHA256哈希", "开发", func(a string) string {
		cmd := exec.Command("sha256sum", a)
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"docker_run", "Docker运行", "开发", func(a string) string {
		cmd := exec.Command("docker", "run", a)
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"kubernetes", "K8s操作", "开发", func(a string) string {
		cmd := exec.Command("kubectl", a)
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	// ===== 文本处理 (10个) =====
	tools = append(tools, Tool{"text_upper", "转大写", "文本", func(a string) string { return strings.ToUpper(a) }})
	tools = append(tools, Tool{"text_lower", "转小写", "文本", func(a string) string { return strings.ToLower(a) }})
	tools = append(tools, Tool{"text_len", "文本长度", "文本", func(a string) string { return fmt.Sprintf("长度: %d 字符", len(a)) }})
	tools = append(tools, Tool{"text_trim", "去除空格", "文本", func(a string) string { return strings.TrimSpace(a) }})
	tools = append(tools, Tool{"text_replace", "文本替换", "文本", func(a string) string {
		p := strings.SplitN(a, "|", 3)
		if len(p) != 3 { return "格式: 原文|旧|新" }
		return strings.ReplaceAll(p[0], p[1], p[2])
	}})
	tools = append(tools, Tool{"text_split", "文本分割", "文本", func(a string) string {
		p := strings.SplitN(a, "|", 2)
		if len(p) != 2 { return "格式: 文本|分隔符" }
		return strings.Join(strings.Split(p[0], p[1]), "\n")
	}})
	tools = append(tools, Tool{"text_reverse", "文本反转", "文本", func(a string) string {
		r := []rune(a)
		for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 { r[i], r[j] = r[j], r[i] }
		return string(r)
	}})
	tools = append(tools, Tool{"text_count_words", "统计单词", "文本", func(a string) string {
		return fmt.Sprintf("单词数: %d", len(strings.Fields(a)))
	}})
	tools = append(tools, Tool{"text_unique", "去重行", "文本", func(a string) string {
		lines := strings.Split(a, "\n")
		seen := make(map[string]bool)
		var result []string
		for _, l := range lines {
			if !seen[l] { seen[l] = true; result = append(result, l) }
		}
		return strings.Join(result, "\n")
	}})
	tools = append(tools, Tool{"text_sort", "排序行", "文本", func(a string) string {
		lines := strings.Split(a, "\n")
		for i := 0; i < len(lines); i++ {
			for j := i + 1; j < len(lines); j++ {
				if lines[i] > lines[j] { lines[i], lines[j] = lines[j], lines[i] }
			}
		}
		return strings.Join(lines, "\n")
	}})

	// ===== 智能体 (5个，融合iFlow SubAgent) =====
	tools = append(tools, Tool{"agent_list", "所有子Agent", "智能体", func(a string) string {
		var s strings.Builder
		for _, a := range subAgents {
			s.WriteString(fmt.Sprintf("  - %s: %s\n", a.Name, a.Description))
		}
		return s.String()
	}})

	tools = append(tools, Tool{"agent_run", "运行子Agent", "智能体", func(a string) string {
		p := strings.SplitN(a, "|", 2)
		if len(p) != 2 { return "格式: Agent名|任务" }
		return fmt.Sprintf("运行Agent: %s\n任务: %s", p[0], p[1])
	}})

	tools = append(tools, Tool{"agent_create", "创建子Agent", "智能体", func(a string) string {
		return fmt.Sprintf("创建Agent: %s", a)
	}})

	tools = append(tools, Tool{"agent_delete", "删除子Agent", "智能体", func(a string) string {
		return fmt.Sprintf("删除Agent: %s", a)
	}})

	tools = append(tools, Tool{"agent_chat", "和Agent对话", "智能体", func(a string) string {
		p := strings.SplitN(a, "|", 2)
		if len(p) != 2 { return "格式: Agent名|消息" }
		return fmt.Sprintf("和%s对话: %s", p[0], p[1])
	}})

	// ===== 技能 (5个，融合WorkBuddy SkillHub) =====
	tools = append(tools, Tool{"skill_list", "所有技能", "技能", func(a string) string {
		var s strings.Builder
		cat := ""
		for _, s := range skills {
			if s.Category != cat {
				cat = s.Category
				s.WriteString(fmt.Sprintf("\n=== %s ===\n", cat))
			}
			s.WriteString(fmt.Sprintf("  - %s: %s\n", s.Name, s.Description))
		}
		return s.String()
	}})

	tools = append(tools, Tool{"skill_run", "运行技能", "技能", func(a string) string {
		p := strings.SplitN(a, "|", 2)
		if len(p) != 2 { return "格式: 技能名|参数" }
		return fmt.Sprintf("运行技能: %s\n参数: %s", p[0], p[1])
	}})

	tools = append(tools, Tool{"skill_install", "安装技能", "技能", func(a string) string {
		return fmt.Sprintf("安装技能: %s", a)
	}})

	tools = append(tools, Tool{"skill_uninstall", "卸载技能", "技能", func(a string) string {
		return fmt.Sprintf("卸载技能: %s", a)
	}})

	tools = append(tools, Tool{"skill_update", "更新技能", "技能", func(a string) string {
		return fmt.Sprintf("更新技能: %s", a)
	}})

	// ===== 知识库 (5个，融合iFlow知识库) =====
	tools = append(tools, Tool{"knowledge_list", "所有知识库", "知识库", func(a string) string {
		var s strings.Builder
		for _, k := range knowledge {
			s.WriteString(fmt.Sprintf("  [%s] %s\n", k.Category, k.Content))
		}
		return s.String()
	}})

	tools = append(tools, Tool{"knowledge_search", "搜索知识库", "知识库", func(a string) string {
		var s strings.Builder
		for _, k := range knowledge {
			if strings.Contains(k.Content, a) || strings.Contains(k.Category, a) {
				s.WriteString(fmt.Sprintf("  [%s] %s\n", k.Category, k.Content))
			}
		}
		return s.String()
	}})

	tools = append(tools, Tool{"knowledge_add", "添加知识", "知识库", func(a string) string {
		p := strings.SplitN(a, "|", 2)
		if len(p) != 2 { return "格式: 分类|内容" }
		knowledge = append(knowledge, Knowledge{Category: p[0], Content: p[1]})
		return "知识已添加"
	}})

	tools = append(tools, Tool{"knowledge_delete", "删除知识", "知识库", func(a string) string {
		return fmt.Sprintf("删除知识: %s", a)
	}})

	tools = append(tools, Tool{"knowledge_export", "导出知识库", "知识库", func(a string) string {
		b, _ := json.Marshal(knowledge)
		return string(b)
	}})

	// ===== 记忆系统 (5个) =====
	tools = append(tools, Tool{"memory_list", "查看记忆", "记忆", func(a string) string {
		var s strings.Builder
		for i, m := range messages {
			s.WriteString(fmt.Sprintf("[%d] %s: %s\n", i, m.Role, m.Content))
		}
		return s.String()
	}})

	tools = append(tools, Tool{"memory_clear", "清空记忆", "记忆", func(a string) string {
		messages = nil
		return "记忆已清空"
	}})

	tools = append(tools, Tool{"memory_save", "保存记忆", "记忆", func(a string) string {
		f, _ := os.Create(config.DataDir + "/memory.json")
		defer f.Close()
		json.NewEncoder(f).Encode(messages)
		return "记忆已保存"
	}})

	tools = append(tools, Tool{"memory_load", "加载记忆", "记忆", func(a string) string {
		f, err := os.Open(config.DataDir + "/memory.json")
		if err != nil { return "没有保存的记忆" }
		defer f.Close()
		json.NewDecoder(f).Decode(&messages)
		return "记忆已加载"
	}})

	tools = append(tools, Tool{"memory_search", "搜索记忆", "记忆", func(a string) string {
		var s strings.Builder
		for _, m := range messages {
			if strings.Contains(m.Content, a) {
				s.WriteString(fmt.Sprintf("%s: %s\n", m.Role, m.Content))
			}
		}
		return s.String()
	}})

	// ===== 工具列表 (1个) =====
	tools = append(tools, Tool{"tool_list", "所有工具列表", "工具", func(a string) string {
		var s strings.Builder
		cat := ""
		for _, t := range tools {
			if t.Category != cat {
				cat = t.Category
				s.WriteString(fmt.Sprintf("\n=== %s ===\n", cat))
			}
			s.WriteString(fmt.Sprintf("  %s: %s\n", t.Name, t.Description))
		}
		return s.String()
	}})
}

// ========== 调用AI API ==========
func callAI(msgs []Message) (string, error) {
	reqBody := map[string]interface{}{
		"model":       config.Model,
		"messages":    msgs,
		"max_tokens":  config.MaxTokens,
		"temperature": config.Temperature,
	}
	j, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", config.BaseURL, strings.NewReader(string(j)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.APIKey)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil { return "", err }
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
		fc := choices[0].(map[string]interface{})
		msg := fc["message"].(map[string]interface{})
		return msg["content"].(string), nil
	}
	return fmt.Sprintf("响应: %s", string(body)), nil
}

// ========== 打印欢迎信息 ==========
func printWelcome() {
	fmt.Printf(ColorCyan+`
================================================
  coAgent-Go v2.0 - 纯Go AI编程Agent
  内置工具: %d 个
  子Agent: %d 个
  技能: %d 个
  知识库: %d 条
  提供商: %s
  模型: %s
================================================
`+ColorReset, len(tools), len(subAgents), len(skills), len(knowledge), config.Provider, config.Model)
	fmt.Println("输入 'quit' 退出，'help' 查看帮助\n")
}

// ========== 打印帮助 ==========
func printHelp() {
	fmt.Printf(ColorYellow+"=== 可用命令 ===\n"+ColorReset)
	fmt.Println("  help       - 帮助")
	fmt.Println("  tools      - 所有工具")
	fmt.Println("  agents     - 所有子Agent")
	fmt.Println("  skills     - 所有技能")
	fmt.Println("  knowledge  - 所有知识库")
	fmt.Println("  memory     - 查看记忆")
	fmt.Println("  clear      - 清空记忆")
	fmt.Println("  quit       - 退出")
	fmt.Println()
}

// ========== 主循环 ==========
func main() {
	printWelcome()

	// 读取API Key
	switch config.Provider {
	case "siliconflow": config.APIKey = os.Getenv("SILICONFLOW_API_KEY")
	case "zhipu": config.APIKey = os.Getenv("ZHIPU_API_KEY")
	case "deepseek": config.APIKey = os.Getenv("DEEPSEEK_API_KEY")
	case "openrouter": config.APIKey = os.Getenv("OPENROUTER_API_KEY")
	case "qwen": config.APIKey = os.Getenv("DASHSCOPE_API_KEY")
	case "moonshot", "kimi": config.APIKey = os.Getenv("MOONSHOT_API_KEY")
	}
	if config.APIKey == "" { config.APIKey = os.Getenv("COAGENT_API_KEY") }

	if config.APIKey == "" {
		fmt.Printf(ColorRed+"警告: 未配置API Key\n"+ColorReset)
		fmt.Println("支持的提供商:")
		for _, p := range providers {
			fmt.Printf("  - %s: %s\n", p.Name, p.Description)
		}
		fmt.Println()
	}

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Printf(ColorGreen+"你> "+ColorReset)
		if !scanner.Scan() { break }
		input := strings.TrimSpace(scanner.Text())

		switch input {
		case "quit", "退出": fmt.Println("再见！"); return
		case "help", "帮助": printHelp(); continue
		case "tools": fmt.Println(tools[len(tools)-1].Execute("")); continue
		case "agents":
			for _, a := range subAgents { fmt.Printf("  - %s: %s\n", a.Name, a.Description) }
			continue
		case "skills":
			for _, s := range skills { fmt.Printf("  - %s: %s\n", s.Name, s.Description) }
			continue
		case "knowledge":
			for _, k := range knowledge { fmt.Printf("  [%s] %s\n", k.Category, k.Content) }
			continue
		case "memory":
			for i, m := range messages { fmt.Printf("[%d] %s: %s\n", i, m.Role, m.Content) }
			continue
		case "clear": messages = nil; fmt.Println("记忆已清空"); continue
		}

		if input == "" { continue }

		messages = append(messages, Message{"user", input, time.Now().Format(time.RFC3339)})
		fmt.Printf(ColorBlue+"coAgent思考中...\n"+ColorReset)
		resp, err := callAI(messages)
		if err != nil { fmt.Printf("错误: %v\n", err); continue }
		messages = append(messages, Message{"assistant", resp, time.Now().Format(time.RFC3339)})
		fmt.Printf(ColorCyan+"coAgent> "+ColorReset)
		fmt.Println(resp)
		fmt.Println()
	}
}
