package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// --- 数据结构 ---

type Node struct {
	Type              string // vless, hysteria2, ss
	Name              string
	Server            string
	Port              string
	UUID              string // VLESS的UUID
	Password          string // Hy2/SS的密码
	Cipher            string // SS的加密方式
	ServerName        string // SNI
	PublicKey         string // Reality公钥
	ShortID           string // Reality ShortID
	ClientFingerprint string // fp
	SkipCertVerify    bool   // insecure
}

// 模式配置参数
type ModeConfig struct {
	Name            string
	IsProvider      bool   // ★ 0号：ShellClash专用 (只输出节点)
	IsMini          bool   
	IsFull          bool   
	IsNoReject      bool   
	UseAdblockPlus  bool   
	AutoGroupType   string 
	UseCountryGroup bool   
	TargetNetflix   string 
	TargetGoogle    string 
}

// 规则源 (ACL4SSR)
const (
	UrlLan          = "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/LocalAreaNetwork.list"
	UrlBanAD        = "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/BanAD.list"
	UrlBanProgramAD = "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/BanProgramAD.list"
	UrlChinaDomain  = "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/ChinaDomain.list"
	UrlChinaIP      = "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/ChinaIp.list"
	UrlProxyLite    = "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/ProxyLite.list"
	UrlApple        = "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Apple.list"
	UrlMicrosoft    = "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Microsoft.list"
	UrlGoogle       = "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/GoogleCN.list"
	UrlTelegram     = "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Telegram.list"
	UrlNetflix      = "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Netflix.list"
	UrlMedia        = "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/ProxyMedia.list"
	UrlSteamCN      = "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Ruleset/SteamCN.list"
	UrlGames        = "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/ProxyGFWlist.list"
	UrlOneDrive     = "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/OneDrive.list"
)

func main() {
	// 防闪退
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("程序发生错误: %v\n按回车退出...", r)
			bufio.NewReader(os.Stdin).ReadBytes('\n')
		}
	}()

	outputFile := "config.yaml"
	var nodes []Node
	
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("=============================================================================")
	fmt.Println("          SS/VLESS/Hy2 转 Clash (v1.2 终极版)")
	fmt.Println("=============================================================================")
	
	// --- 1. 读取链接 ---
	fmt.Println(">>> 步骤1: 请粘贴链接 (支持 ss:// vless:// hy2://)")
	fmt.Println("    (支持多行，粘贴完毕后输入 ok 并回车)")
	fmt.Println("-----------------------------------------------------------------------------")

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.ToLower(line) == "ok" || strings.ToLower(line) == "done" {
			break
		}
		if line == "" { continue }
		
		// 自动识别协议
		if strings.HasPrefix(line, "vless://") {
			node, err := parseVless(line)
			if err != nil {
				fmt.Printf(" [VLESS错误] %v\n", err)
			} else {
				nodes = append(nodes, node)
				fmt.Printf(" [VLESS] %s\n", node.Name)
			}
		} else if strings.HasPrefix(line, "hy2://") || strings.HasPrefix(line, "hysteria2://") {
			node, err := parseHy2(line)
			if err != nil {
				fmt.Printf(" [Hy2错误] %v\n", err)
			} else {
				nodes = append(nodes, node)
				fmt.Printf(" [Hy2] %s\n", node.Name)
			}
		} else if strings.HasPrefix(line, "ss://") {
			node, err := parseSS(line)
			if err != nil {
				fmt.Printf(" [SS错误] %v\n", err)
			} else {
				nodes = append(nodes, node)
				fmt.Printf(" [SS] %s\n", node.Name)
			}
		}
	}

	if len(nodes) == 0 {
		fmt.Println("❌ 未检测到有效节点，请重启。")
		pause(scanner)
		return
	}

	// --- 2. 读取自定义规则 ---
	customRules := readCustomRules(scanner)

	// --- 3. 选择模式 ---
	modeIndex := showMenu(scanner)
	config := getModeConfig(modeIndex)
	
	fmt.Printf("\n🚀 正在生成 [%s] ...\n", config.Name)
	
	if config.IsProvider {
		fmt.Println("ℹ️  Provider 模式：仅生成节点列表。")
		fmt.Println("👉 请将生成的文件导入 ShellClash，然后在菜单里选择【规则模板】(如 DustinWin)。")
	} else {
		// 复杂模式
		if customRules != "" {
			fmt.Println("ℹ️  检测到自定义规则，将智能剔除 ACL4SSR 在线规则的重复项...")
		} else {
			fmt.Println("⏳ 正在并发下载 ACL4SSR 规则库...")
		}
	}

	// --- 4. 生成内容 ---
	content := generateYaml(nodes, config, customRules)

	// --- 5. 写入文件 ---
	err := os.WriteFile(outputFile, []byte(content), 0644)
	if err != nil {
		fmt.Printf("❌ 写入失败: %v\n", err)
	} else {
		fmt.Println("=============================================================================")
		fmt.Printf("✅ 成功！已生成文件: %s\n", outputFile)
		if config.IsProvider {
			fmt.Println("★ 文件类型：Provider (仅节点，供 ShellClash 在线规则使用)")
		} else {
			fmt.Println("★ 文件类型：ACL4SSR 完整配置 (含分流规则)")
		}
		fmt.Println("=============================================================================")
	}
	
	pause(scanner)
}

func readCustomRules(scanner *bufio.Scanner) string {
	fmt.Println("\n>>> 步骤2: 请粘贴自定义规则")
	fmt.Println("    (如果是模式 0，此步骤会被忽略，直接输 ok)")
	fmt.Println("    -----------------------------------------------------------------------------")

	var sb strings.Builder
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.ToLower(line) == "ok" || strings.ToLower(line) == "done" { break }
		if line == "" { continue }
		sb.WriteString("  " + line + "\n")
	}
	return sb.String()
}

func showMenu(scanner *bufio.Scanner) int {
	fmt.Println("\n>>> 步骤3: 请选择模式:")
	fmt.Println("-----------------------------------------------------------------------------")
	fmt.Println(" [0]  ★ ShellClash 专用源 (Provider) - 推荐")
	fmt.Println("      说明：只生成 proxies 列表，给 ShellClash 导入后配合 DustinWin 规则使用。")
	fmt.Println("-----------------------------------------------------------------------------")
	fmt.Println(" [1]  ACL4SSR_Online 默认版")
	fmt.Println(" [2]  ACL4SSR_Online_AdblockPlus 更多去广告")
	fmt.Println(" [3]  ACL4SSR_Online_MultiCountry 多国分组")
	fmt.Println(" [4]  ACL4SSR_Online_NoAuto 无自动测速")
	fmt.Println(" [5]  ACL4SSR_Online_NoReject 无广告拦截")
	fmt.Println(" [6]  ACL4SSR_Online_Mini 精简版 (★默认)")
	fmt.Println(" [7]  ACL4SSR_Online_Mini_AdblockPlus 精简版+更多去广告")
	fmt.Println(" [8]  ACL4SSR_Online_Mini_NoAuto 精简版+无自动测速")
	fmt.Println(" [9]  ACL4SSR_Online_Mini_Fallback 精简版+故障转移")
	fmt.Println(" [10] ACL4SSR_Online_Mini_MultiMode 精简版+多模式")
	fmt.Println(" [11] ACL4SSR_Online_Mini_MultiCountry 精简版+多国分组")
	fmt.Println(" [12] ACL4SSR_Online_Full 全分组")
	fmt.Println(" [13] ACL4SSR_Online_Full_MultiMode 全分组+多模式")
	fmt.Println(" [14] ACL4SSR_Online_Full_NoAuto 全分组+无自动测速")
	fmt.Println(" [15] ACL4SSR_Online_Full_AdblockPlus 全分组+更多去广告")
	fmt.Println(" [16] ACL4SSR_Online_Full_Netflix 全分组+奈飞加强")
	fmt.Println(" [17] ACL4SSR_Online_Full_Google 全分组+谷歌细分")
	fmt.Println("-----------------------------------------------------------------------------")
	fmt.Print("👉 请输入数字 (直接回车默认选 6): ")

	if scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text == "" { return 6 }
		val, err := strconv.Atoi(text)
		if err == nil { return 6 }
		if val >= 0 && val <= 17 { return val }
		return 6
	}
	return 6
}

func getModeConfig(mode int) ModeConfig {
	c := ModeConfig{AutoGroupType: "url-test", TargetNetflix: "🎥 奈飞视频", TargetGoogle: "📢 谷歌服务"}
	switch mode {
	case 0:
		c.Name = "ShellClash Provider (纯节点)"
		c.IsProvider = true // ★ 仅输出 proxies
	case 1: c.Name = "ACL4SSR_Online 默认版"
	case 2: c.Name = "ACL4SSR_Online_AdblockPlus"; c.UseAdblockPlus = true
	case 3: c.Name = "ACL4SSR_Online_MultiCountry"; c.UseCountryGroup = true
	case 4: c.Name = "ACL4SSR_Online_NoAuto"; c.AutoGroupType = "select"
	case 5: c.Name = "ACL4SSR_Online_NoReject"; c.IsNoReject = true
	case 6: c.Name = "ACL4SSR_Online_Mini"; c.IsMini = true
	case 7: c.Name = "ACL4SSR_Online_Mini_AdblockPlus"; c.IsMini = true; c.UseAdblockPlus = true
	case 8: c.Name = "ACL4SSR_Online_Mini_NoAuto"; c.IsMini = true; c.AutoGroupType = "select"
	case 9: c.Name = "ACL4SSR_Online_Mini_Fallback"; c.IsMini = true; c.AutoGroupType = "fallback"
	case 10: c.Name = "ACL4SSR_Online_Mini_MultiMode"; c.IsMini = true; c.AutoGroupType = "all"
	case 11: c.Name = "ACL4SSR_Online_Mini_MultiCountry"; c.IsMini = true; c.UseCountryGroup = true
	case 12: c.Name = "ACL4SSR_Online_Full"; c.IsFull = true
	case 13: c.Name = "ACL4SSR_Online_Full_MultiMode"; c.IsFull = true; c.AutoGroupType = "all"
	case 14: c.Name = "ACL4SSR_Online_Full_NoAuto"; c.IsFull = true; c.AutoGroupType = "select"
	case 15: c.Name = "ACL4SSR_Online_Full_AdblockPlus"; c.IsFull = true; c.UseAdblockPlus = true
	case 16: c.Name = "ACL4SSR_Online_Full_Netflix"; c.IsFull = true
	case 17: c.Name = "ACL4SSR_Online_Full_Google"; c.IsFull = true
	default:
		c.Name = "ACL4SSR_Online_Mini"
		c.IsMini = true
	}
	if c.IsMini {
		c.TargetNetflix = "🚀 节点选择"
		c.TargetGoogle = "🚀 节点选择"
	}
	return c
}

func generateYaml(nodes []Node, c ModeConfig, customRules string) string {
	var sb strings.Builder

	// --- 0. 如果是 Provider 模式，只输出 proxies 块 ---
	if c.IsProvider {
		sb.WriteString("proxies:\n")
		for _, n := range nodes {
			writeNode(&sb, n)
		}
		return sb.String()
	}

	// --- 1. 基础头部 (Config模式) ---
	sb.WriteString("port: 7890\nsocks-port: 7891\nallow-lan: true\nmode: Rule\nlog-level: info\nexternal-controller: 127.0.0.1:9090\n")

	// --- 2. 写入节点 ---
	sb.WriteString("\nproxies:\n")
	for _, n := range nodes {
		writeNode(&sb, n)
	}

	// --- 4. 复杂 ACL4SSR 模式逻辑 ---
	countryGroups := map[string][]string{}
	if c.UseCountryGroup {
		countryGroups = classifyNodes(nodes)
	}

	sb.WriteString("\nproxy-groups:\n")
	sb.WriteString("  - name: 🚀 节点选择\n    type: select\n    proxies:\n")
	if c.AutoGroupType == "all" {
		sb.WriteString("      - ♻️ 自动选择\n      - 🔯 故障转移\n      - ⚖️ 负载均衡\n")
	} else {
		sb.WriteString("      - ♻️ 自动选择\n")
	}
	
	if c.UseCountryGroup {
		for _, name := range []string{"HK", "TW", "JP", "SG", "US", "Other"} {
			if len(countryGroups[name]) > 0 {
				sb.WriteString(fmt.Sprintf("      - %s\n", getCountryGroupName(name)))
			}
		}
	}
	for _, n := range nodes { sb.WriteString(fmt.Sprintf("      - %s\n", n.Name)) }

	if c.AutoGroupType == "all" {
		writeAutoGroup(&sb, "♻️ 自动选择", "url-test", nodes)
		writeAutoGroup(&sb, "🔯 故障转移", "fallback", nodes)
		writeAutoGroup(&sb, "⚖️ 负载均衡", "load-balance", nodes)
	} else {
		writeAutoGroup(&sb, "♻️ 自动选择", c.AutoGroupType, nodes)
	}

	if !c.IsMini {
		writeProxyGroup(&sb, "📲 电报消息", "select")
		writeProxyGroup(&sb, "📹 油管视频", "select")
		writeProxyGroup(&sb, "🎥 奈飞视频", "select")
		writeProxyGroup(&sb, "🌍 国外媒体", "select")
		writeProxyGroup(&sb, "Ⓜ️ 微软服务", "select")
		writeProxyGroup(&sb, "📢 谷歌服务", "select")
		writeProxyGroup(&sb, "🍎 苹果服务", "select")
		if c.IsFull {
			writeProxyGroup(&sb, "🎮 游戏服务", "select")
			writeProxyGroup(&sb, "☁️ 微软云盘", "select")
			writeProxyGroup(&sb, "🚂 Steam", "select")
		}
	}
	if !c.IsNoReject {
		sb.WriteString("  - name: 🛑 广告拦截\n    type: select\n    proxies:\n      - REJECT\n      - DIRECT\n")
	}
	sb.WriteString("  - name: 🎯 全球直连\n    type: select\n    proxies:\n      - DIRECT\n      - 🚀 节点选择\n")
	sb.WriteString("  - name: 🐟 漏网之鱼\n    type: select\n    proxies:\n      - 🚀 节点选择\n      - DIRECT\n")

	sb.WriteString("\nrules:\n")
	
	// 智能去重逻辑
	exclusionMap := make(map[string]bool)
	if customRules != "" {
		sb.WriteString(customRules)
		lines := strings.Split(customRules, "\n")
		for _, line := range lines {
			parts := strings.Split(line, ",")
			if len(parts) >= 2 {
				exclusionMap[strings.TrimSpace(strings.ToLower(parts[1]))] = true
			}
		}
	}

	rules := downloadRules()
	processRule(&sb, rules[UrlLan], "🎯 全球直连", "", exclusionMap)
	if !c.IsNoReject {
		processRule(&sb, rules[UrlBanAD], "🛑 广告拦截", "", exclusionMap)
		if c.UseAdblockPlus { processRule(&sb, rules[UrlBanProgramAD], "🛑 广告拦截", "", exclusionMap) }
	}
	if !c.IsMini {
		processRule(&sb, rules[UrlMicrosoft], "Ⓜ️ 微软服务", "", exclusionMap)
		processRule(&sb, rules[UrlApple], "🍎 苹果服务", "", exclusionMap)
		processRule(&sb, rules[UrlGoogle], c.TargetGoogle, "", exclusionMap)
		processRule(&sb, rules[UrlTelegram], "📲 电报消息", "", exclusionMap)
		processRule(&sb, rules[UrlNetflix], c.TargetNetflix, "", exclusionMap)
		processRule(&sb, rules[UrlProxyLite], "🚀 节点选择", "", exclusionMap)
		if c.IsFull {
			processRule(&sb, rules[UrlOneDrive], "☁️ 微软云盘", "", exclusionMap)
			processRule(&sb, rules[UrlSteamCN], "🚂 Steam", "", exclusionMap)
			processRule(&sb, rules[UrlGames], "🎮 游戏服务", "", exclusionMap)
		}
		processRule(&sb, rules[UrlMedia], "🌍 国外媒体", "", exclusionMap)
	} else {
		processRule(&sb, rules[UrlProxyLite], "🚀 节点选择", "", exclusionMap)
		processRule(&sb, rules[UrlGoogle], "🚀 节点选择", "", exclusionMap)
		processRule(&sb, rules[UrlTelegram], "🚀 节点选择", "", exclusionMap)
	}
	processRule(&sb, rules[UrlChinaDomain], "🎯 全球直连", "", exclusionMap)
	processRule(&sb, rules[UrlChinaIP], "🎯 全球直连", "no-resolve", exclusionMap)
	sb.WriteString("  - MATCH,🐟 漏网之鱼\n")

	return sb.String()
}

// --- 辅助函数 ---

func writeNode(sb *strings.Builder, n Node) {
	if n.Type == "vless" {
		sb.WriteString(fmt.Sprintf("  - {name: %s, server: %s, port: %s, type: vless, tls: true, packet-encoding: xudp, uuid: %s, servername: %s, host: %s, path: /, reality-opts: {public-key: %s, short-id: %s}, client-fingerprint: %s, skip-cert-verify: true, udp: true}\n",
			n.Name, n.Server, n.Port, n.UUID, n.ServerName, n.ServerName, n.PublicKey, n.ShortID, n.ClientFingerprint))
	} else if n.Type == "hysteria2" {
		skipCert := "false"
		if n.SkipCertVerify { skipCert = "true" }
		sb.WriteString(fmt.Sprintf("  - {name: %s, type: hysteria2, server: %s, port: %s, password: %s, sni: %s, skip-cert-verify: %s}\n",
			n.Name, n.Server, n.Port, n.Password, n.ServerName, skipCert))
	} else if n.Type == "ss" {
		sb.WriteString(fmt.Sprintf("  - {name: %s, type: ss, server: %s, port: %s, cipher: %s, password: %s, udp: true}\n",
			n.Name, n.Server, n.Port, n.Cipher, n.Password))
	}
}

func writeAutoGroup(sb *strings.Builder, name, gType string, nodes []Node) {
	sb.WriteString(fmt.Sprintf("  - name: %s\n    type: %s\n", name, gType))
	if gType != "select" {
		sb.WriteString("    url: http://www.gstatic.com/generate_204\n    interval: 300\n    tolerance: 50\n")
	}
	sb.WriteString("    proxies:\n")
	for _, n := range nodes { sb.WriteString(fmt.Sprintf("      - %s\n", n.Name)) }
}

func writeProxyGroup(sb *strings.Builder, name, gType string) {
	sb.WriteString(fmt.Sprintf("  - name: %s\n    type: %s\n", name, gType))
	sb.WriteString("    proxies:\n      - 🚀 节点选择\n      - ♻️ 自动选择\n      - 🎯 全球直连\n")
}

func downloadRules() map[string]string {
	urls := []string{
		UrlLan, UrlBanAD, UrlBanProgramAD, UrlChinaDomain, UrlChinaIP, 
		UrlProxyLite, UrlApple, UrlMicrosoft, UrlGoogle, UrlTelegram, 
		UrlNetflix, UrlMedia, UrlSteamCN, UrlGames, UrlOneDrive,
	}
	res := make(map[string]string)
	var wg sync.WaitGroup
	var mu sync.Mutex
	client := http.Client{Timeout: 30 * time.Second}
	for _, u := range urls {
		wg.Add(1)
		go func(urlStr string) {
			defer wg.Done()
			resp, err := client.Get(urlStr)
			if err == nil {
				defer resp.Body.Close()
				b, _ := io.ReadAll(resp.Body)
				mu.Lock()
				res[urlStr] = string(b)
				mu.Unlock()
			}
		}(u)
	}
	wg.Wait()
	return res
}

func processRule(sb *strings.Builder, content, target, extra string, exclusionMap map[string]bool) {
	if content == "" { return }
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") { continue }
		if idx := strings.Index(line, "#"); idx > 0 { line = strings.TrimSpace(line[:idx]) }
		parts := strings.Split(line, ",")
		if len(parts) >= 2 {
			if exclusionMap[strings.TrimSpace(strings.ToLower(parts[1]))] { continue }
		}
		if strings.Contains(line, ",") {
			if len(parts) < 2 { continue }
			if extra != "" {
				sb.WriteString(fmt.Sprintf("  - %s,%s,%s,%s\n", parts[0], parts[1], target, extra))
			} else {
				sb.WriteString(fmt.Sprintf("  - %s,%s,%s\n", parts[0], parts[1], target))
			}
		} else {
			if strings.Contains(line, "/") {
				sb.WriteString(fmt.Sprintf("  - IP-CIDR,%s,%s,no-resolve\n", line, target))
			} else {
				sb.WriteString(fmt.Sprintf("  - DOMAIN-SUFFIX,%s,%s\n", line, target))
			}
		}
	}
}

func classifyNodes(nodes []Node) map[string][]string {
	groups := map[string][]string{ "HK": {}, "TW": {}, "JP": {}, "SG": {}, "US": {}, "Other": {} }
	regHK := regexp.MustCompile(`(?i)(HK|Hong|Kong|香港|🇭🇰)`)
	regTW := regexp.MustCompile(`(?i)(TW|Taiwan|台湾|🇹🇼)`)
	regJP := regexp.MustCompile(`(?i)(JP|Japan|日本|🇯🇵)`)
	regSG := regexp.MustCompile(`(?i)(SG|Singapore|新加坡|🦁|🇸🇬)`)
	regUS := regexp.MustCompile(`(?i)(US|America|States|美国|🇺🇸)`)
	for _, n := range nodes {
		if regHK.MatchString(n.Name) { groups["HK"] = append(groups["HK"], n.Name)
		} else if regTW.MatchString(n.Name) { groups["TW"] = append(groups["TW"], n.Name)
		} else if regJP.MatchString(n.Name) { groups["JP"] = append(groups["JP"], n.Name)
		} else if regSG.MatchString(n.Name) { groups["SG"] = append(groups["SG"], n.Name)
		} else if regUS.MatchString(n.Name) { groups["US"] = append(groups["US"], n.Name)
		} else { groups["Other"] = append(groups["Other"], n.Name) }
	}
	return groups
}

func getCountryGroupName(code string) string {
	switch code {
	case "HK": return "🇭🇰 香港节点"
	case "TW": return "🇹🇼 台湾节点"
	case "JP": return "🇯🇵 日本节点"
	case "SG": return "🇸🇬 新加坡节点"
	case "US": return "🇺🇸 美国节点"
	default: return "🏳️‍🌈 其他地区"
	}
}

func pause(scanner *bufio.Scanner) {
	fmt.Println("\n按回车键退出...")
	scanner.Scan()
}

// --- 链接解析器 ---

func decodeBase64(s string) (string, error) {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "-", "+")
	s = strings.ReplaceAll(s, "_", "/")
	padding := len(s) % 4
	if padding > 0 {
		s += strings.Repeat("=", 4-padding)
	}
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func parseSS(link string) (Node, error) {
	u, err := url.Parse(link)
	if err != nil { return Node{}, err }
	
	name := u.Fragment
	if name == "" { name = "ss" }
	name, _ = url.QueryUnescape(name)
	
	var method, password string
	userInfo := u.User.String()
	
	if userInfo != "" {
		// 尝试将 User 信息进行 Base64 解码 (标准的 ss:// 往往是 base64(method:password)@host:port)
		decoded, err := decodeBase64(userInfo)
		if err == nil && strings.Contains(decoded, ":") {
			parts := strings.SplitN(decoded, ":", 2)
			method = parts[0]
			password = parts[1]
		} else {
			// 不是 Base64，那就是明文的 method:password
			method = u.User.Username()
			p, ok := u.User.Password()
			if ok { password = p }
		}
	} else {
		// 有时候整个 host 部分全是 base64，比如 ss://BASE64(method:pass@host:port)#name
		decoded, err := decodeBase64(u.Host)
		if err == nil && strings.Contains(decoded, "@") {
			parts := strings.SplitN(decoded, "@", 2)
			cred := parts[0]
			serverInfo := parts[1]
			
			credParts := strings.SplitN(cred, ":", 2)
			if len(credParts) == 2 {
				method = credParts[0]
				password = credParts[1]
			}
			
			serverParts := strings.SplitN(serverInfo, ":", 2)
			if len(serverParts) == 2 {
				u.Host = serverParts[0] + ":" + serverParts[1]
			} else {
				u.Host = serverInfo
			}
		}
	}

	return Node{
		Type:     "ss",
		Name:     name,
		Server:   u.Hostname(),
		Port:     u.Port(),
		Cipher:   method,
		Password: password,
	}, nil
}

func parseVless(link string) (Node, error) {
	u, err := url.Parse(link)
	if err != nil { return Node{}, err }
	query := u.Query()
	name := u.Fragment
	if name == "" { name = "vless" }
	name, _ = url.QueryUnescape(name)
	return Node{
		Type: "vless",
		Name: name, Server: u.Hostname(), Port: u.Port(), UUID: u.User.Username(),
		ServerName: query.Get("sni"), PublicKey: query.Get("pbk"), ShortID: query.Get("sid"), ClientFingerprint: query.Get("fp"),
	}, nil
}

func parseHy2(link string) (Node, error) {
	u, err := url.Parse(link)
	if err != nil { return Node{}, err }
	query := u.Query()
	
	name := u.Fragment
	if name == "" { name = "hy2" }
	name, _ = url.QueryUnescape(name)
	
	password := u.User.Username() 
	if password == "" {
		p, ok := u.User.Password()
		if ok { password = p }
	}
	
	skipCert := false
	if query.Get("insecure") == "1" { skipCert = true }

	return Node{
		Type: "hysteria2",
		Name: name, Server: u.Hostname(), Port: u.Port(), Password: password,
		ServerName: query.Get("sni"), SkipCertVerify: skipCert,
	}, nil
}
