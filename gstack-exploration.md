# gstack 项目探索报告

> 基于 `~/Downloads/gstack-main` 目录结构的分析

## 项目概述

gstack 是 **Garry Tan**（Y Combinator CEO）开源的 AI 工程工作流框架。它为 Claude Code、Codex、Kimi 等 AI 编码助手提供结构化的**技能（skills）**，将一个人的 AI 编码会话变成一个完整的虚拟工程团队。

---

## 1. 核心架构：持久浏览器守护进程

gstack 最核心的技术组件是一个**长驻的 Chromium 守护进程**（daemon）：

```
Claude Code (AI Agent)
    ↓ 工具调用（~100-200ms）
gstack CLI（编译后的二进制）
    ↓ HTTP POST localhost:PORT
Bun Server
    ↓ CDP 协议
Chromium（headless 或可视化）
```

### 关键特性

- **首次调用**：启动整个链路（约 3 秒）
- **后续调用**：只需 ~100-200ms，浏览器保持运行状态
- **状态持久**：Cookie、localStorage、登录会话、标签页在多次调用间保持
- **自动生命周期**：30 分钟空闲后自动关闭，无需手动管理
- **端口选择**：随机端口（10000-60000），支持多工作空间零冲突
- **版本自动重启**：编译新二进制后，下次调用自动杀掉旧进程启动新进程

---

## 2. 技能工作流（Skills Workflow）

gstack 将软件工程的不同角色封装为 **40+ 个 slash command 技能**，按工作阶段组织。

### 阶段一：规划评审（Plan Mode）

用户编写计划前，用这些技能进行多维度评审：

| 技能 | 作用 |
|------|------|
| `/office-hours` | 产品构思——6个强制性问题帮你重新框定需求 |
| `/plan-ceo-review` | CEO视角评审：找10星产品机会，4种范围模式（扩张/精选/保持/缩减） |
| `/plan-eng-review` | 工程经理视角：锁定架构、数据流、边界情况、测试覆盖 |
| `/plan-design-review` | 设计师视角：各设计维度0-10分评分 |
| `/plan-devex-review` | 开发者体验评审：上手时间、摩擦点、竞品对比 |
| `/autoplan` | 一键串行运行 CEO → 设计 → 工程 → DX 评审 |
| `/design-consultation` | 从零构建完整的设计系统 |

### 阶段二：实现与代码审查

编码过程中的专项技能：

| 技能 | 作用 |
|------|------|
| `/review` | 预合并 PR 审查：找 CI 通过但生产会出的 bug |
| `/codex` | 调用 OpenAI Codex 获取第二意见 |
| `/investigate` | 系统化根因调试——"无调查不修复" |
| `/design-review` | 对线上站点做视觉审计，原子提交修复 |
| `/design-shotgun` | 生成多个 AI 设计变体，对比迭代 |
| `/design-html` | 生成生产级 Pretext-native HTML/CSS |
| `/devex-review` | 实测开发者体验（对比计划时的预估） |
| `/qa` | 打开真实浏览器找 bug，修复并重新验证 |
| `/qa-only` | 同 `/qa` 但只报告不修改 |
| `/scrape` | 从网页提取数据，首次调用原型化，后续 ~200ms |
| `/skillify` | 将成功的 `/scrape` 流程固化成永久浏览器技能 |

### 阶段三：发布与部署

从 PR 到生产的完整流程：

| 技能 | 作用 |
|------|------|
| `/ship` | 运行测试、审查、推送、创建 PR（工作区感知的版本队列） |
| `/land-and-deploy` | 合并 PR、等待 CI 和部署、验证生产健康 |
| `/canary` | 部署后用浏览器守护进程做持续监控 |
| `/landing-report` | 只读的工作区发布队列仪表盘 |
| `/document-release` | 自动更新所有文档以匹配刚发布的代码 |
| `/setup-deploy` | 一次性部署配置检测（Fly.io, Render, Vercel 等） |
| `/gstack-upgrade` | 更新 gstack 到最新版本 |

### 阶段四：运维与记忆

长期项目维护：

| 技能 | 作用 |
|------|------|
| `/context-save` / `/context-restore` | 保存/恢复工作状态，跨会话甚至跨工作空间 |
| `/learn` | 管理 gstack 跨会话学到的知识 |
| `/retro` | 每周工程复盘，含个人贡献分解和 shipping streaks |
| `/health` | 代码质量仪表板（类型检查、测试、死代码等） |
| `/benchmark` | 性能回归检测（页面加载、Core Web Vitals） |
| `/benchmark-models` | 跨模型技能基准测试（Claude/GPT/Gemini 并排对比） |
| `/cso` | OWASP Top 10 + STRIDE 威胁模型安全审计 |
| `/setup-gbrain` | 设置 gbrain 实现跨机器会话记忆同步 |
| `/sync-gbrain` | 保持 gbrain 与仓库代码同步 |

### 阶段五：浏览器与代理集成

| 技能 | 作用 |
|------|------|
| `/browse` | 无头浏览器——真实 Chromium、真实点击、~100ms/命令 |
| `/open-gstack-browser` | 启动带侧边栏 + 反检测的可见浏览器窗口 |
| `/setup-browser-cookies` | 从真实浏览器导入 Cookie 用于认证测试 |
| `/pair-agent` | 配对远程 AI 代理（OpenClaw, Codex 等）共享浏览器 |

### 阶段六：安全与范围控制

| 技能 | 作用 |
|------|------|
| `/careful` | 危险命令前警告（rm -rf, DROP TABLE, force-push） |
| `/freeze` | 锁定编辑范围到一个目录（硬阻止，不只是警告） |
| `/guard` | 同时激活 careful + freeze |
| `/unfreeze` | 解除目录编辑限制 |
| `/make-pdf` | 将 Markdown 转为出版级 PDF |

---

## 3. 安装与集成流程

### 个人安装（30秒）

```bash
git clone --single-branch --depth 1 https://github.com/garrytan/gstack.git ~/.claude/skills/gstack
cd ~/.claude/skills/gstack && ./setup
```

### 团队模式（推荐）

在项目内运行：

```bash
(cd ~/.claude/skills/gstack && ./setup --team) && ~/.claude/skills/gstack/bin/gstack-team-init required
```

特性：
- 不在仓库中提交 vendor 文件
- 每小时自动检查更新（网络失败安全、完全静默）
- 队友打开 Claude Code 时自动获得 gstack

### 多平台支持

| 平台 | 支持方式 |
|------|---------|
| **Claude Code** | 主要目标平台，技能安装到 `~/.claude/skills/gstack` |
| **OpenAI Codex** | 安装到 `~/.codex/skills/gstack` |
| **Kimi** | 安装到 `~/.kimi/skills/gstack` |
| **Kiro / Factory / OpenCode** | 对应配置目录 |
| **OpenClaw / Hermes** | 通过 ACP 协议原生集成，提供方法论工件而非完整技能安装 |

---

## 4. 技术栈选择逻辑

| 技术 | 选择原因 |
|------|---------|
| **Bun** | 编译为单二进制文件（~58MB）、原生 SQLite、原生 TypeScript、内置 HTTP 服务器 `Bun.serve()` |
| **Playwright + Puppeteer** | 通过 Chrome DevTools Protocol (CDP) 控制 Chromium |
| **Markdown** | 所有技能都是 `SKILL.md` 文件，零依赖，易读易改 |
| **ngrok** | 用于 `/pair-agent` 的远程代理浏览器隧道 |
| **SQLite** | Bun 原生支持，直接读取 Chromium 的 SQLite cookie 数据库 |

---

## 5. 安全模型

### 本地监听

- HTTP 服务器只绑定 `127.0.0.1`，不对外网暴露

### 双监听器架构（v1.6.0.0）

为 `/pair-agent` 引入的安全设计：

| 监听器 | 绑定时机 | 功能 |
|--------|---------|------|
| **本地端口** | 始终绑定 | 完整功能：bootstrap、`/health`、cookie picker、inspector、完整命令集 |
| **隧道端口** | 按需绑定（`/tunnel/start`） | 受限白名单：`/connect`（配对）、`/command`（仅限浏览器驱动命令）、`/sidebar-chat` |

- **物理端口分离**：隧道调用者无法访问 `/health` 或 `/cookie-picker`，因为这些路径在隧道端口上不存在
- ngrok 只转发隧道端口

---

## 6. 项目结构要点

```
gstack-main/
├── AGENTS.md              # AI 代理工作流说明
├── ARCHITECTURE.md        # 架构设计文档
├── CLAUDE.md             # Claude Code 集成指南
├── SKILL.md              # 主技能文件（自动生成）
├── SKILL.md.tmpl         # 技能模板
├── CHANGELOG.md          # 变更日志
├── CONTRIBUTING.md       # 贡献指南
├── DESIGN.md             # 设计系统
├── ETHOS.md              # 项目理念
├── TODOS.md              # 待办事项
├── VERSION               # 版本号（1.31.1.0）
├── package.json          # Bun 项目配置
├── setup                 # 安装脚本（bash）
├── browse/               # 浏览器守护进程核心代码
│   ├── src/cli.ts        # CLI 入口
│   ├── src/server.ts     # Bun HTTP 服务器
│   └── dist/browse       # 编译后的二进制
├── design/               # 设计相关工具
├── make-pdf/             # PDF 生成工具
├── extension/            # Chrome 扩展（侧边栏、inspector）
├── bin/                  # 各种辅助脚本/二进制
├── test/                 # 测试套件
├── scripts/              # 构建/生成脚本
├── lib/                  # 共享库
├── hosts/                # 主机平台适配
├── agents/               # 代理配置
├── docs/                 # 文档
└── [skill-name]/         # 40+ 技能目录，每个包含 SKILL.md
```

---

## 7. 关键数据

- **版本**：1.31.1.0
- **许可证**：MIT
- **作者**：Garry Tan（Y Combinator President & CEO）
- **核心依赖**：Bun ≥1.0.0, Playwright, Puppeteer, ngrok, marked, diff
- **浏览器命令延迟**：首次 ~3s，后续 ~100-200ms
- **编译后二进制大小**：~58MB
- **技能数量**：40+
- **支持平台**：macOS, Linux, Windows（部分测试子集）

---

## 8. 文件存储策略：全局目录 vs 本地项目目录

gstack 采用**分层存储策略**：部分文件放本地项目目录，大部分放全局 `~/.gstack/`。

### 存储位置对比

| 数据类型 | 位置 | 示例 |
|---------|------|------|
| **浏览器守护进程状态** | 项目本地 `.gstack/browse.json` | PID、端口、token |
| **学习记录、检查点、计划** | 全局 `~/.gstack/projects/$SLUG/` | `learnings.jsonl`, `checkpoints/`, `ceo-plans/` |
| **会话追踪** | 全局 `~/.gstack/sessions/` | PPID 文件（2小时TTL） |
| **遥测/分析** | 全局 `~/.gstack/analytics/` | `skill-usage.jsonl` |
| **仓库模式缓存** | 全局 `~/.gstack/projects/$SLUG/repo-mode.json` | solo/collaborative |
| **安全日志** | 全局 `~/.gstack/security/` | `attempts.jsonl`, 设备盐值 |
| **ML 模型缓存** | 全局 `~/.gstack/models/` | `testsavant-small/`, `deberta-v3-injection/` |
| **项目 slug 缓存** | 全局 `~/.gstack/slug-cache/` | 路径→slug 映射 |
| **gbrain 队列** | 全局 `~/.gstack/.brain-queue.jsonl` | 跨机器同步数据 |

### 为什么大部分数据放全局目录？

#### 1. 跨会话/跨机器持久性
学习记录（`learnings.jsonl`）和检查点（`checkpoints/`）的核心价值是**让未来的会话能接续过去的工作**。如果放在本地 `.gstack/`，新克隆的仓库或 CI 环境会完全丢失这些积累。全局目录 + `$SLUG` 子目录实现了"无论在哪台机器、哪个工作目录，只要进入同一个项目，就能加载历史学习"。

> CHANGELOG 原文：`"/context-save" and "/context-restore" write session state to plaintext markdown in "~/.gstack/projects/$SLUG/checkpoints/", you can read and edit and move between machines.`

#### 2. 跨项目发现（Cross-project discovery）
gstack 支持搜索**其他项目**的学习记录来匹配当前问题。例如你在项目 A 修过某个 bug，项目 B 遇到相似模式时 gstack 可以提示。这要求所有学习记录在一个统一的全局命名空间下。

#### 3. 不污染项目仓库
gstack 的推荐安装模式是 **team mode**（`setup --team`），设计原则之一是"仓库中不 vendoring gstack 文件"。如果大量数据放在项目目录，就需要复杂的 `.gitignore` 管理，且队友可能意外提交这些文件。全局存储让项目目录保持干净。

#### 4. gbrain 同步
`~/.gstack/` 本身可以被初始化为 git 仓库，通过 gbrain 功能在多台机器间同步学习记录、检查点和配置。这是"个人知识库"架构——数据属于开发者，不属于某个 git 工作副本。

#### 5. 隐私与安全
敏感数据（安全日志、设备盐值、遥测）放在用户 home 目录下，权限可控（`umask 077`，`chmod 600`），不会被团队成员或 CI 意外读取。

### 为什么浏览器状态是例外？

`.gstack/browse.json` 放在**项目本地**，因为它与**当前工作目录强绑定**：
- 不同项目可能需要独立的浏览器实例（不同端口、不同 token）
- 守护进程的生命周期与项目工作流耦合（你在项目 A 做 QA，不应该影响项目 B 的浏览器）
- 旧版本曾用 `/tmp` 存放状态，后来改为项目本地（CHANGELOG: **"Project-local browse state. state file, logs, and all server state now live in `.gstack/` inside the project root"**）

### 总结

| 设计目标 | 全局 `~/.gstack/` | 本地 `.gstack/` |
|---------|------------------|----------------|
| 跨机器同步 | ✅ | ❌ |
| 跨项目检索 | ✅ | ❌ |
| 不污染 git | ✅ | ⚠️ 需 `.gitignore` |
| 按项目隔离 | 通过 `$SLUG` 子目录 | 天然隔离 |
| 进程级隔离 | ❌ | ✅（浏览器状态） |

gstack 的设计哲学是：**持久性数据属于开发者个人**（放全局），**运行时状态属于当前工作上下文**（放本地）。
