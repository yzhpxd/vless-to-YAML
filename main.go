package main

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strings"
)

// Node 结构体
type Node struct {
	Name             string
	Server           string
	Port             string
	UUID             string
	ServerName       string
	PublicKey        string
	ShortID          string
	ClientFingerprint string
}

func main() {
	outputFile := "config.yaml"
	var nodes []Node
	scanner := bufio.NewScanner(os.Stdin)

	// --- 1. 交互式提示 ---
	fmt.Println("==================================================")
	fmt.Println("  VLESS 转 Clash 工具 (粘贴模式)")
	fmt.Println("==================================================")
	fmt.Println("请直接在此处粘贴你的 vless:// 链接 (可以一次粘贴多行)。")
	fmt.Println("粘贴完成后，输入 ok 并按回车，即可生成配置。")
	fmt.Println("--------------------------------------------------")

	// --- 2. 读取用户输入 ---
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 如果用户输入 ok 或 done，则停止读取
		if strings.ToLower(line) == "ok" || strings.ToLower(line) == "done" {
			break
		}

		if line == "" {
			continue
		}

		// 解析链接
		if strings.HasPrefix(line, "vless://") {
			node, err := parseVless(line)
			if err != nil {
				fmt.Printf("[跳过] 解析错误: %v\n", err)
			} else {
				nodes = append(nodes, node)
				fmt.Printf("[已添加] %s\n", node.Name)
			}
		}
	}

	if len(nodes) == 0 {
		fmt.Println("\n❌ 未检测到有效的 VLESS 链接。")
		pause()
		return
	}

	// --- 3. 生成并写入 ---
	fmt.Printf("\n正在处理 %d 个节点...\n", len(nodes))
	yamlContent := generateYaml(nodes)

	err := os.WriteFile(outputFile, []byte(yamlContent), 0644)
	if err != nil {
		fmt.Printf("❌ 写入失败: %v\n", err)
	} else {
		fmt.Printf("✅ 成功！已生成文件: %s\n", outputFile)
	}
	
	pause()
}

// 解析 VLESS (保持不变)
func parseVless(link string) (Node, error) {
	u, err := url.Parse(link)
	if err != nil {
		return Node{}, err
	}
	query := u.Query()
	name := u.Fragment
	if name == "" { name = "unknown" }
	name, _ = url.QueryUnescape(name)

	return Node{
		Name:              name,
		Server:            u.Hostname(),
		Port:              u.Port(),
		UUID:              u.User.Username(),
		ServerName:        query.Get("sni"),
		PublicKey:         query.Get("pbk"),
		ShortID:           query.Get("sid"),
		ClientFingerprint: query.Get("fp"),
	}, nil
}

// 生成 YAML (保持不变)
func generateYaml(nodes []Node) string {
	var sb strings.Builder
	sb.WriteString("socks-port: 7891\nallow-lan: true\nmode: Rule\nlog-level: info\nexternal-controller: 127.0.0.1:9090\n")
	
	sb.WriteString("proxies:\n")
	for _, n := range nodes {
		line := fmt.Sprintf("  - {name: %s, server: %s, port: %s, type: vless, tls: true, packet-encoding: xudp, uuid: %s, servername: %s, host: %s, path: /, reality-opts: {public-key: %s, short-id: %s}, client-fingerprint: %s, skip-cert-verify: true, udp: true}\n",
			n.Name, n.Server, n.Port, n.UUID, n.ServerName, n.ServerName, n.PublicKey, n.ShortID, n.ClientFingerprint)
		sb.WriteString(line)
	}

	sb.WriteString("proxy-groups:\n")
	sb.WriteString("  - name: 🚀 节点选择\n    type: select\n    proxies:\n      - ♻️ 自动选择\n")
	for _, n := range nodes { sb.WriteString(fmt.Sprintf("      - %s\n", n.Name)) }

	sb.WriteString("  - name: ♻️ 自动选择\n    type: url-test\n    url: http://www.gstatic.com/generate_204\n    interval: 300\n    tolerance: 50\n    proxies:\n")
	for _, n := range nodes { sb.WriteString(fmt.Sprintf("      - %s\n", n.Name)) }

	sb.WriteString("  - name: 🎯 全球直连\n    type: select\n    proxies:\n      - DIRECT\n      - 🚀 节点选择\n      - ♻️ 自动选择\n")
	sb.WriteString("  - name: 🐟 漏网之鱼\n    type: select\n    proxies:\n      - 🚀 节点选择\n      - ♻️ 自动选择\n      - DIRECT\n")

	return sb.String()
}

func pause() {
	fmt.Println("\n按回车键退出...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}
