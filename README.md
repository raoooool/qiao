# qiao

`qiao` is a Go command-line translation tool with a provider-oriented architecture. It ships with three translation providers: Codex CLI, Claude Code, and Tencent Cloud Machine Translation API.

## Install

Install the latest GitHub Release to `~/.local/bin`:

```bash
curl -fsSL https://raw.githubusercontent.com/raoooool/qiao/main/install.sh | bash
```

Install a specific version or override the target directory:

```bash
curl -fsSL https://raw.githubusercontent.com/raoooool/qiao/main/install.sh | VERSION=v0.1.0 INSTALL_DIR=/usr/local/bin bash
```

Install from source with Go:

```bash
go install github.com/raoooool/qiao/cmd/qiao@latest
```

Build and run locally:

```bash
go run ./cmd/qiao --help
```

Manual download:

Open `https://github.com/raoooool/qiao/releases/latest` and download the archive for your platform.

## Usage

Translate positional text:

```bash
qiao "How are you?"
```

Translate stdin:

```bash
printf 'How are you?' | qiao
```

Override languages or provider:

```bash
qiao -f en -t zh "How are you?"
qiao -p claude "How are you?"
```

Return structured output:

```bash
qiao --json "How are you?"
```

Show executed command and elapsed time:

```bash
qiao -v "How are you?"
```

List available providers:

```bash
qiao providers
```

Initialize provider configuration:

```bash
qiao init
```

### CLI Flags

| Flag         | Short | Description                            | Default                       |
| ------------ | ----- | -------------------------------------- | ----------------------------- |
| `--from`     | `-f`  | Source language                        | `auto`                        |
| `--to`       | `-t`  | Target language                        | `zh`                          |
| `--provider` | `-p`  | Translation provider                   | _(configured by `qiao init`)_ |
| `--json`     |       | Output structured JSON                 | `false`                       |
| `--verbose`  | `-v`  | Show executed command and elapsed time | `false`                       |

## Configuration

`qiao` reads configuration from:

```text
~/.config/qiao/config.yaml
```

### Managing Config via CLI

```bash
qiao config set <key> <value>    # Set a value
qiao config get <key>            # Get a value
qiao config list                 # List all values
qiao config delete <key>         # Delete a value
```

### Config Keys

#### Top-level keys

| Key                | Description                                            | Default          |
| ------------------ | ------------------------------------------------------ | ---------------- |
| `default_provider` | Translation provider used when `--provider` is omitted | Example: `codex` |
| `default_source`   | Default source language                                | `auto`           |
| `default_target`   | Default target language                                | `zh`             |

#### Provider-specific keys

Provider config uses the format `providers.<name>.<field>`.

**codex** (`providers.codex.*`)

| Key                      | Description                  | Default           |
| ------------------------ | ---------------------------- | ----------------- |
| `providers.codex.model`  | Model to use for translation | _(codex default)_ |
| `providers.codex.binary` | Path to the codex CLI binary | `codex`           |

**claude** (`providers.claude.*`)

| Key                       | Description                   | Default            |
| ------------------------- | ----------------------------- | ------------------ |
| `providers.claude.model`  | Model to use for translation  | _(claude default)_ |
| `providers.claude.binary` | Path to the claude CLI binary | `claude`           |

**tencent** (`providers.tencent.*`)

| Key                            | Description                 | Default                                    |
| ------------------------------ | --------------------------- | ------------------------------------------ |
| `providers.tencent.secret_id`  | Tencent Cloud API SecretId  | `$TENCENTCLOUD_SECRET_ID`                  |
| `providers.tencent.secret_key` | Tencent Cloud API SecretKey | `$TENCENTCLOUD_SECRET_KEY`                 |
| `providers.tencent.region`     | Tencent Cloud API region    | `ap-guangzhou` (or `$TENCENTCLOUD_REGION`) |

### Example config file

```yaml
default_provider: codex
default_source: auto
default_target: zh

providers:
  codex:
    model: o3
  claude:
    model: sonnet
  tencent:
    secret_id: AKIDxxxxxxxx
    secret_key: xxxxxxxx
    region: ap-guangzhou
```

CLI flags override config file values for provider, source language, and target language.

## Supported Providers

| Provider  | Description                                                                         | Requirements                         |
| --------- | ----------------------------------------------------------------------------------- | ------------------------------------ |
| `codex`   | Uses [Codex CLI](https://github.com/openai/codex) via `codex exec`                  | `codex` binary in PATH               |
| `claude`  | Uses [Claude Code](https://claude.ai/code) via `claude -p`                          | `claude` binary in PATH              |
| `tencent` | Uses [Tencent Cloud Machine Translation API](https://cloud.tencent.com/product/tmt) | API credentials (env vars or config) |

## Environment Variables

| Variable                  | Description                 | Used by          |
| ------------------------- | --------------------------- | ---------------- |
| `TENCENTCLOUD_SECRET_ID`  | Tencent Cloud API SecretId  | tencent provider |
| `TENCENTCLOUD_SECRET_KEY` | Tencent Cloud API SecretKey | tencent provider |
| `TENCENTCLOUD_REGION`     | Tencent Cloud API region    | tencent provider |

## Releasing

Push a version tag and GitHub Actions will publish the release artifacts automatically:

```bash
go test ./...
git tag v0.1.0
git push origin v0.1.0
```

The release workflow builds archives for macOS, Linux, and Windows, then uploads checksums and binaries to GitHub Releases.

---

# qiao

`qiao` 是一个使用 Go 编写的命令行翻译工具，采用面向 provider 的架构。当前内置了三种翻译 provider：Codex CLI、Claude Code 和腾讯云机器翻译 API。

## 安装

通过 GitHub Release 安装到 `~/.local/bin`：

```bash
curl -fsSL https://raw.githubusercontent.com/raoooool/qiao/main/install.sh | bash
```

安装指定版本，或覆盖安装目录：

```bash
curl -fsSL https://raw.githubusercontent.com/raoooool/qiao/main/install.sh | VERSION=v0.1.0 INSTALL_DIR=/usr/local/bin bash
```

使用 Go 从源码安装：

```bash
go install github.com/raoooool/qiao/cmd/qiao@latest
```

在本地构建并运行：

```bash
go run ./cmd/qiao --help
```

手动下载：

打开 `https://github.com/raoooool/qiao/releases/latest`，下载对应平台的压缩包。

## 用法

翻译位置参数中的文本：

```bash
qiao "How are you?"
```

翻译标准输入：

```bash
printf 'How are you?' | qiao
```

覆盖语言或 provider：

```bash
qiao -f en -t zh "How are you?"
qiao -p claude "How are you?"
```

返回结构化输出：

```bash
qiao --json "How are you?"
```

显示执行的命令和耗时：

```bash
qiao -v "How are you?"
```

列出可用的 provider：

```bash
qiao providers
```

初始化 provider 配置：

```bash
qiao init
```

### CLI 参数

| 参数         | 短参数 | 说明                 | 默认值                    |
| ------------ | ------ | -------------------- | ------------------------- |
| `--from`     | `-f`   | 源语言               | `auto`                    |
| `--to`       | `-t`   | 目标语言             | `zh`                      |
| `--provider` | `-p`   | 翻译 provider        | _（由 `qiao init` 配置）_ |
| `--json`     |        | 输出结构化 JSON      | `false`                   |
| `--verbose`  | `-v`   | 显示执行的命令和耗时 | `false`                   |

## 配置

`qiao` 从以下路径读取配置：

```text
~/.config/qiao/config.yaml
```

### 通过 CLI 管理配置

```bash
qiao config set <key> <value>    # 设置值
qiao config get <key>            # 获取值
qiao config list                 # 列出所有值
qiao config delete <key>         # 删除值
```

### 配置项

#### 顶层配置项

| Key                | 说明                                      | 默认值        |
| ------------------ | ----------------------------------------- | ------------- |
| `default_provider` | 未指定 `--provider` 时使用的翻译 provider | 示例：`codex` |
| `default_source`   | 默认源语言                                | `auto`        |
| `default_target`   | 默认目标语言                              | `zh`          |

#### Provider 专属配置项

Provider 配置使用 `providers.<name>.<field>` 格式。

**codex** (`providers.codex.*`)

| Key                      | 说明                 | 默认值             |
| ------------------------ | -------------------- | ------------------ |
| `providers.codex.model`  | 翻译时使用的模型     | _（codex 默认值）_ |
| `providers.codex.binary` | codex CLI 二进制路径 | `codex`            |

**claude** (`providers.claude.*`)

| Key                       | 说明                  | 默认值              |
| ------------------------- | --------------------- | ------------------- |
| `providers.claude.model`  | 翻译时使用的模型      | _（claude 默认值）_ |
| `providers.claude.binary` | claude CLI 二进制路径 | `claude`            |

**tencent** (`providers.tencent.*`)

| Key                            | 说明                 | 默认值                                      |
| ------------------------------ | -------------------- | ------------------------------------------- |
| `providers.tencent.secret_id`  | 腾讯云 API SecretId  | `$TENCENTCLOUD_SECRET_ID`                   |
| `providers.tencent.secret_key` | 腾讯云 API SecretKey | `$TENCENTCLOUD_SECRET_KEY`                  |
| `providers.tencent.region`     | 腾讯云 API 区域      | `ap-guangzhou`（或 `$TENCENTCLOUD_REGION`） |

### 配置文件示例

```yaml
default_provider: codex
default_source: auto
default_target: zh

providers:
  codex:
    model: o3
  claude:
    model: sonnet
  tencent:
    secret_id: AKIDxxxxxxxx
    secret_key: xxxxxxxx
    region: ap-guangzhou
```

CLI 参数会覆盖配置文件中的 provider、源语言和目标语言设置。

## 支持的 Providers

| Provider  | 说明                                                                | 依赖要求                        |
| --------- | ------------------------------------------------------------------- | ------------------------------- |
| `codex`   | 通过 `codex exec` 使用 [Codex CLI](https://github.com/openai/codex) | PATH 中需要有 `codex`           |
| `claude`  | 通过 `claude -p` 使用 [Claude Code](https://claude.ai/code)         | PATH 中需要有 `claude`          |
| `tencent` | 使用 [腾讯云机器翻译 API](https://cloud.tencent.com/product/tmt)    | 需要 API 凭证（环境变量或配置） |

## 环境变量

| 变量                      | 说明                 | 使用方           |
| ------------------------- | -------------------- | ---------------- |
| `TENCENTCLOUD_SECRET_ID`  | 腾讯云 API SecretId  | tencent provider |
| `TENCENTCLOUD_SECRET_KEY` | 腾讯云 API SecretKey | tencent provider |
| `TENCENTCLOUD_REGION`     | 腾讯云 API 区域      | tencent provider |

## 发布

推送版本标签后，GitHub Actions 会自动发布 Release 产物：

```bash
go test ./...
git tag v0.1.0
git push origin v0.1.0
```

发布流程会为 macOS、Linux 和 Windows 构建压缩包，并将校验文件与二进制上传到 GitHub Releases。
