# net-merge

`net-merge` 是一个纯 Golang 编写的命令行工具，无需任何第三方依赖，用于处理 IPv4 CIDR 网段的合并与测试。支持从 URL、本地文件、命令行参数中获取网段信息，自动合并相邻 CIDR，生成规范格式的输出，并支持测试指定 IP 是否在网段范围内。

---

## 📦 功能概述

### `merge` 动作

将多个 CIDR 段信息合并输出为文件，可选生成普通文本格式或 nftables 格式。

**支持输入来源：**
- `--url(-u)`：指定一个或多个 URL（以逗号分隔），每个 URL 返回的文本中应包含一行一个 CIDR。
- `--file(-f)`：指定一个或多个本地文件路径（以逗号分隔），每个文件包含若干行 CIDR。
- `--network(-net)`：直接指定一个或多个 CIDR（以逗号分隔）作为输入。

**输出格式：**
- `--type(-t) txt`（默认）：每行一个 CIDR（如：`192.168.0.0/16`）
- `--type(-t) nft`：以 nftables 格式输出，如：
```nft
  define netlist = {
      10.0.0.0/8,
      192.168.0.0/16
  }

```

输出文件名：

使用 --out(-o) 指定输出文件名，默认为 merge.list

命令示例：
```bash
./cidr-tool merge \
  --url https://example.com/cidrs.txt \
  --file ./local-cidrs.txt \
  --network 172.16.0.0/12,10.0.0.0/8 \
  --out all-cidrs.nft \
  --type nft \
  --name whitelist

```

### `test` 动作
测试指定 IP 是否属于给定的合并结果文件中的某个 CIDR 范围。

参数说明：

`--in(-i)`：指定输入文件，默认为 merge.list

`--type(-t)`：输入文件类型，支持 txt（默认）或 nft

`--name(-n)`：如果是 nft 格式，则指定定义名称（默认 netlist）

命令示例：

```bash

./cidr-tool test --in all-cidrs.nft --type nft --name whitelist 192.168.1.100
```


## 🔧 构建方式
无需任何第三方依赖，使用 Go 编译：

```bash
go build -o cidr-tool main.go
```

## 🧠 功能特性
* 自动识别重复 CIDR 并去重

* 递归合并连续网段为更大范围

* 无需第三方库，跨平台可编译

* 兼容 nftables 格式输出，可直接用于 Linux 防火墙配置

## 📂 示例输出

普通格式（txt）

```
10.0.0.0/8
172.16.0.0/12
192.168.0.0/16
```
nft 格式

```nft
define netlist = {
    10.0.0.0/8,
    172.16.0.0/12,
    192.168.0.0/16
}
```



