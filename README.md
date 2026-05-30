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

## 状态：P0（骨架已打通）

端到端跑通：抓取 → 解析（QuantumultX）→ 归一化 → 输出 `.json` + `.srs` → 回读校验。

- 已支持格式：`quantumultx`
- 路线图：P1 多格式 + 并发抓取 + 全量分类；P2 去重/收敛/CIDR 合并/override；
  P3 AdGuard/hosts/IP-ASN + 多版本（v1/v2/v3）输出。

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
