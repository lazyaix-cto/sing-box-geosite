# sing-box-geosite

将上游各家代理规则（QuantumultX `.list`、Clash `.yaml` 等）转换为 **sing-box rule-set**
（源格式 `.json` + 二进制 `.srs`），并通过 GitHub Actions 每日自动更新。

与同类项目（如 `7ongOrz/sing-box-geosite`）的主要区别：

- **Go 实现，进程内编译 `.srs`** —— 直接调用 sing-box 库（`common/srs`），CI 不再需要
  下载安装 `sing-box` 二进制；版本由 `go.mod` 锁定、可复现。
- **以中间表示（IR）为枢纽的管线** —— `Parser → IR(model.RuleSet) → Transform → Compiler`，
  新增输入格式只需写一个解析器，新增输出版本只需改编译器。
- **不静默失败** —— 抓取/解析错误逐源上报，无法表示的规则类型（如 `IP-ASN`）计数并打印，
  绝不悄悄丢弃。
- **产物可校验** —— 每个 `.srs` 编译后立即用 `srs.Read` 回读校验，确保真实 sing-box 可加载。

## 状态：P3（功能完整）

管线：并发抓取 → 自动嗅探 → 解析 → **override 定制** → **优化**（去重/正则校验/
关键字收敛/后缀收敛/CIDR 聚合）→ 输出 `.json` + 多版本 `.srs` → 回读校验。

- 已支持格式：`quantumultx`（QuantumultX / Clash classical）、`clash`（rule-provider
  payload）、`singbox`（原生 rule-set 源格式，含 **logical** 逻辑规则）、`domainlist`
  （纯域名 / v2fly dlc 前缀）、`adguard`（AdGuard/uBlock DNS 过滤子集）、`hosts`
  （hosts / dnsmasq）；`format` 留空即按内容 + URL 后缀自动嗅探。
- **逻辑规则**：`and`/`or` 组合规则在 IR、解析、编译全链路保留（原版放弃的能力）。
- **多版本输出**（`-srs-versions`）：默认 `auto` 取内容所需最小格式版本（当前 v1，
  兼容 sing-box ≥1.8）；可指定 `1,2,3` 额外产出 `<Category>.v2.srs`/`.v3.srs`，
  低于最小版的请求会被跳过并提示。`MinVersion()` 保证版本声明真实。
- **优化（`-optimize`，默认开）**——均为行为保持的安全收敛：
  - 关键字收敛：含某 `domain_keyword` 子串的 domain/suffix 必然也命中该 keyword，可删；
  - 后缀收敛：被更短 `domain_suffix` 覆盖的 domain/子 suffix（标签对齐）可删；
  - CIDR 聚合：`netipx` 合并重叠/相邻前缀，非法项丢弃并计数；
  - 非法正则丢弃，避免单条坏 regex 拖垮整个分类。
  - 实测：OpenAI 34→26、Global 35002→26271、Microsoft 708→521、China CIDR 21→15。
- **定制**：`rules/overrides/<Category>.yaml` 的 `add` / `exclude` 增删条目
  （模板见 `rules/overrides/_example.yaml`）。
- 并发抓取（`-concurrency`，默认 8），单源失败仅记 `[FAIL]`，不影响其余。
- 无法表示的类型（`USER-AGENT`/`IP-ASN`/`HOST-WILDCARD` 等）计数并打印，绝不静默丢弃。
- **IP-ASN 展开**（`-resolve-asn`，默认关）：将 `IP-ASN` 规则经 ipverse/asn-ip 公共数据集
  （免密钥、GitHub 托管）展开为 CIDR 并入 `ip_cidr`，带缓存；数据源可替换为在线 BGP API
  或离线 GeoLite2-ASN。默认关以保持产物确定、不引第三方依赖；未解析的 ASN 计数不静默丢弃。
- 单元测试覆盖各解析器、嗅探、各优化 pass、override、逻辑规则编译、ASN 解析；CI
  （`go test` + 每日生成提交）见 `.github/workflows/build.yml`。

## 使用

```bash
# 生成 rules/sources.yaml 里定义的全部分类到 ./rule/
go run ./cmd/generator -config rules/sources.yaml -out rule

# 可选：指定 srs 二进制格式版本（1/2/3）
go run ./cmd/generator -srs-version 1
```

## 配置：`rules/sources.yaml`

```yaml
sources:
  - category: OpenAI          # 产物名：rule/OpenAI.json + rule/OpenAI.srs
    format: quantumultx       # 解析器；留空或 auto 走自动（P0 暂等价 quantumultx）
    urls:                     # 多个 URL 会合并去重
      - https://raw.githubusercontent.com/blackmatrix7/ios_rule_script/master/rule/QuantumultX/OpenAI/OpenAI.list
```

## 在 sing-box 中引用

```json
{
  "type": "remote",
  "tag": "OpenAI",
  "format": "binary",
  "url": "https://raw.githubusercontent.com/<you>/sing-box-geosite/main/rule/OpenAI.srs",
  "download_detour": "proxy"
}
```

## 目录结构

```
cmd/generator/      入口：读配置 → 抓取 → 管线 → 写盘
internal/
  config/           sources.yaml 加载
  source/           HTTP 抓取（错误不吞）
  parser/           格式解析器（接口 + 注册表）
  model/            中间表示 RuleSet
  compile/          json.go（源格式）+ srs.go（进程内编译 + 回读校验）
rules/sources.yaml  源定义
rule/               产物（committed）
```

## 致谢

规则数据来自 blackmatrix7/ios_rule_script 等上游；思路参考 7ongOrz/sing-box-geosite。
