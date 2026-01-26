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
	AutoGroupType   string // 自动选择组类型
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
	// 防闪退
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("程序发生错误: %v\n按回车退出...", r)
			bufio.NewReader(os.Stdin).ReadBytes('\n')
		}
	}()

	outputFile := "config.yaml"
	var nodes []Node
	
	// 全程单例 Scanner
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("=============================================================================")
	fmt.Println("          VLESS 转 Clash (v8.1 规则交互优化版)")
	fmt.Println("=============================================================================")
	
	// --- 1. 读取 VLESS ---
	fmt.Println(">>> 步骤1: 请粘贴 VLESS 链接")
	fmt.Println("    (支持多行，粘贴完毕后输入 ok 并回车)")
	fmt.Println("-----------------------------------------------------------------------------")

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
		pause(scanner)
		return
	}

	// --- 2. 读取自定义规则 (已添加提示) ---
	customRules := readCustomRules(scanner)

	// --- 3. 选择模式 ---
	modeIndex := showMenu17(scanner)
	config := getModeConfig(modeIndex)
	
	fmt.Printf("\n🚀 正在生成 [%s] ...\n", config.Name)
	fmt.Println("⏳ 正在并发下载 ACL4SSR 规则库，请稍候...")

	// --- 4. 生成内容 ---
	content := generateYaml(nodes, config, customRules)

	// --- 5. 写入文件 ---
	err := os.WriteFile(outputFile, []byte(content), 0644)
	if err != nil {
		fmt.Printf("❌ 写入失败: %v\n", err)
	} else {
		fmt.Println("=============================================================================")
		fmt.Printf("✅ 成功！已生成文件: %s\n", outputFile)
		if customRules != "" {
			fmt.Println("   ★ 已成功插入你的自定义规则 (优先匹配)。")
		}
		fmt.Println("   包含了在线抓取的数千条规则，断网可用。")
		fmt.Println("=============================================================================")
	}
	
	fmt.Println("\n按回车键退出...")
	scanner.Scan() 
}

// 读取用户粘贴的自定义规则
func readCustomRules(scanner *bufio.Scanner) string {
	fmt.Println("\n>>> 步骤2: 请粘贴自定义规则 (放在所有规则最前面)")
	fmt.Println("    例如: - DOMAIN-SUFFIX,baidu.com,DIRECT")
	fmt.Println("    ⚠️  注意：粘贴完毕后，请按回车换行，输入 ok 并回车！") // 关键提示
	fmt.Println("    (如果没有规则，直接输入 ok 跳过)")
	fmt.Println("-----------------------------------------------------------------------------")

	var sb strings.Builder
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		if strings.ToLower(line) == "ok" || strings.ToLower(line) == "done" {
			break
		}
		if line == "" { continue }
		
		sb.WriteString("  " + line + "\n")
	}
	return sb.String()
}

// 17 个选项菜单
func showMenu17(scanner *bufio.Scanner) int {
	fmt.Println("\n>>> 步骤3: 请选择配置模式 (ACL4SSR 在线版复刻):")
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
		if err == nil && val >= 1 && val <= 17 { return val }
		fmt.Print("❌ 输入错误，默认使用 [6]: ")
	}
	return 6
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
	case 16: c.Name = "ACL4SSR_Online_Full_Netflix"; c.IsFull = true
	case 17: c.Name = "ACL4SSR_Online_Full_Google"; c.IsFull = true
	}
	if c.IsMini {
		c.TargetNetflix = "🚀 节点选择"
		c.TargetGoogle = "🚀 节点选择"
	}
	return c
}

// 核心生成逻辑
func generateYaml(nodes []Node, c ModeConfig, customRules string) string {
	var sb strings.Builder
	sb.WriteString("socks-port: 7891\nallow-lan: true\nmode: Rule\nlog-level: info\nexternal-controller: 127.0.0.1:9090\n")

	sb.WriteString("\nproxies:\n")
	for _, n := range nodes {
		sb.WriteString(fmt.Sprintf("  - {name: %s, server: %s, port: %s, type: vless, tls: true, packet-encoding: xudp, uuid: %s, servername: %s, host: %s, path: /, reality-opts: {public-key: %s, short-id: %s}, client-fingerprint: %s, skip-cert-verify: true, udp: true}\n",
			n.Name, n.Server, n.Port, n.UUID, n.ServerName, n.ServerName, n.PublicKey, n.ShortID, n.ClientFingerprint))
	}

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

	if !c.IsMini {
		common := "select"
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

	if !c.IsNoReject {
		sb.WriteString("  - name: 🛑 广告拦截\n    type: select\n    proxies:\n      - REJECT\n      - DIRECT\n")
	}
	sb.WriteString("  - name: 🎯 全球直连\n    type: select\n    proxies:\n      - DIRECT\n      - 🚀 节点选择\n")
	sb.WriteString("  - name: 🐟 漏网之鱼\n    type: select\n    proxies:\n      - 🚀 节点选择\n      - DIRECT\n")

	sb.WriteString("\nrules:\n")
	
	// ★★★ 写入用户粘贴的自定义规则 (最高优先级) ★★★
	if customRules != "" {
		sb.WriteString(customRules)
	}

	// 下载规则
	rules := downloadRules()

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
		processRule(&sb, rules[UrlMedia], "🌍 国外媒体", "")
		processRule(&sb, rules[UrlProxyLite], "🚀 节点选择", "")
	} else {
		processRule(&sb, rules[UrlProxyLite], "🚀 节点选择", "")
		processRule(&sb, rules[UrlGoogle], "🚀 节点选择", "")
		processRule(&sb, rules[UrlTelegram], "🚀 节点选择", "")
	}

	processRule(&sb, rules[UrlChinaDomain], "🎯 全球直连", "")
	processRule(&sb, rules[UrlChinaIP], "🎯 全球直连", "no-resolve")
	sb.WriteString("  - MATCH,🐟 漏网之鱼\n")

	return sb.String()
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

func processRule(sb *strings.Builder, content, target, extra string) {
	if content == "" { return }
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") { continue }
		if idx := strings.Index(line, "#"); idx > 0 { line = strings.TrimSpace(line[:idx]) }
		
		if strings.Contains(line, ",") {
			parts := strings.Split(line, ",")
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

func pause(scanner *bufio.Scanner) {
	fmt.Println("\n按回车键退出...")
	scanner.Scan()
}
