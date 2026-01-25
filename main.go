package main

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strings"
)

// Node 结构体用于存储解析后的节点信息
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
	// 1. 读取 vless.txt
	inputFile := "vless.txt"
	outputFile := "config.yaml"

	file, err := os.Open(inputFile)
	if err != nil {
		fmt.Printf("无法打开文件 %s: %v\n请确保目录下存在 vless.txt 文件，并将链接粘贴进去。\n", inputFile, err)
		pause()
		return
	}
	defer file.Close()

	var nodes []Node
	scanner := bufio.NewScanner(file)

	fmt.Println("正在解析节点...")

	// 2. 逐行解析链接
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "vless://") {
			continue
		}

		node, err := parseVless(line)
		if err != nil {
			fmt.Printf("解析错误跳过: %s\n", err)
			continue
		}
		nodes = append(nodes, node)
	}

	if len(nodes) == 0 {
		fmt.Println("未找到有效的 VLESS 链接。")
		pause()
		return
	}

	// 3. 生成 YAML 内容
	yamlContent := generateYaml(nodes)

	// 4. 写入文件
	err = os.WriteFile(outputFile, []byte(yamlContent), 0644)
	if err != nil {
		fmt.Printf("写入文件失败: %v\n", err)
	} else {
		fmt.Printf("成功！已生成 %s，包含 %d 个节点。\n", outputFile, len(nodes))
	}
	pause()
}

// 解析 VLESS 链接逻辑
func parseVless(link string) (Node, error) {
	u, err := url.Parse(link)
	if err != nil {
		return Node{}, err
	}

	query := u.Query()
	
	// 处理 fragment (节点名称)
	name := u.Fragment
	if name == "" {
		name = "unknown-node"
	}
    // 解码名称中的特殊字符
    name, _ = url.QueryUnescape(name)

	return Node{
		Name:             name,
		Server:           u.Hostname(),
		Port:             u.Port(),
		UUID:             u.User.Username(),
		ServerName:       query.Get("sni"),
		PublicKey:        query.Get("pbk"),
		ShortID:          query.Get("sid"),
		ClientFingerprint: query.Get("fp"),
	}, nil
}

// 生成 YAML 字符串
func generateYaml(nodes []Node) string {
	var sb strings.Builder

	// 头部配置
	sb.WriteString("socks-port: 7891\n")
	sb.WriteString("allow-lan: true\n")
	sb.WriteString("mode: Rule\n")
	sb.WriteString("log-level: info\n")
	sb.WriteString("external-controller: 127.0.0.1:9090\n")
	
	// Proxies 部分
	sb.WriteString("proxies:\n")
	for _, n := range nodes {
		// 按照你要求的单行格式构建
		line := fmt.Sprintf("  - {name: %s, server: %s, port: %s, type: vless, tls: true, packet-encoding: xudp, uuid: %s, servername: %s, host: %s, path: /, reality-opts: {public-key: %s, short-id: %s}, client-fingerprint: %s, skip-cert-verify: true, udp: true}\n",
			n.Name, n.Server, n.Port, n.UUID, n.ServerName, n.ServerName, n.PublicKey, n.ShortID, n.ClientFingerprint)
		sb.WriteString(line)
	}

	// Proxy Groups 部分
	sb.WriteString("proxy-groups:\n")
	
	// 1. 节点选择
	sb.WriteString("  - name: 🚀 节点选择\n")
	sb.WriteString("    type: select\n")
	sb.WriteString("    proxies:\n")
	sb.WriteString("      - ♻️ 自动选择\n")
	for _, n := range nodes {
		sb.WriteString(fmt.Sprintf("      - %s\n", n.Name))
	}

	// 2. 自动选择
	sb.WriteString("  - name: ♻️ 自动选择\n")
	sb.WriteString("    type: url-test\n")
	sb.WriteString("    url: http://www.gstatic.com/generate_204\n")
	sb.WriteString("    interval: 300\n")
	sb.WriteString("    tolerance: 50\n")
	sb.WriteString("    proxies:\n")
	for _, n := range nodes {
		sb.WriteString(fmt.Sprintf("      - %s\n", n.Name))
	}

	// 3. 全球直连
	sb.WriteString("  - name: 🎯 全球直连\n")
	sb.WriteString("    type: select\n")
	sb.WriteString("    proxies:\n")
	sb.WriteString("      - DIRECT\n")
	sb.WriteString("      - 🚀 节点选择\n")
	sb.WriteString("      - ♻️ 自动选择\n")

	// 4. 漏网之鱼
	sb.WriteString("  - name: 🐟 漏网之鱼\n")
	sb.WriteString("    type: select\n")
	sb.WriteString("    proxies:\n")
	sb.WriteString("      - 🚀 节点选择\n")
	sb.WriteString("      - ♻️ 自动选择\n")
	sb.WriteString("      - DIRECT\n")

	return sb.String()
}

func pause() {
	fmt.Println("\n按回车键退出...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}