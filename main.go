package main

import (
	"bufio"
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
	Name, Server, Port, UUID, ServerName, PublicKey, ShortID, ClientFingerprint string
}

// 模式配置参数
type ModeConfig struct {
	Name            string
	IsMini          bool   // 是否精简版
	IsFull          bool   // 是否全分组
	IsNoReject      bool   // 是否无广告拦截
	UseAdblockPlus  bool   // 是否强力去广告
	AutoGroupType   string // 自动选择组类型: url-test, select, fallback, all(多模式)
	UseCountryGroup bool   // 是否启用多国分组
	TargetNetflix   string // 奈飞规则指向哪里
	TargetGoogle    string // 谷歌规则指向哪里
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
	outputFile := "config.yaml"
	var nodes []Node
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("=============================================================================")
	fmt.Println("          VLESS 转 Clash (ACL4SSR 17种模式完美复刻版 v1.1)")
	fmt.Println("=============================================================================")
	fmt.Println(">>> 步骤1: 请粘贴 VLESS 链接 (支持多行，输入 'ok' 完成):")

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.ToLower(line) == "ok" || strings.ToLower(line) == "done" {
			break
		}
		if line == "" { continue }
		if strings.HasPrefix(line, "vless://") {
			node, err := parseVless(line)
			if err != nil {
				fmt.Printf(" [跳过] %v\n", err)
			} else {
				nodes = append(nodes, node)
				fmt.Printf(" [已添加] %s\n", node.Name)
			}
		}
	}

	if len(nodes) == 0 {
		fmt.Println("❌ 未检测到节点，请重启。")
		pause()
		return
	}

	// 菜单
	modeIndex := showMenu17()

	// 配置配方
	config := getModeConfig(modeIndex)
	
	fmt.Printf("\n🚀 正在生成 [%s] ...\n", config.Name)
	fmt.Println("⏳ 正在并发下载规则库 (ChinaIP, AD, Netflix等)...")

	content := generateYaml(nodes, config)

	err := os.WriteFile(outputFile, []byte(content), 0644)
	if err != nil {
		fmt.Printf("❌ 写入失败: %v\n", err)
	} else {
		fmt.Println("=============================================================================")
		fmt.Printf("✅ 成功！已生成文件: %s\n", outputFile)
		fmt.Println("   包含所有规则和策略组，断网可用！")
		fmt.Println("=============================================================================")
	}
	pause()
}

// 17 个选项菜单
func showMenu17() int {
	fmt.Println("\n>>> 步骤2: 请选择配置模式 (与在线版完全一致):")
	fmt.Println("-----------------------------------------------------------------------------")
	fmt.Println(" [1]  ACL4SSR_Online 默认版")
	fmt.Println(" [2]  ACL4SSR_Online_AdblockPlus 更多去广告")
	fmt.Println(" [3]  ACL4SSR_Online_MultiCountry 多国分组")
	fmt.Println(" [4]  ACL4SSR_Online_NoAuto 无自动测速")
	fmt.Println(" [5]  ACL4SSR_Online_NoReject 无广告拦截")
	fmt.Println(" [6]  ACL4SSR_Online_Mini 精简版")
	fmt.Println(" [7]  ACL4SSR_Online_Mini_AdblockPlus 精简版+更多去广告")
	fmt.Println(" [8]  ACL4SSR_Online_Mini_NoAuto 精简版+无自动测速")
	fmt.Println(" [9]  ACL4SSR_Online_Mini_Fallback 精简版+故障转移")
	fmt.Println(" [10] ACL4SSR_Online_Mini_MultiMode 精简版+多模式(自动/故障/负载)")
	fmt.Println(" [11] ACL4SSR_Online_Mini_MultiCountry 精简版+多国分组")
	fmt.Println(" [12] ACL4SSR_Online_Full 全分组")
	fmt.Println(" [13] ACL4SSR_Online_Full_MultiMode 全分组+多模式")
	fmt.Println(" [14] ACL4SSR_Online_Full_NoAuto 全分组+无自动测速")
	fmt.Println(" [15] ACL4SSR_Online_Full_AdblockPlus 全分组+更多去广告")
	fmt.Println(" [16] ACL4SSR_Online_Full_Netflix 全分组+奈飞加强")
	fmt.Println(" [17] ACL4SSR_Online_Full_Google 全分组+谷歌细分")
	fmt.Println("-----------------------------------------------------------------------------")
	fmt.Print("👉 请输入数字 (1-17): ")

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		val, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
		if err == nil && val >= 1 && val <= 17 {
			return val
		}
		fmt.Print("❌ 输入错误，请输入 1-17: ")
	}
	return 1
}

// 获取模式配方
func getModeConfig(mode int) ModeConfig {
	c := ModeConfig{AutoGroupType: "url-test", TargetNetflix: "🎥 奈飞视频", TargetGoogle: "📢 谷歌服务"}
	
	switch mode {
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
	case 16: c.Name = "ACL4SSR_Online_Full_Netflix"; c.IsFull = true // 逻辑上 Full 已包含 Netflix
	case 17: c.Name = "ACL4SSR_Online_Full_Google"; c.IsFull = true // 逻辑上 Full 已包含 Google
	}

	// 修正精简版的目标组
	if c.IsMini {
		c.TargetNetflix = "🚀 节点选择"
		c.TargetGoogle = "🚀 节点选择"
	}
	return c
}

// 核心生成逻辑
func generateYaml(nodes []Node, c ModeConfig) string {
	var sb strings.Builder
	sb.WriteString("socks-port: 7891\nallow-lan: true\nmode: Rule\nlog-level: info\nexternal-controller: 127.0.0.1:9090\n")

	// 写入 Proxies
	sb.WriteString("\nproxies:\n")
	for _, n := range nodes {
		sb.WriteString(fmt.Sprintf("  - {name: %s, server: %s, port: %s, type: vless, tls: true, packet-encoding: xudp, uuid: %s, servername: %s, host: %s, path: /, reality-opts: {public-key: %s, short-id: %s}, client-fingerprint: %s, skip-cert-verify: true, udp: true}\n",
			n.Name, n.Server, n.Port, n.UUID, n.ServerName, n.ServerName, n.PublicKey, n.ShortID, n.ClientFingerprint))
	}

	// 准备分组列表
	countryGroups := map[string][]string{}
	if c.UseCountryGroup {
		countryGroups = classifyNodes(nodes)
	}

	// 写入 Proxy Groups
	sb.WriteString("\nproxy-groups:\n")

	// 1. 核心组: 节点选择
	sb.WriteString("  - name: 🚀 节点选择\n    type: select\n    proxies:\n")
	if c.AutoGroupType == "all" {
		sb.WriteString("      - ♻️ 自动选择\n      - 🔯 故障转移\n      - ⚖️ 负载均衡\n")
	} else {
		sb.WriteString("      - ♻️ 自动选择\n")
	}
	
	// 如果有多国分组，先加入国家组
	if c.UseCountryGroup {
		for _, name := range []string{"HK", "TW", "JP", "SG", "US", "Other"} { // 保持顺序
			if len(countryGroups[name]) > 0 {
				sb.WriteString(fmt.Sprintf("      - %s\n", getCountryGroupName(name)))
			}
		}
	}
	// 再加入所有节点
	for _, n := range nodes { sb.WriteString(fmt.Sprintf("      - %s\n", n.Name)) }

	// 2. 自动/故障/负载组
	if c.AutoGroupType == "all" {
		writeAutoGroup(&sb, "♻️ 自动选择", "url-test", nodes)
		writeAutoGroup(&sb, "🔯 故障转移", "fallback", nodes)
		writeAutoGroup(&sb, "⚖️ 负载均衡", "load-balance", nodes)
	} else {
		writeAutoGroup(&sb, "♻️ 自动选择", c.AutoGroupType, nodes)
	}

	// 3. 国家分组定义 (如果启用)
	if c.UseCountryGroup {
		for _, code := range []string{"HK", "TW", "JP", "SG", "US", "Other"} {
			if list, ok := countryGroups[code]; ok && len(list) > 0 {
				sb.WriteString(fmt.Sprintf("  - name: %s\n    type: url-test\n    url: http://www.gstatic.com/generate_204\n    interval: 300\n    tolerance: 50\n    proxies:\n", getCountryGroupName(code)))
				for _, nodeName := range list {
					sb.WriteString(fmt.Sprintf("      - %s\n", nodeName))
				}
			}
		}
	}

	// 4. 功能分组 (非 Mini)
	if !c.IsMini {
		common := "select" // 默认手动
		writeProxyGroup(&sb, "📲 电报消息", common)
		writeProxyGroup(&sb, "📹 油管视频", common)
		writeProxyGroup(&sb, "🎥 奈飞视频", common)
		writeProxyGroup(&sb, "🌍 国外媒体", common)
		writeProxyGroup(&sb, "Ⓜ️ 微软服务", common)
		writeProxyGroup(&sb, "📢 谷歌服务", common)
		writeProxyGroup(&sb, "🍎 苹果服务", common)
		
		if c.IsFull {
			writeProxyGroup(&sb, "🎮 游戏服务", common)
			writeProxyGroup(&sb, "☁️ 微软云盘", common)
			writeProxyGroup(&sb, "🚂 Steam", common)
		}
	}

	// 5. 底部通用
	if !c.IsNoReject {
		sb.WriteString("  - name: 🛑 广告拦截\n    type: select\n    proxies:\n      - REJECT\n      - DIRECT\n")
	}
	sb.WriteString("  - name: 🎯 全球直连\n    type: select\n    proxies:\n      - DIRECT\n      - 🚀 节点选择\n")
	sb.WriteString("  - name: 🐟 漏网之鱼\n    type: select\n    proxies:\n      - 🚀 节点选择\n      - DIRECT\n")

	// === 规则处理 ===
	sb.WriteString("\nrules:\n")
	rules := downloadRules() // 并发下载

	// 写入规则逻辑
	processRule(&sb, rules[UrlLan], "🎯 全球直连", "")
	
	if !c.IsNoReject {
		processRule(&sb, rules[UrlBanAD], "🛑 广告拦截", "")
		if c.UseAdblockPlus { processRule(&sb, rules[UrlBanProgramAD], "🛑 广告拦截", "") }
	}

	if !c.IsMini {
		processRule(&sb, rules[UrlMicrosoft], "Ⓜ️ 微软服务", "")
		processRule(&sb, rules[UrlApple], "🍎 苹果服务", "")
		processRule(&sb, rules[UrlGoogle], c.TargetGoogle, "")
		processRule(&sb, rules[UrlTelegram], "📲 电报消息", "")
		processRule(&sb, rules[UrlNetflix], c.TargetNetflix, "")
		
		if c.IsFull {
			processRule(&sb, rules[UrlOneDrive], "☁️ 微软云盘", "")
			processRule(&sb, rules[UrlSteamCN], "🚂 Steam", "")
			processRule(&sb, rules[UrlGames], "🎮 游戏服务", "")
		}
		
		processRule(&sb, rules[UrlMedia], "🌍 国外媒体", "") // 含YouTube
		processRule(&sb, rules[UrlProxyLite], "🚀 节点选择", "")
	} else {
		// Mini 版简化规则
		processRule(&sb, rules[UrlProxyLite], "🚀 节点选择", "")
		processRule(&sb, rules[UrlGoogle], "🚀 节点选择", "")
		processRule(&sb, rules[UrlTelegram], "🚀 节点选择", "")
	}

	processRule(&sb, rules[UrlChinaDomain], "🎯 全球直连", "")
	processRule(&sb, rules[UrlChinaIP], "🎯 全球直连", "no-resolve")
	sb.WriteString("  - MATCH,🐟 漏网之鱼\n")

	return sb.String()
}

// 辅助: 写自动组
func writeAutoGroup(sb *strings.Builder, name, gType string, nodes []Node) {
	sb.WriteString(fmt.Sprintf("  - name: %s\n    type: %s\n", name, gType))
	if gType != "select" {
		sb.WriteString("    url: http://www.gstatic.com/generate_204\n    interval: 300\n    tolerance: 50\n")
	}
	sb.WriteString("    proxies:\n")
	for _, n := range nodes { sb.WriteString(fmt.Sprintf("      - %s\n", n.Name)) }
}

// 辅助: 写功能组
func writeProxyGroup(sb *strings.Builder, name, gType string) {
	sb.WriteString(fmt.Sprintf("  - name: %s\n    type: %s\n", name, gType))
	sb.WriteString("    proxies:\n      - 🚀 节点选择\n      - ♻️ 自动选择\n      - 🎯 全球直连\n")
}

// 下载
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

// 规则清洗
func processRule(sb *strings.Builder, content, target, extra string) {
	if content == "" { return }
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") { continue }
		if idx := strings.Index(line, "#"); idx > 0 { line = strings.TrimSpace(line[:idx]) } // 去行尾注释
		
		if strings.Contains(line, ",") {
			parts := strings.Split(line, ",")
			if len(parts) < 2 { continue }
			if extra != "" {
				sb.WriteString(fmt.Sprintf("  - %s,%s,%s,%s\n", parts[0], parts[1], target, extra))
			} else {
				sb.WriteString(fmt.Sprintf("  - %s,%s,%s\n", parts[0], parts[1], target))
			}
		} else {
			// 纯 IP/域名 容错
			if strings.Contains(line, "/") {
				sb.WriteString(fmt.Sprintf("  - IP-CIDR,%s,%s,no-resolve\n", line, target))
			} else {
				sb.WriteString(fmt.Sprintf("  - DOMAIN-SUFFIX,%s,%s\n", line, target))
			}
		}
	}
}

// 节点分类逻辑 (实现多国分组)
func classifyNodes(nodes []Node) map[string][]string {
	groups := map[string][]string{
		"HK": {}, "TW": {}, "JP": {}, "SG": {}, "US": {}, "Other": {},
	}
	// 简单正则匹配
	regHK := regexp.MustCompile(`(?i)(HK|Hong|Kong|香港|🇭🇰)`)
	regTW := regexp.MustCompile(`(?i)(TW|Taiwan|台湾|🇹🇼)`)
	regJP := regexp.MustCompile(`(?i)(JP|Japan|日本|🇯🇵)`)
	regSG := regexp.MustCompile(`(?i)(SG|Singapore|新加坡|🦁|🇸🇬)`)
	regUS := regexp.MustCompile(`(?i)(US|America|States|美国|🇺🇸)`)

	for _, n := range nodes {
		if regHK.MatchString(n.Name) {
			groups["HK"] = append(groups["HK"], n.Name)
		} else if regTW.MatchString(n.Name) {
			groups["TW"] = append(groups["TW"], n.Name)
		} else if regJP.MatchString(n.Name) {
			groups["JP"] = append(groups["JP"], n.Name)
		} else if regSG.MatchString(n.Name) {
			groups["SG"] = append(groups["SG"], n.Name)
		} else if regUS.MatchString(n.Name) {
			groups["US"] = append(groups["US"], n.Name)
		} else {
			groups["Other"] = append(groups["Other"], n.Name)
		}
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

func parseVless(link string) (Node, error) {
	u, err := url.Parse(link)
	if err != nil { return Node{}, err }
	query := u.Query()
	name := u.Fragment
	if name == "" { name = "unknown" }
	name, _ = url.QueryUnescape(name)
	return Node{
		Name: name, Server: u.Hostname(), Port: u.Port(), UUID: u.User.Username(),
		ServerName: query.Get("sni"), PublicKey: query.Get("pbk"), ShortID: query.Get("sid"), ClientFingerprint: query.Get("fp"),
	}, nil
}

func pause() {
	fmt.Println("\n按回车键退出...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}
