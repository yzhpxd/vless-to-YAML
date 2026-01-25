# VLESS to Clash Converter (Go)

这是一个简单的 Go 语言工具，用于将 VLESS 链接批量转换为 Clash 的 YAML 配置文件。

## 功能
- 读取同目录下的 `vless.txt`
- 自动解析 VLESS 链接参数 (UUID, IP, Port, Sni, PBK, SID 等)
- 生成包含自动测速和故障转移策略的 `config.yaml`

## 如何使用

1. 下载本工具或自行编译。
2. 在同目录下创建一个名为 `vless.txt` 的文件。
3. 将 VLESS 链接粘贴进去（一行一个）。
4. 运行程序，即可生成 `config.yaml`。

## 编译方法
```bash
go mod init converter
go build -ldflags="-s -w" -o converter.exe main.go