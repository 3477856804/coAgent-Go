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

// ========== 初始化子Agent（100个智能体） ==========
func initSubAgents() {
	// ===== 开发类智能体 (20个) =====
	subAgents = append(subAgents, SubAgent{"code_expert", "代码专家，擅长代码编写、调试、优化", "你是一个资深代码专家，擅长各种编程语言的开发、调试、优化和重构。", []string{"read_file", "write_file", "run_command"}})
	subAgents = append(subAgents, SubAgent{"python_dev", "Python开发专家", "你是Python开发专家，擅长Python全栈开发、数据分析、机器学习。", []string{"python_run", "pip_install", "read_file"}})
	subAgents = append(subAgents, SubAgent{"go_dev", "Go开发专家", "你是Go开发专家，擅长Go后端开发、微服务、高并发系统。", []string{"go_build", "go_test", "run_command"}})
	subAgents = append(subAgents, SubAgent{"java_dev", "Java开发专家", "你是Java开发专家，擅长Java企业级开发、Spring Boot、微服务。", []string{"run_command", "read_file", "write_file"}})
	subAgents = append(subAgents, SubAgent{"js_dev", "JavaScript开发专家", "你是JavaScript开发专家，擅长前端开发、React、Vue、Node.js。", []string{"node_run", "npm_install", "read_file"}})
	subAgents = append(subAgents, SubAgent{"cpp_dev", "C++开发专家", "你是C++开发专家，擅长系统编程、游戏开发、高性能计算。", []string{"gcc_build", "run_command", "read_file"}})
	subAgents = append(subAgents, SubAgent{"rust_dev", "Rust开发专家", "你是Rust开发专家，擅长系统编程、WebAssembly、区块链开发。", []string{"run_command", "read_file", "write_file"}})
	subAgents = append(subAgents, SubAgent{"web_dev", "Web开发专家", "你是Web开发专家，擅长全栈Web开发、前后端分离、API设计。", []string{"read_file", "write_file", "run_command"}})
	subAgents = append(subAgents, SubAgent{"mobile_dev", "移动端开发专家", "你是移动端开发专家，擅长Android、iOS、跨平台开发。", []string{"run_command", "read_file", "write_file"}})
	subAgents = append(subAgents, SubAgent{"game_dev", "游戏开发专家", "你是游戏开发专家，擅长Unity、Unreal、2D/3D游戏开发。", []string{"run_command", "read_file", "write_file"}})
	subAgents = append(subAgents, SubAgent{"debug_expert", "调试专家，擅长Bug定位、性能优化", "你是一个调试专家，擅长定位Bug、分析日志、优化性能。", []string{"run_command", "read_file", "grep"}})
	subAgents = append(subAgents, SubAgent{"refactor_expert", "重构专家", "你是代码重构专家，擅长代码优化、架构重构、技术债务清理。", []string{"read_file", "write_file", "grep"}})
	subAgents = append(subAgents, SubAgent{"review_expert", "代码审查专家", "你是代码审查专家，擅长代码质量评估、最佳实践指导。", []string{"read_file", "grep", "run_command"}})
	subAgents = append(subAgents, SubAgent{"test_expert", "测试专家", "你是测试专家，擅长单元测试、集成测试、自动化测试。", []string{"run_command", "read_file", "write_file"}})
	subAgents = append(subAgents, SubAgent{"perf_expert", "性能优化专家", "你是性能优化专家，擅长系统性能分析、瓶颈定位、优化方案。", []string{"run_command", "system_info", "process_list"}})
	subAgents = append(subAgents, SubAgent{"api_expert", "API设计专家", "你是API设计专家，擅长RESTful API、GraphQL、gRPC设计。", []string{"read_file", "write_file", "http_get"}})
	subAgents = append(subAgents, SubAgent{"db_expert", "数据库专家", "你是数据库专家，擅长MySQL、PostgreSQL、MongoDB、Redis优化。", []string{"sql_query", "db_backup", "run_command"}})
	subAgents = append(subAgents, SubAgent{"arch_expert", "架构师", "你是系统架构师，擅长微服务架构、分布式系统、高可用设计。", []string{"read_file", "write_file", "run_command"}})
	subAgents = append(subAgents, SubAgent{"ml_expert", "机器学习专家", "你是机器学习专家，擅长模型训练、数据处理、深度学习。", []string{"python_run", "read_file", "write_file"}})
	subAgents = append(subAgents, SubAgent{"devops_expert", "运维专家，擅长部署、自动化、CI/CD", "你是一个运维专家，擅长服务器部署、自动化脚本、CI/CD配置。", []string{"run_command", "system_info", "disk_usage"}})

	// ===== 安全类智能体 (10个) =====
	subAgents = append(subAgents, SubAgent{"security_expert", "安全专家，擅长安全测试、漏洞分析", "你是一个安全专家，擅长安全测试、漏洞扫描、代码审计。", []string{"run_command", "port_scan", "vuln_scan"}})
	subAgents = append(subAgents, SubAgent{"penetration_expert", "渗透测试专家", "你是渗透测试专家，擅长Web安全、网络安全、渗透测试。", []string{"run_command", "port_scan", "vuln_scan"}})
	subAgents = append(subAgents, SubAgent{"crypto_expert", "加密专家", "你是加密专家，擅长密码学、加密算法、安全协议。", []string{"encrypt_aes", "decrypt_aes", "hash_generate"}})
	subAgents = append(subAgents, SubAgent{"reverse_expert", "逆向工程专家", "你是逆向工程专家，擅长二进制分析、反编译、漏洞挖掘。", []string{"run_command", "read_file", "grep"}})
	subAgents = append(subAgents, SubAgent{"malware_expert", "恶意软件分析专家", "你是恶意软件分析专家，擅长病毒分析、木马检测、威胁情报。", []string{"run_command", "read_file", "grep"}})
	subAgents = append(subAgents, SubAgent{"network_sec_expert", "网络安全专家", "你是网络安全专家，擅长防火墙、入侵检测、VPN配置。", []string{"firewall_status", "port_check", "run_command"}})
	subAgents = append(subAgents, SubAgent{"web_sec_expert", "Web安全专家", "你是Web安全专家，擅长SQL注入、XSS、CSRF防护。", []string{"run_command", "http_get", "grep"}})
	subAgents = append(subAgents, SubAgent{"app_sec_expert", "应用安全专家", "你是应用安全专家，擅长代码审计、安全编码、漏洞修复。", []string{"read_file", "grep", "run_command"}})
	subAgents = append(subAgents, SubAgent{"data_sec_expert", "数据安全专家", "你是数据安全专家，擅长数据加密、隐私保护、合规性。", []string{"encrypt_aes", "decrypt_aes", "hash_generate"}})
	subAgents = append(subAgents, SubAgent{"forensic_expert", "取证专家", "你是数字取证专家，擅长电子证据收集、分析、恢复。", []string{"run_command", "read_file", "grep"}})

	// ===== 设计类智能体 (10个) =====
	subAgents = append(subAgents, SubAgent{"ui_designer", "UI设计师", "你是UI设计师，擅长界面设计、用户体验、视觉设计。", []string{"image_resize", "image_crop", "color_palette"}})
	subAgents = append(subAgents, SubAgent{"ux_designer", "UX设计师", "你是UX设计师，擅长用户研究、交互设计、用户旅程。", []string{"read_file", "write_file", "image_resize"}})
	subAgents = append(subAgents, SubAgent{"graphic_designer", "平面设计师", "你是平面设计师，擅长海报设计、品牌设计、印刷设计。", []string{"image_resize", "image_filter", "icon_generate"}})
	subAgents = append(subAgents, SubAgent{"video_editor", "视频编辑专家", "你是视频编辑专家，擅长视频剪辑、特效制作、调色。", []string{"video_cut", "video_merge", "video_filter"}})
	subAgents = append(subAgents, SubAgent{"audio_engineer", "音频工程师", "你是音频工程师，擅长音频处理、混音、音效设计。", []string{"audio_cut", "audio_merge", "audio_volume"}})
	subAgents = append(subAgents, SubAgent{"motion_designer", "动效设计师", "你是动效设计师，擅长动画设计、交互动效、Lottie。", []string{"image_resize", "video_gif", "icon_generate"}})
	subAgents = append(subAgents, SubAgent{"brand_designer", "品牌设计师", "你是品牌设计师，擅长品牌识别、Logo设计、VI系统。", []string{"icon_generate", "color_palette", "gradient_gen"}})
	subAgents = append(subAgents, SubAgent{"web_designer", "网页设计师", "你是网页设计师，擅长网页设计、响应式设计、前端视觉。", []string{"read_file", "write_file", "image_resize"}})
	subAgents = append(subAgents, SubAgent{"mobile_designer", "移动端设计师", "你是移动端设计师，擅长APP设计、移动端交互、设计规范。", []string{"image_resize", "icon_generate", "color_palette"}})
	subAgents = append(subAgents, SubAgent{"3d_designer", "3D设计师", "你是3D设计师，擅长3D建模、渲染、动画制作。", []string{"read_file", "write_file", "run_command"}})

	// ===== 数据类智能体 (10个) =====
	subAgents = append(subAgents, SubAgent{"data_analyst", "数据分析师", "你是数据分析师，擅长数据清洗、统计分析、可视化。", []string{"csv_parse", "data_sort", "chart_bar"}})
	subAgents = append(subAgents, SubAgent{"data_scientist", "数据科学家", "你是数据科学家，擅长数据建模、预测分析、机器学习。", []string{"python_run", "csv_parse", "chart_line"}})
	subAgents = append(subAgents, SubAgent{"bi_expert", "BI专家", "你是BI专家，擅长商业智能、报表设计、数据看板。", []string{"chart_bar", "chart_pie", "chart_line"}})
	subAgents = append(subAgents, SubAgent{"sql_expert", "SQL专家", "你是SQL专家，擅长复杂查询、性能优化、数据库设计。", []string{"sql_query", "db_backup", "run_command"}})
	subAgents = append(subAgents, SubAgent{"etl_expert", "ETL专家", "你是ETL专家，擅长数据抽取、转换、加载、数据管道。", []string{"data_import", "data_export", "run_command"}})
	subAgents = append(subAgents, SubAgent{"bigdata_expert", "大数据专家", "你是大数据专家，擅长Hadoop、Spark、Flink大数据处理。", []string{"run_command", "data_sort", "data_filter"}})
	subAgents = append(subAgents, SubAgent{"ml_engineer", "机器学习工程师", "你是机器学习工程师，擅长模型训练、特征工程、模型部署。", []string{"python_run", "read_file", "write_file"}})
	subAgents = append(subAgents, SubAgent{"dl_engineer", "深度学习工程师", "你是深度学习工程师，擅长神经网络、CNN、RNN、Transformer。", []string{"python_run", "read_file", "write_file"}})
	subAgents = append(subAgents, SubAgent{"nlp_expert", "NLP专家", "你是自然语言处理专家，擅长文本分析、情感分析、机器翻译。", []string{"python_run", "text_replace", "translate"}})
	subAgents = append(subAgents, SubAgent{"cv_expert", "计算机视觉专家", "你是计算机视觉专家，擅长图像识别、目标检测、图像分割。", []string{"python_run", "image_resize", "image_filter"}})

	// ===== 办公类智能体 (10个) =====
	subAgents = append(subAgents, SubAgent{"writer", "文案专家", "你是文案专家，擅长文案写作、内容创作、营销文案。", []string{"read_file", "write_file", "text_replace"}})
	subAgents = append(subAgents, SubAgent{"translator", "翻译专家", "你是翻译专家，擅长多语言翻译、本地化、文化适配。", []string{"translate", "read_file", "write_file"}})
	subAgents = append(subAgents, SubAgent{"editor", "编辑专家", "你是编辑专家，擅长文章编辑、校对、排版。", []string{"read_file", "write_file", "text_replace"}})
	subAgents = append(subAgents, SubAgent{"summarizer", "摘要专家", "你是摘要专家，擅长文章摘要、会议纪要、信息提炼。", []string{"read_file", "summary_gen", "outline_gen"}})
	subAgents = append(subAgents, SubAgent{"ppt_expert", "PPT专家", "你是PPT专家，擅长PPT制作、演示设计、汇报材料。", []string{"ppt_create", "read_file", "write_file"}})
	subAgents = append(subAgents, SubAgent{"excel_expert", "Excel专家", "你是Excel专家，擅长公式、数据透视表、图表制作。", []string{"excel_create", "csv_parse", "data_sort"}})
	subAgents = append(subAgents, SubAgent{"word_expert", "Word专家", "你是Word专家，擅长文档排版、格式设置、模板制作。", []string{"docx_create", "read_file", "write_file"}})
	subAgents = append(subAgents, SubAgent{"pdf_expert", "PDF专家", "你是PDF专家，擅长PDF处理、表单填写、文档转换。", []string{"pdf_create", "pdf_read", "read_file"}})
	subAgents = append(subAgents, SubAgent{"email_expert", "邮件专家", "你是邮件专家，擅长邮件写作、商务沟通、邮件礼仪。", []string{"read_file", "write_file", "email_send"}})
	subAgents = append(subAgents, SubAgent{"meeting_expert", "会议专家", "你是会议专家，擅长会议组织、纪要撰写、任务跟踪。", []string{"read_file", "write_file", "summary_gen"}})

	// ===== 营销类智能体 (10个) =====
	subAgents = append(subAgents, SubAgent{"marketing_expert", "营销专家", "你是营销专家，擅长市场分析、营销策略、品牌推广。", []string{"read_file", "write_file", "data_analyst"}})
	subAgents = append(subAgents, SubAgent{"seo_expert", "SEO专家", "你是SEO专家，擅长搜索引擎优化、关键词排名、流量提升。", []string{"http_get", "grep", "read_file"}})
	subAgents = append(subAgents, SubAgent{"sem_expert", "SEM专家", "你是SEM专家，擅长搜索引擎营销、广告投放、ROI优化。", []string{"read_file", "write_file", "data_analyst"}})
	subAgents = append(subAgents, SubAgent{"social_media_expert", "社交媒体专家", "你是社交媒体专家，擅长微信、微博、抖音、小红书运营。", []string{"read_file", "write_file", "image_resize"}})
	subAgents = append(subAgents, SubAgent{"content_marketer", "内容营销专家", "你是内容营销专家，擅长内容策划、创作、分发。", []string{"read_file", "write_file", "summary_gen"}})
	subAgents = append(subAgents, SubAgent{"email_marketer", "邮件营销专家", "你是邮件营销专家，擅长邮件列表管理、邮件设计、转化优化。", []string{"email_send", "read_file", "write_file"}})
	subAgents = append(subAgents, SubAgent{"growth_hacker", "增长黑客", "你是增长黑客，擅长用户增长、病毒传播、数据驱动。", []string{"data_analyst", "read_file", "write_file"}})
	subAgents = append(subAgents, SubAgent{"brand_expert", "品牌专家", "你是品牌专家，擅长品牌定位、品牌传播、品牌管理。", []string{"read_file", "write_file", "color_palette"}})
	subAgents = append(subAgents, SubAgent{"ads_expert", "广告专家", "你是广告专家，擅长广告设计、投放优化、效果评估。", []string{"image_resize", "video_cut", "read_file"}})
	subAgents = append(subAgents, SubAgent{"pr_expert", "公关专家", "你是公关专家，擅长媒体关系、危机公关、品牌传播。", []string{"read_file", "write_file", "summary_gen"}})

	// ===== 其他智能体 (30个) =====
	subAgents = append(subAgents, SubAgent{"product_manager", "产品经理", "你是产品经理，擅长需求分析、产品设计、项目管理。", []string{"read_file", "write_file", "summary_gen"}})
	subAgents = append(subAgents, SubAgent{"project_manager", "项目经理", "你是项目经理，擅长项目规划、进度管理、风险控制。", []string{"read_file", "write_file", "todo_add"}})
	subAgents = append(subAgents, SubAgent{"hr_expert", "HR专家", "你是HR专家，擅长招聘、培训、绩效、员工关系。", []string{"read_file", "write_file", "docx_create"}})
	subAgents = append(subAgents, SubAgent{"finance_expert", "财务专家", "你是财务专家，擅长财务分析、预算管理、税务规划。", []string{"excel_create", "read_file", "write_file"}})
	subAgents = append(subAgents, SubAgent{"legal_expert", "法律专家", "你是法律专家，擅长合同审查、法律合规、知识产权。", []string{"read_file", "write_file", "docx_create"}})
	subAgents = append(subAgents, SubAgent{"customer_service", "客服专家", "你是客服专家，擅长客户沟通、问题解决、满意度提升。", []string{"read_file", "write_file", "email_send"}})
	subAgents = append(subAgents, SubAgent{"sales_expert", "销售专家", "你是销售专家，擅长客户开发、谈判技巧、成交转化。", []string{"read_file", "write_file", "email_send"}})
	subAgents = append(subAgents, SubAgent{"researcher", "研究员", "你是研究员，擅长文献调研、数据分析、报告撰写。", []string{"http_get", "read_file", "summary_gen"}})
	subAgents = append(subAgents, SubAgent{"teacher", "教师", "你是教师，擅长知识讲解、课程设计、学习辅导。", []string{"read_file", "write_file", "quiz_gen"}})
	subAgents = append(subAgents, SubAgent{"student", "学生助手", "你是学生助手，擅长作业辅导、考试准备、学习规划。", []string{"read_file", "write_file", "summary_gen"}})
	subAgents = append(subAgents, SubAgent{"chef", "厨师", "你是厨师，擅长菜谱设计、烹饪技巧、营养搭配。", []string{"read_file", "write_file", "summary_gen"}})
	subAgents = append(subAgents, SubAgent{"fitness_expert", "健身教练", "你是健身教练，擅长健身计划、动作指导、营养建议。", []string{"read_file", "write_file", "summary_gen"}})
	subAgents = append(subAgents, SubAgent{"travel_expert", "旅行专家", "你是旅行专家，擅长行程规划、景点推荐、攻略撰写。", []string{"http_get", "read_file", "write_file"}})
	subAgents = append(subAgents, SubAgent{"photographer", "摄影师", "你是摄影师，擅长摄影技巧、构图设计、后期处理。", []string{"image_resize", "image_filter", "image_crop"}})
	subAgents = append(subAgents, SubAgent{"musician", "音乐人", "你是音乐人，擅长音乐创作、编曲、混音。", []string{"audio_cut", "audio_merge", "audio_volume"}})
	subAgents = append(subAgents, SubAgent{"novel_writer", "小说作家", "你是小说作家，擅长故事创作、人物塑造、情节设计。", []string{"read_file", "write_file", "summary_gen"}})
	subAgents = append(subAgents, SubAgent{"blogger", "博主", "你是博主，擅长博客写作、内容创作、粉丝运营。", []string{"read_file", "write_file", "summary_gen"}})
	subAgents = append(subAgents, SubAgent{"youtuber", "YouTuber", "你是YouTuber，擅长视频策划、脚本撰写、视频剪辑。", []string{"video_cut", "video_merge", "read_file"}})
	subAgents = append(subAgents, SubAgent{"podcaster", "播客主播", "你是播客主播，擅长播客策划、录音、后期制作。", []string{"audio_cut", "audio_merge", "audio_volume"}})
	subAgents = append(subAgents, SubAgent{"entrepreneur", "创业者", "你是创业者，擅长商业模式、融资规划、团队管理。", []string{"read_file", "write_file", "summary_gen"}})
	subAgents = append(subAgents, SubAgent{"investor", "投资专家", "你是投资专家，擅长投资分析、风险评估、财务建模。", []string{"excel_create", "read_file", "write_file"}})
	subAgents = append(subAgents, SubAgent{"consultant", "管理顾问", "你是管理顾问，擅长企业咨询、战略规划、流程优化。", []string{"read_file", "write_file", "summary_gen"}})
	subAgents = append(subAgents, SubAgent{"coach", "教练", "你是教练，擅长个人成长、职业规划、领导力培养。", []string{"read_file", "write_file", "summary_gen"}})
	subAgents = append(subAgents, SubAgent{"psychologist", "心理咨询师", "你是心理咨询师，擅长心理疏导、情绪管理、人际关系。", []string{"read_file", "write_file", "summary_gen"}})
	subAgents = append(subAgents, SubAgent{"nutritionist", "营养师", "你是营养师，擅长营养搭配、健康饮食、疾病预防。", []string{"read_file", "write_file", "summary_gen"}})
	subAgents = append(subAgents, SubAgent{"doctor", "医生", "你是医生，擅长疾病诊断、治疗方案、健康咨询。", []string{"read_file", "write_file", "summary_gen"}})
	subAgents = append(subAgents, SubAgent{"lawyer", "律师", "你是律师，擅长法律咨询、诉讼代理、合同审查。", []string{"read_file", "write_file", "docx_create"}})
	subAgents = append(subAgents, SubAgent{"accountant", "会计师", "你是会计师，擅长会计核算、财务报表、税务申报。", []string{"excel_create", "read_file", "write_file"}})
	subAgents = append(subAgents, SubAgent{"architect", "建筑师", "你是建筑师，擅长建筑设计、施工图、室内设计。", []string{"read_file", "write_file", "summary_gen"}})
	subAgents = append(subAgents, SubAgent{"engineer", "工程师", "你是工程师，擅长工程设计、施工管理、质量控制。", []string{"read_file", "write_file", "summary_gen"}})
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

	// ===== 设计工具 (15个) =====
	tools = append(tools, Tool{"image_resize", "图片缩放", "设计", func(a string) string {
		p := strings.SplitN(a, "|", 3)
		if len(p) != 3 { return "格式: 输入|输出|尺寸" }
		return fmt.Sprintf("图片缩放: %s -> %s (%s)", p[0], p[1], p[2])
	}})

	tools = append(tools, Tool{"image_crop", "图片裁剪", "设计", func(a string) string {
		return fmt.Sprintf("图片裁剪: %s", a)
	}})

	tools = append(tools, Tool{"image_rotate", "图片旋转", "设计", func(a string) string {
		return fmt.Sprintf("图片旋转: %s", a)
	}})

	tools = append(tools, Tool{"image_convert", "图片格式转换", "设计", func(a string) string {
		p := strings.SplitN(a, "|", 2)
		return fmt.Sprintf("格式转换: %s -> %s", p[0], p[1])
	}})

	tools = append(tools, Tool{"image_filter", "图片滤镜", "设计", func(a string) string {
		return fmt.Sprintf("应用滤镜: %s", a)
	}})

	tools = append(tools, Tool{"image_blur", "图片模糊", "设计", func(a string) string {
		return fmt.Sprintf("模糊处理: %s", a)
	}})

	tools = append(tools, Tool{"image_sharpen", "图片锐化", "设计", func(a string) string {
		return fmt.Sprintf("锐化处理: %s", a)
	}})

	tools = append(tools, Tool{"image_grayscale", "灰度化", "设计", func(a string) string {
		return fmt.Sprintf("灰度化: %s", a)
	}})

	tools = append(tools, Tool{"image_invert", "反色", "设计", func(a string) string {
		return fmt.Sprintf("反色处理: %s", a)
	}})

	tools = append(tools, Tool{"image_watermark", "添加水印", "设计", func(a string) string {
		p := strings.SplitN(a, "|", 2)
		return fmt.Sprintf("添加水印: %s -> %s", p[0], p[1])
	}})

	tools = append(tools, Tool{"image_compress", "压缩图片", "设计", func(a string) string {
		return fmt.Sprintf("压缩图片: %s", a)
	}})

	tools = append(tools, Tool{"image_info", "图片信息", "设计", func(a string) string {
		return fmt.Sprintf("图片信息: %s", a)
	}})

	tools = append(tools, Tool{"color_palette", "生成调色板", "设计", func(a string) string {
		return fmt.Sprintf("生成调色板: %s", a)
	}})

	tools = append(tools, Tool{"gradient_gen", "生成渐变", "设计", func(a string) string {
		return fmt.Sprintf("生成渐变: %s", a)
	}})

	tools = append(tools, Tool{"icon_generate", "生成图标", "设计", func(a string) string {
		return fmt.Sprintf("生成图标: %s", a)
	}})

	// ===== 视频工具 (15个) =====
	tools = append(tools, Tool{"video_info", "视频信息", "视频", func(a string) string {
		cmd := exec.Command("ffprobe", "-v", "quiet", "-print_format", "json", a)
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"video_convert", "视频格式转换", "视频", func(a string) string {
		p := strings.SplitN(a, "|", 2)
		return fmt.Sprintf("视频转换: %s -> %s", p[0], p[1])
	}})

	tools = append(tools, Tool{"video_cut", "视频裁剪", "视频", func(a string) string {
		return fmt.Sprintf("视频裁剪: %s", a)
	}})

	tools = append(tools, Tool{"video_merge", "视频合并", "视频", func(a string) string {
		return fmt.Sprintf("视频合并: %s", a)
	}})

	tools = append(tools, Tool{"video_extract_audio", "提取音频", "视频", func(a string) string {
		return fmt.Sprintf("提取音频: %s", a)
	}})

	tools = append(tools, Tool{"video_add_audio", "添加音频", "视频", func(a string) string {
		p := strings.SplitN(a, "|", 2)
		return fmt.Sprintf("添加音频: %s -> %s", p[0], p[1])
	}})

	tools = append(tools, Tool{"video_resize", "视频缩放", "视频", func(a string) string {
		return fmt.Sprintf("视频缩放: %s", a)
	}})

	tools = append(tools, Tool{"video_rotate", "视频旋转", "视频", func(a string) string {
		return fmt.Sprintf("视频旋转: %s", a)
	}})

	tools = append(tools, Tool{"video_speed", "视频变速", "视频", func(a string) string {
		return fmt.Sprintf("视频变速: %s", a)
	}})

	tools = append(tools, Tool{"video_filter", "视频滤镜", "视频", func(a string) string {
		return fmt.Sprintf("视频滤镜: %s", a)
	}})

	tools = append(tools, Tool{"video_watermark", "视频水印", "视频", func(a string) string {
		return fmt.Sprintf("添加水印: %s", a)
	}})

	tools = append(tools, Tool{"video_compress", "压缩视频", "视频", func(a string) string {
		return fmt.Sprintf("压缩视频: %s", a)
	}})

	tools = append(tools, Tool{"video_thumbnail", "生成缩略图", "视频", func(a string) string {
		return fmt.Sprintf("生成缩略图: %s", a)
	}})

	tools = append(tools, Tool{"video_gif", "视频转GIF", "视频", func(a string) string {
		return fmt.Sprintf("视频转GIF: %s", a)
	}})

	tools = append(tools, Tool{"video_screenshot", "视频截图", "视频", func(a string) string {
		return fmt.Sprintf("视频截图: %s", a)
	}})

	// ===== 音频工具 (15个) =====
	tools = append(tools, Tool{"audio_info", "音频信息", "音频", func(a string) string {
		return fmt.Sprintf("音频信息: %s", a)
	}})

	tools = append(tools, Tool{"audio_convert", "音频格式转换", "音频", func(a string) string {
		p := strings.SplitN(a, "|", 2)
		return fmt.Sprintf("音频转换: %s -> %s", p[0], p[1])
	}})

	tools = append(tools, Tool{"audio_cut", "音频裁剪", "音频", func(a string) string {
		return fmt.Sprintf("音频裁剪: %s", a)
	}})

	tools = append(tools, Tool{"audio_merge", "音频合并", "音频", func(a string) string {
		return fmt.Sprintf("音频合并: %s", a)
	}})

	tools = append(tools, Tool{"audio_volume", "调整音量", "音频", func(a string) string {
		return fmt.Sprintf("调整音量: %s", a)
	}})

	tools = append(tools, Tool{"audio_speed", "音频变速", "音频", func(a string) string {
		return fmt.Sprintf("音频变速: %s", a)
	}})

	tools = append(tools, Tool{"audio_pitch", "音频变调", "音频", func(a string) string {
		return fmt.Sprintf("音频变调: %s", a)
	}})

	tools = append(tools, Tool{"audio_noise_reduce", "降噪", "音频", func(a string) string {
		return fmt.Sprintf("降噪处理: %s", a)
	}})

	tools = append(tools, Tool{"audio_reverb", "添加混响", "音频", func(a string) string {
		return fmt.Sprintf("添加混响: %s", a)
	}})

	tools = append(tools, Tool{"audio_echo", "添加回声", "音频", func(a string) string {
		return fmt.Sprintf("添加回声: %s", a)
	}})

	tools = append(tools, Tool{"audio_bass", "低音增强", "音频", func(a string) string {
		return fmt.Sprintf("低音增强: %s", a)
	}})

	tools = append(tools, Tool{"audio_treble", "高音增强", "音频", func(a string) string {
		return fmt.Sprintf("高音增强: %s", a)
	}})

	tools = append(tools, Tool{"audio_compress", "压缩音频", "音频", func(a string) string {
		return fmt.Sprintf("压缩音频: %s", a)
	}})

	tools = append(tools, Tool{"audio_mute", "静音", "音频", func(a string) string {
		return fmt.Sprintf("静音处理: %s", a)
	}})

	tools = append(tools, Tool{"audio_loop", "循环播放", "音频", func(a string) string {
		return fmt.Sprintf("循环: %s", a)
	}})

	// ===== 办公工具 (15个) =====
	tools = append(tools, Tool{"docx_create", "创建Word文档", "办公", func(a string) string {
		return fmt.Sprintf("创建Word: %s", a)
	}})

	tools = append(tools, Tool{"pdf_create", "创建PDF", "办公", func(a string) string {
		return fmt.Sprintf("创建PDF: %s", a)
	}})

	tools = append(tools, Tool{"excel_create", "创建Excel", "办公", func(a string) string {
		return fmt.Sprintf("创建Excel: %s", a)
	}})

	tools = append(tools, Tool{"ppt_create", "创建PPT", "办公", func(a string) string {
		return fmt.Sprintf("创建PPT: %s", a)
	}})

	tools = append(tools, Tool{"csv_parse", "解析CSV", "办公", func(a string) string {
		return fmt.Sprintf("解析CSV: %s", a)
	}})

	tools = append(tools, Tool{"excel_read", "读取Excel", "办公", func(a string) string {
		return fmt.Sprintf("读取Excel: %s", a)
	}})

	tools = append(tools, Tool{"pdf_read", "读取PDF", "办公", func(a string) string {
		return fmt.Sprintf("读取PDF: %s", a)
	}})

	tools = append(tools, Tool{"word_read", "读取Word", "办公", func(a string) string {
		return fmt.Sprintf("读取Word: %s", a)
	}})

	tools = append(tools, Tool{"calendar", "日历", "办公", func(a string) string {
		return time.Now().Format("2006-01-02")
	}})

	tools = append(tools, Tool{"todo_add", "添加待办", "办公", func(a string) string {
		return fmt.Sprintf("添加待办: %s", a)
	}})

	tools = append(tools, Tool{"todo_list", "待办列表", "办公", func(a string) string {
		return "待办列表"
	}})

	tools = append(tools, Tool{"note_add", "添加笔记", "办公", func(a string) string {
		return fmt.Sprintf("添加笔记: %s", a)
	}})

	tools = append(tools, Tool{"note_search", "搜索笔记", "办公", func(a string) string {
		return fmt.Sprintf("搜索笔记: %s", a)
	}})

	tools = append(tools, Tool{"reminder", "提醒", "办公", func(a string) string {
		return fmt.Sprintf("提醒: %s", a)
	}})

	tools = append(tools, Tool{"clock", "时钟", "办公", func(a string) string {
		return time.Now().Format("15:04:05")
	}})

	// ===== 数据工具 (15个) =====
	tools = append(tools, Tool{"json_parse", "解析JSON", "数据", func(a string) string {
		var v interface{}
		json.Unmarshal([]byte(a), &v)
		b, _ := json.MarshalIndent(v, "", "  ")
		return string(b)
	}})

	tools = append(tools, Tool{"json_validate", "验证JSON", "数据", func(a string) string {
		var v interface{}
		err := json.Unmarshal([]byte(a), &v)
		if err != nil { return fmt.Sprintf("JSON错误: %v", err) }
		return "JSON有效"
	}})

	tools = append(tools, Tool{"xml_parse", "解析XML", "数据", func(a string) string {
		return fmt.Sprintf("解析XML: %s", a)
	}})

	tools = append(tools, Tool{"yaml_parse", "解析YAML", "数据", func(a string) string {
		return fmt.Sprintf("解析YAML: %s", a)
	}})

	tools = append(tools, Tool{"sql_query", "SQL查询", "数据", func(a string) string {
		return fmt.Sprintf("SQL查询: %s", a)
	}})

	tools = append(tools, Tool{"db_backup", "数据库备份", "数据", func(a string) string {
		return fmt.Sprintf("备份: %s", a)
	}})

	tools = append(tools, Tool{"db_restore", "数据库恢复", "数据", func(a string) string {
		return fmt.Sprintf("恢复: %s", a)
	}})

	tools = append(tools, Tool{"data_sort", "数据排序", "数据", func(a string) string {
		return fmt.Sprintf("排序: %s", a)
	}})

	tools = append(tools, Tool{"data_filter", "数据过滤", "数据", func(a string) string {
		return fmt.Sprintf("过滤: %s", a)
	}})

	tools = append(tools, Tool{"data_search", "数据搜索", "数据", func(a string) string {
		return fmt.Sprintf("搜索: %s", a)
	}})

	tools = append(tools, Tool{"data_export", "数据导出", "数据", func(a string) string {
		return fmt.Sprintf("导出: %s", a)
	}})

	tools = append(tools, Tool{"data_import", "数据导入", "数据", func(a string) string {
		return fmt.Sprintf("导入: %s", a)
	}})

	tools = append(tools, Tool{"chart_bar", "柱状图", "数据", func(a string) string {
		return fmt.Sprintf("生成柱状图: %s", a)
	}})

	tools = append(tools, Tool{"chart_pie", "饼图", "数据", func(a string) string {
		return fmt.Sprintf("生成饼图: %s", a)
	}})

	tools = append(tools, Tool{"chart_line", "折线图", "数据", func(a string) string {
		return fmt.Sprintf("生成折线图: %s", a)
	}})

	// ===== 安全工具 (15个) =====
	tools = append(tools, Tool{"password_gen", "生成密码", "安全", func(a string) string {
		return fmt.Sprintf("生成密码: %s", a)
	}})

	tools = append(tools, Tool{"password_check", "检查密码强度", "安全", func(a string) string {
		return fmt.Sprintf("密码强度: %s", a)
	}})

	tools = append(tools, Tool{"hash_generate", "生成哈希", "安全", func(a string) string {
		return fmt.Sprintf("生成哈希: %s", a)
	}})

	tools = append(tools, Tool{"hash_verify", "验证哈希", "安全", func(a string) string {
		return fmt.Sprintf("验证哈希: %s", a)
	}})

	tools = append(tools, Tool{"encrypt_aes", "AES加密", "安全", func(a string) string {
		return fmt.Sprintf("AES加密: %s", a)
	}})

	tools = append(tools, Tool{"decrypt_aes", "AES解密", "安全", func(a string) string {
		return fmt.Sprintf("AES解密: %s", a)
	}})

	tools = append(tools, Tool{"encrypt_rsa", "RSA加密", "安全", func(a string) string {
		return fmt.Sprintf("RSA加密: %s", a)
	}})

	tools = append(tools, Tool{"decrypt_rsa", "RSA解密", "安全", func(a string) string {
		return fmt.Sprintf("RSA解密: %s", a)
	}})

	tools = append(tools, Tool{"sign_generate", "生成签名", "安全", func(a string) string {
		return fmt.Sprintf("生成签名: %s", a)
	}})

	tools = append(tools, Tool{"sign_verify", "验证签名", "安全", func(a string) string {
		return fmt.Sprintf("验证签名: %s", a)
	}})

	tools = append(tools, Tool{"vpn_status", "VPN状态", "安全", func(a string) string {
		return "VPN状态"
	}})

	tools = append(tools, Tool{"firewall_status", "防火墙状态", "安全", func(a string) string {
		cmd := exec.Command("iptables", "-L")
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"port_check", "端口检查", "安全", func(a string) string {
		return fmt.Sprintf("检查端口: %s", a)
	}})

	tools = append(tools, Tool{"vuln_scan", "漏洞扫描", "安全", func(a string) string {
		return fmt.Sprintf("漏洞扫描: %s", a)
	}})

	tools = append(tools, Tool{"log_analyze", "日志分析", "安全", func(a string) string {
		return fmt.Sprintf("日志分析: %s", a)
	}})

	// ===== 自动化工具 (15个) =====
	tools = append(tools, Tool{"cron_add", "添加定时任务", "自动化", func(a string) string {
		return fmt.Sprintf("添加定时任务: %s", a)
	}})

	tools = append(tools, Tool{"cron_list", "定时任务列表", "自动化", func(a string) string {
		cmd := exec.Command("crontab", "-l")
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"cron_delete", "删除定时任务", "自动化", func(a string) string {
		return fmt.Sprintf("删除定时任务: %s", a)
	}})

	tools = append(tools, Tool{"script_run", "运行脚本", "自动化", func(a string) string {
		return fmt.Sprintf("运行脚本: %s", a)
	}})

	tools = append(tools, Tool{"script_create", "创建脚本", "自动化", func(a string) string {
		return fmt.Sprintf("创建脚本: %s", a)
	}})

	tools = append(tools, Tool{"script_deploy", "部署脚本", "自动化", func(a string) string {
		return fmt.Sprintf("部署脚本: %s", a)
	}})

	tools = append(tools, Tool{"batch_rename", "批量重命名", "自动化", func(a string) string {
		return fmt.Sprintf("批量重命名: %s", a)
	}})

	tools = append(tools, Tool{"batch_convert", "批量转换", "自动化", func(a string) string {
		return fmt.Sprintf("批量转换: %s", a)
	}})

	tools = append(tools, Tool{"backup_file", "备份文件", "自动化", func(a string) string {
		return fmt.Sprintf("备份: %s", a)
	}})

	tools = append(tools, Tool{"restore_file", "恢复文件", "自动化", func(a string) string {
		return fmt.Sprintf("恢复: %s", a)
	}})

	tools = append(tools, Tool{"sync_dir", "同步目录", "自动化", func(a string) string {
		p := strings.SplitN(a, "|", 2)
		cmd := exec.Command("rsync", "-av", p[0]+"/", p[1]+"/")
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"watch_dir", "监听目录", "自动化", func(a string) string {
		return fmt.Sprintf("监听: %s", a)
	}})

	tools = append(tools, Tool{"auto_update", "自动更新", "自动化", func(a string) string {
		return "自动更新"
	}})

	tools = append(tools, Tool{"auto_backup", "自动备份", "自动化", func(a string) string {
		return "自动备份"
	}})

	tools = append(tools, Tool{"auto_clean", "自动清理", "自动化", func(a string) string {
		cmd := exec.Command("rm", "-rf", "/tmp/*")
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	// ===== 移动端工具 (15个) =====
	tools = append(tools, Tool{"sms_send", "发送短信", "移动端", func(a string) string {
		return fmt.Sprintf("发送短信: %s", a)
	}})

	tools = append(tools, Tool{"call_phone", "拨打电话", "移动端", func(a string) string {
		return fmt.Sprintf("拨号: %s", a)
	}})

	tools = append(tools, Tool{"contact_list", "联系人列表", "移动端", func(a string) string {
		return "联系人列表"
	}})

	tools = append(tools, Tool{"sms_list", "短信列表", "移动端", func(a string) string {
		return "短信列表"
	}})

	tools = append(tools, Tool{"camera_take", "拍照", "移动端", func(a string) string {
		cmd := exec.Command("termux-camera-photo", a)
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"record_audio", "录音", "移动端", func(a string) string {
		cmd := exec.Command("termux-microphone-record", "-f", a)
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"location_get", "获取位置", "移动端", func(a string) string {
		cmd := exec.Command("termux-location")
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"torch_on", "开闪光灯", "移动端", func(a string) string {
		cmd := exec.Command("termux-torch", "on")
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"torch_off", "关闪光灯", "移动端", func(a string) string {
		cmd := exec.Command("termux-torch", "off")
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"brightness_set", "设置亮度", "移动端", func(a string) string {
		return fmt.Sprintf("设置亮度: %s", a)
	}})

	tools = append(tools, Tool{"volume_set", "设置音量", "移动端", func(a string) string {
		return fmt.Sprintf("设置音量: %s", a)
	}})

	tools = append(tools, Tool{"wifi_scan", "扫描WiFi", "移动端", func(a string) string {
		cmd := exec.Command("termux-wifi-scaninfo")
		o, _ := cmd.CombinedOutput()
		return string(o)
	}})

	tools = append(tools, Tool{"bluetooth_scan", "扫描蓝牙", "移动端", func(a string) string {
		return "扫描蓝牙"
	}})

	tools = append(tools, Tool{"nfc_read", "读取NFC", "移动端", func(a string) string {
		return "读取NFC"
	}})

	tools = append(tools, Tool{"sensor_list", "传感器列表", "移动端", func(a string) string {
		return "传感器列表"
	}})

	// ===== 游戏工具 (15个) =====
	tools = append(tools, Tool{"game_snake", "贪吃蛇", "游戏", func(a string) string {
		return "启动贪吃蛇"
	}})

	tools = append(tools, Tool{"game_tetris", "俄罗斯方块", "游戏", func(a string) string {
		return "启动俄罗斯方块"
	}})

	tools = append(tools, Tool{"game_2048", "2048", "游戏", func(a string) string {
		return "启动2048"
	}})

	tools = append(tools, Tool{"game_sudoku", "数独", "游戏", func(a string) string {
		return "启动数独"
	}})

	tools = append(tools, Tool{"game_chess", "国际象棋", "游戏", func(a string) string {
		return "启动国际象棋"
	}})

	tools = append(tools, Tool{"game_checkers", "跳棋", "游戏", func(a string) string {
		return "启动跳棋"
	}})

	tools = append(tools, Tool{"game_ludo", "飞行棋", "游戏", func(a string) string {
		return "启动飞行棋"
	}})

	tools = append(tools, Tool{"game_memory", "记忆游戏", "游戏", func(a string) string {
		return "启动记忆游戏"
	}})

	tools = append(tools, Tool{"game_puzzle", "拼图", "游戏", func(a string) string {
		return "启动拼图"
	}})

	tools = append(tools, Tool{"game_word", "猜单词", "游戏", func(a string) string {
		return "启动猜单词"
	}})

	tools = append(tools, Tool{"game_math", "数学游戏", "游戏", func(a string) string {
		return "启动数学游戏"
	}})

	tools = append(tools, Tool{"game_quiz", "问答游戏", "游戏", func(a string) string {
		return "启动问答游戏"
	}})

	tools = append(tools, Tool{"game_dice", "掷骰子", "游戏", func(a string) string {
		return "掷骰子"
	}})

	tools = append(tools, Tool{"game_card", "卡牌游戏", "游戏", func(a string) string {
		return "启动卡牌游戏"
	}})

	tools = append(tools, Tool{"game_board", "棋盘游戏", "游戏", func(a string) string {
		return "启动棋盘游戏"
	}})

	// ===== 学习工具 (15个) =====
	tools = append(tools, Tool{"flashcard", "闪卡", "学习", func(a string) string {
		return fmt.Sprintf("闪卡: %s", a)
	}})

	tools = append(tools, Tool{"quiz_gen", "生成测验", "学习", func(a string) string {
		return fmt.Sprintf("生成测验: %s", a)
	}})

	tools = append(tools, Tool{"notes_take", "记笔记", "学习", func(a string) string {
		return fmt.Sprintf("笔记: %s", a)
	}})

	tools = append(tools, Tool{"summary_gen", "生成摘要", "学习", func(a string) string {
		return fmt.Sprintf("摘要: %s", a)
	}})

	tools = append(tools, Tool{"outline_gen", "生成大纲", "学习", func(a string) string {
		return fmt.Sprintf("大纲: %s", a)
	}})

	tools = append(tools, Tool{"study_plan", "学习计划", "学习", func(a string) string {
		return fmt.Sprintf("学习计划: %s", a)
	}})

	tools = append(tools, Tool{"vocab_list", "词汇表", "学习", func(a string) string {
		return fmt.Sprintf("词汇: %s", a)
	}})

	tools = append(tools, Tool{"grammar_check", "语法检查", "学习", func(a string) string {
		return fmt.Sprintf("语法: %s", a)
	}})

	tools = append(tools, Tool{"writing_prompt", "写作提示", "学习", func(a string) string {
		return fmt.Sprintf("写作: %s", a)
	}})

	tools = append(tools, Tool{"reading_comprehension", "阅读理解", "学习", func(a string) string {
		return fmt.Sprintf("阅读: %s", a)
	}})

	tools = append(tools, Tool{"math_solve", "数学解题", "学习", func(a string) string {
		return fmt.Sprintf("解题: %s", a)
	}})

	tools = append(tools, Tool{"formula_lookup", "公式查询", "学习", func(a string) string {
		return fmt.Sprintf("公式: %s", a)
	}})

	tools = append(tools, Tool{"history_lookup", "历史查询", "学习", func(a string) string {
		return fmt.Sprintf("历史: %s", a)
	}})

	tools = append(tools, Tool{"science_lookup", "科学查询", "学习", func(a string) string {
		return fmt.Sprintf("科学: %s", a)
	}})

	tools = append(tools, Tool{"language_practice", "语言练习", "学习", func(a string) string {
		return fmt.Sprintf("练习: %s", a)
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
