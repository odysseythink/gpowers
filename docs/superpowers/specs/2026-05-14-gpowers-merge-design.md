# gpowers — 设计文档

> 把 **gstack**（Garry Tan 的 Claude Code 虚拟工程团队，23 个角色 + 8 个工具）和 **superpowers**（Jesse Vincent 的可组合软件开发方法论，14 个核心 skill + hooks 自动触发）组合成一个统一的跨平台发行版。

- 状态：草案
- 日期：2026-05-14
- 作者：George Ran Wei + AI brainstorm
- 上游：[gstack](https://github.com/garrytan/gstack)（v 当前 main）、[superpowers](https://github.com/obra/superpowers) v5.1.0
- 目标平台：Claude Code、Codex、Gemini、Cursor、OpenCode、Copilot CLI、Kimi CLI

## 锁定的约束（来自 brainstorming）

| 维度 | 决议 |
|---|---|
| **形态** | 新建统一发行版（不是并存安装、也不是单独基底扩展） |
| **范围** | 全期产品工厂：方法论 + 角色审查 + 工具 + 商业自动化 全集 |
| **触发哲学** | 双轨：core/ 用 hooks 自动触发；roles/ 和 tools/ 用显式 slash command |
| **跨平台** | 全平台第一公民（7 个平台） |
| **浏览器/QA** | 抽象出 9 个动词的「浏览器驱动」接口，按平台选不同实现 |
| **项目名** | **gpowers** |
| **命名空间** | 保留两个心智（方法论 vs 角色），通过模块目录 + skill 元数据明确区分 |

非目标：team mode（用户明确不需要）。

---

## §1 总体架构

### 模块布局

```
gpowers/                                  ← 单一 git repo
├── core/                                 ← 方法论层（来自 superpowers）
│   ├── skills/                           14 个核心 skill
│   └── hooks/                            SessionStart hook
│
├── roles/                                ← 角色化审查层（来自 gstack）
│   └── skills/                           20 个角色 skill
│
├── tools/                                ← 能力工具层（来自 gstack）
│   ├── skills/                           ~28 个工具 skill
│   ├── drivers/
│   │   ├── browser/                      9 动词抽象 + 多 driver 实现
│   │   ├── git/
│   │   └── shell/
│   └── bin/                              跨平台 CLI
│
├── business/                             ← 商业自动化层（可选安装，来自 gstack）
│   └── skills/                           ~20 个商业 skill
│
├── platforms/                            ← 7 平台注册清单
│   ├── claude-code/
│   ├── codex/
│   ├── gemini/
│   ├── cursor/
│   ├── opencode/
│   ├── copilot/
│   └── kimi/
│
├── install                               主安装脚本
├── upgrade                               主升级脚本（按模块拉上游）
├── tests/                                三层测试
├── docs/
├── CLAUDE.md / AGENTS.md / GEMINI.md
├── README.md
└── upstream-sources.json                 记录每个模块的 git 来源 + sha
```

### 一次典型使用的数据流

```
用户在 Claude Code 说 "我要做一个 X 功能"
  ↓
core/hooks/session-start 已加载 → using-gpowers 已注入
  ↓
core/skills/brainstorming 自动触发（按 hook 引导）
  ↓
用户和 agent 讨论清楚 → 写 spec → 调用 core/skills/writing-plans
  ↓
中途可选：用户调 /plan-ceo-review（roles/）做产品视角审查
       或调 /plan-eng-review 做工程审查
  ↓
计划好后 → core/skills/executing-plans + subagent-driven-development
  ↓
（开发期间自动）systematic-debugging on bug、verification-before-completion 在 PR 前
  ↓
完成 → 用户调 /pr-review（roles/）+ /cso（roles/，安全审查）
  ↓
通过 → 用户调 /ship（tools/）→ /land-and-deploy（tools/）→ /canary（tools/）
  ↓
（浏览器相关 skill 都通过 tools/drivers/browser/ 决定 MCP 还是 Playwright CLI）
```

### 不变量

1. 自动触发只发生在 `core/` 内的 14 个方法论 skill。`roles/` 和 `tools/` 全部是用户主动 slash command。
2. `business/` 模块的 skill 不会自动注入 CLAUDE.md，仅在 `install --with-business` 时注册到 platforms/。
3. Skill 内容只在 `~/.gpowers/` 一份真源；各平台通过软链或前缀注册暴露。
4. 运行时数据双层布局：全局共享数据在 `~/.gpowers/`（按 config/state/cache/data/analytics 分类）；项目相关数据在 `<repo>/.gpowers/`（plans/designs/evals/retros/learnings 等）。详见 §7。

---

## §2 core/ 模块（方法论层）

### 目录与文件

```
core/
├── skills/
│   ├── using-gpowers/                    入口 skill（改写自 using-superpowers）
│   │   └── SKILL.md
│   ├── brainstorming/                    来自 superpowers，几乎原样
│   ├── writing-plans/
│   ├── executing-plans/
│   ├── subagent-driven-development/
│   ├── test-driven-development/
│   ├── systematic-debugging/
│   ├── verification-before-completion/
│   ├── requesting-code-review/
│   ├── receiving-code-review/
│   ├── finishing-a-development-branch/
│   ├── dispatching-parallel-agents/
│   ├── using-git-worktrees/
│   └── writing-skills/
├── hooks/
│   ├── session-start                     唯一 hook，注入 using-gpowers
│   ├── run-hook.cmd                      Windows/Unix polyglot wrapper
│   └── hooks.json                        Claude Code 注册
└── upstream-source.json
```

### 核心改造点

1. **`using-superpowers` → `using-gpowers`**：重写入口 skill，要教 agent 四件事：
   - 方法论 skill（core/）默认遵守，自动应用
   - 角色 skill（roles/）按用户显式 slash command 触发，agent 在合适场景下"建议"用户用，但不自己调
   - 工具 skill（tools/）按需调用
   - 命名空间标签：agent 在回答中引用 skill 时加 `(core)` / `(roles)` / `(tools)` / `(business)` 标签

2. **`hooks/session-start`**：改自 superpowers 的同名 hook。改动点：注入 `using-gpowers` 而非 `using-superpowers`。其余（Cursor / Claude / Copilot JSON 格式分支、Windows polyglot wrapper、legacy 警告）原样保留。

3. **保留所有 14 个 core skill 的原始内容**，仅两类小改：
   - 内部引用从 `superpowers:writing-plans` 改成 `gpowers:writing-plans`
   - 每个 skill 的 SKILL.md frontmatter 加 `namespace: core` 和 `upstream: superpowers@v5.1.0`

4. **跨平台**：core/ 的 14 个 skill 本来就是 Markdown，本身跨平台。`platforms/<platform>/` 下放一份注册清单。Kimi 上需要额外生成 `gpowers-<name>/SKILL.md` 适配版（superpowers 上游没做 kimi 适配，这是新工作）。

### 双轨触发的具体兑现

- **自动轨**：session-start hook 注入 using-gpowers，agent 看到引导后自己用 `Skill` 工具调 brainstorming/TDD/debugging 等。**仅覆盖 core/**。
- **显式轨**：roles/ 和 tools/ 的 skill 通过 platforms/claude-code/commands/ 暴露为 slash command。用户主动调。
- **桥接**：using-gpowers 教 agent 在某些 trigger 词出现时**建议**用户调用 roles/tools skill（但不自己调）。例：用户说"准备上线"，agent 提示「建议先跑 /pr-review、/cso、/qa，再 /ship」。

### 不进 core/ 的"看似方法论"的 skill

- `investigate`（gstack）→ 放进 `roles/`，与 systematic-debugging 功能重叠但是角色化包装（带 Plan 文档输出）。保留 investigate 作为 roles 是为了不破坏 gstack 用户的肌肉记忆。
- `office-hours`（gstack）→ `roles/`，是 brainstorming 的产品视角变体。

---

## §3 roles/ 模块（20 个角色 skill）

### Skill 清单（按角色域分组）

**产品 / 策略（pre-coding）**
| Slash command | Skill 目录 | 上游 | 说明 |
|---|---|---|---|
| `/office-hours` | `office-hours` | gstack | YC 风格产品论坛，brainstorming 的产品变体 |
| `/plan-ceo-review` | `plan-ceo-review` | gstack | 产品视角审 plan（scope expansion/reduction） |
| `/autoplan` | `autoplan` | gstack | 一键串联 CEO + eng + design + devex 四轮审查 |

**工程 / 技术**
| Slash command | Skill 目录 | 上游 | 说明 |
|---|---|---|---|
| `/plan-eng-review` | `plan-eng-review` | gstack | 架构 + 数据流 + 边界 lock-in |
| `/plan-devex-review` | `plan-devex-review` | gstack | 面向 API/SDK/CLI 的 devex plan 审查 |
| `/devex-review` | `devex-review` | gstack | 实施后的 DX 走查 |
| `/investigate` | `investigate` | gstack | 角色化根因分析（Iron Law: no fixes without root cause） |
| `/codex` | `codex` | gstack | OpenAI Codex CLI 作为"200 IQ second opinion" |
| **`/pr-review`** ⚠️ 改名 | `pr-review` | gstack（原名 `review`） | 改名避免和 superpowers `requesting-code-review` 概念混淆 |

**设计**
| Slash command | Skill 目录 | 上游 | 说明 |
|---|---|---|---|
| `/plan-design-review` | `plan-design-review` | gstack | plan 期设计审查 |
| `/design-consultation` | `design-consultation` | gstack | 完整设计系统咨询（生成 DESIGN.md） |
| `/design-shotgun` | `design-shotgun` | gstack | 多变体 AI 设计 + 对比板 |
| `/design-html` | `design-html` | gstack | 把 mockup 落地为生产 HTML/CSS |
| `/design-review` | `design-review` | gstack | 实施后的视觉审查（依赖 tools/drivers/browser） |

**安全 / 合规**
| Slash command | Skill 目录 | 上游 | 说明 |
|---|---|---|---|
| `/cso` | `cso` | gstack | OWASP + STRIDE + 供应链 + LLM 安全审计 |

**回顾 / 记忆 / 协作**
| Slash command | Skill 目录 | 上游 | 说明 |
|---|---|---|---|
| `/retro` | `retro` | gstack | 周回顾，分析提交历史 |
| `/document-release` | `document-release` | gstack | 发布后同步文档 |
| `/learn` | `learn` | gstack | 管理项目"学到的东西" |
| `/pair-agent` | `pair-agent` | gstack | 跨 agent 共享浏览器（跨平台仅 Claude Code 完整工作） |
| `/plan-tune` | `plan-tune` | gstack | 调节"问题敏感度"和开发者画像 |

### 命名冲突最终清单

| 冲突 | 决议 |
|---|---|
| `review` | gstack `/review` → 改名 **`/pr-review`**；superpowers 的 `requesting-code-review` 和 `receiving-code-review` 保持原名（这两个是 skill 不是 slash command） |
| `investigate` vs `systematic-debugging` | 都保留。using-gpowers 引导中说明：bug 报告自动触发 systematic-debugging；用户要带文档输出的根因分析则用 `/investigate` |
| `plan-*` vs `writing-plans` | 不冲突。`writing-plans`（core）是写实现计划的方法论；`plan-*-review`（roles）是审查已有 plan 的角色 |
| `design-review` | 只放一份在 `roles/design-review`，tools/ 不重复 |
| `make-pdf` / `fix-the-roof` / `simplify` 等小工具 | 全部归 `tools/`，business/ 不重复 |

---

## §4 tools/ 模块（能力工具层）+ 浏览器驱动抽象

### 目录结构

```
tools/
├── skills/
│   ├── browse/                   通用浏览（drivers/browser 必需）
│   ├── qa/  qa-only/             QA 测试（drivers/browser 必需）
│   ├── canary/                   部署后监控（drivers/browser 必需）
│   ├── benchmark/                性能基线（drivers/browser 必需）
│   ├── benchmark-models/         AI 模型基线（独立）
│   ├── health/                   代码质量分（独立）
│   ├── ship/                     PR 创建 + 推送
│   ├── land-and-deploy/          合并 + 部署 + 验证
│   ├── landing-report/           队列仪表盘
│   ├── setup-deploy/             部署设置
│   ├── setup-browser-cookies/    导入 cookie（drivers/browser）
│   ├── setup-gbrain/  sync-gbrain/  gbrain 集成
│   ├── open-gstack-browser/      启动 gstack 浏览器
│   ├── context-save/  context-restore/   跨 session 上下文
│   ├── make-pdf/                 文档→PDF
│   ├── careful/  freeze/  guard/  unfreeze/    安全护栏
│   ├── fix-the-roof/             紧急修复模式
│   ├── simplify/                 简化代码审查
│   ├── fewer-permission-prompts/ 配置允许列表
│   ├── aidesigner/  aidesigner-frontend/   AIDesigner 集成
│   └── gpowers-upgrade/          自我升级（原 gstack-upgrade 改名）
├── drivers/
│   ├── browser/
│   │   ├── interface.md          抽象接口规范
│   │   ├── claude-in-chrome.md   Claude Code 实现
│   │   ├── playwright-cli.md     跨平台后备
│   │   └── select-driver.sh      运行时检测并 export GPOWERS_BROWSER_DRIVER
│   ├── git/
│   │   └── worktree.md           包装 superpowers 的 using-git-worktrees
│   └── shell/
│       └── platform-detect.sh    Bash vs PowerShell 切换
├── bin/                          跨平台 CLI（搬自 gstack/bin/）
│   ├── gpowers-update-check
│   ├── gpowers-health
│   ├── gpowers-browse
│   ├── gpowers-canary
│   ├── gpowers-benchmark
│   ├── gpowers-ship-helper
│   └── gpowers-path              ← 运行时目录解析 helper（见 §7）
└── upstream-source.json
```

### 浏览器驱动抽象（核心创新）

`drivers/browser/interface.md` 定义 9 个动词。Skill 流程只用这些动词，不直接调 MCP 或 CLI。

| 动词 | 含义 | 输入 | 输出 |
|---|---|---|---|
| `browser.open` | 打开 URL | url, viewport | tab_id |
| `browser.click` | 点击元素 | tab_id, selector | ok/err |
| `browser.type` | 输入文字 | tab_id, selector, text | ok/err |
| `browser.read` | 读取页面 | tab_id, mode (text/dom/console) | string |
| `browser.screenshot` | 截图 | tab_id, region | path |
| `browser.wait` | 等待条件 | tab_id, condition (selector/network-idle) | ok/timeout |
| `browser.eval` | 执行 JS | tab_id, code | json |
| `browser.cookies` | 读/写 cookie | tab_id, op, domain | json |
| `browser.close` | 关闭 tab | tab_id | ok |

**驱动选择逻辑**（`select-driver.sh`）：

```bash
if [ Claude Code 环境 (MCP claude-in-chrome 可用) ]; then
    export GPOWERS_BROWSER_DRIVER=claude-in-chrome
elif command -v playwright >/dev/null; then
    export GPOWERS_BROWSER_DRIVER=playwright-cli
else
    export GPOWERS_BROWSER_DRIVER=missing
    echo "Install: bun add -g @playwright/test  # or use Claude Code"
fi
```

**每个 driver 实现 9 个动词：**
- `claude-in-chrome.md`：动词 → MCP 工具映射表
- `playwright-cli.md`：动词 → playwright CLI 命令模板

**对 skill 作者的契约：**
- skill 文档里**不允许**直接出现 `mcp__claude-in-chrome__*` 或 `playwright` 字样
- 只能写 `browser.click`、`browser.read` 等动词
- skill 的 Preamble 自动 source `drivers/browser/select-driver.sh`

---

## §5 business/ 模块 + 安装/升级 + 平台注册

### business/

**定位**：商业/产品策略类 skill，**默认不安装**（`install` 时需要 `--with-business` flag）。全部 Markdown，跨平台无依赖。

```
business/
├── skills/
│   ├── money/                    主入口路由 skill
│   ├── money-discover/  money-product/
│   ├── money-content/  money-ads/  money-social/  money-seo/
│   ├── money-outreach/  money-ops/
│   ├── money-finance/  money-strategy/
│   ├── sell-the-outcome/
│   ├── pain-archaeology/  contrarian-timing/
│   ├── acquire-retain/  mvp-first/
│   ├── idea-generator/  idea-evaluator/
│   └── compounding-filter/  jtbd-mapping/
└── upstream-source.json
```

### 安装机制

主安装脚本 `install`：

```
gpowers install [选项]

选项：
  --with-business           安装 business/ 模块（默认不装）
  --core-only               只装 core/（极简模式）
  --no-tools                跳过 tools/
  --platforms=<list>        指定平台，逗号分隔
                            默认：检测当前可用平台并全部注册
                            可选：claude-code,codex,gemini,cursor,opencode,copilot,kimi
  --location=<path>         自定义安装目录，默认 ~/.gpowers/
  --link                    符号链接而非复制（开发模式）
  --uninstall               卸载
```

**安装产物布局：**

```
~/.gpowers/                              单一真源目录
├── core/  roles/  tools/  business/     内容仓库
├── platforms/                           注册清单生成器
├── bin/                                 CLI 入口（PATH 加进去）
├── manifest.json                        已安装模块清单
└── upstream-sources.json                各模块 git 来源 + sha

各平台外露（由 install 生成）：
~/.claude/plugins/gpowers/               Claude Code 软链 → ~/.gpowers
~/.codex/plugins/gpowers/                Codex 软链
~/.config/gemini/extensions/gpowers/     Gemini 软链
~/.cursor/plugins/gpowers/               Cursor 软链
~/.config/opencode/plugins/gpowers/      OpenCode 软链
~/.config/copilot-cli/plugins/gpowers/   Copilot 软链
~/.kimi/skills/                          Kimi（前缀注册）
   ├── gpowers/                          真源软链
   ├── gpowers-brainstorming/            14 个 core skill 适配
   ├── gpowers-plan-ceo-review/          roles skill 适配
   ├── gpowers-qa/                       tools skill 适配
   └── ...
```

**关键设计：单一真源 + 平台特定外露**。改一处生效全部。

**Kimi 适配生成**：
- `gen-kimi-adapters` 子命令读取 core/roles/tools/business 全部 skill
- 为每个生成 `~/.kimi/skills/gpowers-<name>/SKILL.md`
- 开头是 Preamble（设置 `GPOWERS_ROOT`）+ `<!-- SOURCE: $GPOWERS_ROOT/<module>/skills/<name>/SKILL.md -->` 引用
- Kimi 不支持 SessionStart hook，把 using-gpowers 的引导内容拼到每个 kimi 适配 skill 的 Preamble 头部

### 升级机制

```
gpowers upgrade [模块]

子命令：
  gpowers upgrade           升级所有模块到 upstream 最新
  gpowers upgrade core      只升 core/（拉 superpowers 上游）
  gpowers upgrade roles     只升 roles/（拉 gstack 上游）
  gpowers upgrade tools     只升 tools/
  gpowers upgrade business  只升 business/
  gpowers upgrade --check   只看有没有新版，不操作
```

**实现**：每个模块对应一个 git subtree：

```
~/.gpowers/  (git repo)
├── .git/
├── core/         subtree from github.com/obra/superpowers main
├── roles/        subtree from github.com/garrytan/gstack main 的子集
├── tools/        subtree from github.com/garrytan/gstack main 的子集
└── business/     subtree from github.com/garrytan/gstack main 的子集
```

**冲突处理**：subtree pull 出冲突时，脚本停下来打印 `git status` 并指引手动 resolve。

**自动检查**：tools/skills/gpowers-upgrade 在 session-start 时（被 core/hooks/session-start 调用）每小时一次后台检查上游版本，发现新版打印一行提示（不阻塞）。

### platforms/ 注册清单

每个平台目录：

```
platforms/<platform>/
├── plugin.json              平台原生插件清单
├── commands/                Slash command 文件
│   ├── pr-review.md         → 引用 roles/skills/pr-review
│   ├── plan-ceo-review.md
│   ├── ship.md
│   └── ...
├── skills.json              该平台暴露的 skill 列表 + 路径映射
└── hooks.json               该平台支持的 hook 配置
```

### 平台能力差异表

| 平台 | Slash command | 自动 hook | Skill 加载 | 命名空间 |
|---|---|---|---|---|
| Claude Code | ✅ /command | ✅ SessionStart | ✅ Skill 工具 | plugin-scoped |
| Codex | ✅ /command | ⚠️ 部分支持 | ✅ skill 工具 | plugin-scoped |
| Gemini | ✅ /command | ⚠️ 通过 GEMINI.md 注入 | ✅ activate_skill 工具 | extension |
| Cursor | ⚠️ 通过 .cursorrules | ⚠️ session 注入 | ⚠️ context inject | flat（前缀） |
| OpenCode | ✅ /command | ✅ | ✅ | plugin-scoped |
| Copilot CLI | ✅ /command | ⚠️ 通过 prompt | ✅ skill 工具 | plugin-scoped |
| Kimi | ⚠️ 用 skill 名调用 | ❌（嵌入 Preamble） | ✅ skill 名前缀 | gpowers- 前缀 |

### 跨平台 skill 暴露矩阵

**roles/**

| Skill 类 | Claude Code | Codex | Gemini | Cursor | OpenCode | Copilot | Kimi |
|---|---|---|---|---|---|---|---|
| 全 roles（除 design-review、pair-agent） | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| design-review | ✅ | ⚠️ playwright | ⚠️ playwright | ⚠️ playwright | ⚠️ playwright | ⚠️ playwright | ⚠️ playwright |
| pair-agent | ✅ | ⚠️ degraded | ⚠️ degraded | ⚠️ degraded | ⚠️ degraded | ⚠️ degraded | ⚠️ degraded |

**tools/**

| Skill | Claude Code | Codex | Gemini | Cursor | OpenCode | Copilot | Kimi |
|---|---|---|---|---|---|---|---|
| browse, qa, qa-only, canary, benchmark, setup-browser-cookies | ✅ | ⚠️ via playwright | ⚠️ | ⚠️ | ⚠️ | ⚠️ | ⚠️ |
| ship, land-and-deploy, landing-report, setup-deploy, health, benchmark-models, context-save/restore, careful, freeze, guard, unfreeze, make-pdf, fix-the-roof, simplify, fewer-permission-prompts, gpowers-upgrade | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| open-gstack-browser, aidesigner*, setup-gbrain, sync-gbrain | ✅ | ⚠️ degraded | ⚠️ degraded | ⚠️ degraded | ⚠️ degraded | ⚠️ degraded | ⚠️ degraded |

**business/**：所有 ✅（全部 7 个平台），无浏览器依赖。

⚠️ "degraded" 含义：skill 仍安装可用，但功能依赖 tools/drivers/browser 的非 Claude Code 路径，体验略弱。

---

## §6 测试 + 文档 + 卸载 + 迁移路径

### 测试策略

```
tests/
├── unit/                              节点级
│   ├── drivers/browser/
│   │   ├── claude-in-chrome.test.sh   9 个动词每个都跑一遍
│   │   └── playwright-cli.test.sh
│   ├── install.test.sh                各 flag 组合产物对不对
│   └── upgrade.test.sh                subtree pull mock
├── integration/                       skill 级
│   ├── core/brainstorming.test.md
│   ├── roles/pr-review.test.md
│   ├── tools/qa.test.md               真起一个 demo 站点跑 QA
│   └── ...
├── platform-smoke/                    平台级
│   ├── claude-code.test.sh            真实启动一次
│   ├── codex.test.sh
│   ├── gemini.test.sh
│   ├── cursor.test.sh
│   ├── opencode.test.sh
│   ├── copilot.test.sh
│   └── kimi.test.sh                   检查 gpowers-* 前缀 skill 都加载
└── fixtures/
    ├── demo-site/                     QA/canary/benchmark 用
    └── sample-repo/                   TDD/ship/pr-review 用
```

**自动化触发：**
- 每次 `gpowers upgrade <module>` 后自动跑 `unit/` + 该模块的 `integration/`
- 每次 release tag 时跑全部三层，必须 100% 通过才发版
- platform-smoke 跨 7 个平台并行，3 个失败视为发版阻塞

**自带方法论**：用 core/test-driven-development 来开发 gpowers 自己。

### 文档结构

```
docs/
├── README.md                          入口
├── INSTALL.md                         7 个平台各自的安装指引
├── ARCHITECTURE.md                    4 模块边界 + drivers 抽象
├── SKILLS.md                          全部 skill 索引（按模块分组）
├── COMMANDS.md                        全部 slash command 索引（按场景分组）
├── PLATFORMS.md                       7 平台对照表 + 已知限制
├── UPGRADING.md                       上游同步策略 + 冲突处理
├── CONTRIBUTING.md                    怎么加新 skill / 改 driver
├── DRIVERS.md                         浏览器驱动接口规范 + 怎么加新 driver
├── RUNTIME_LAYOUT.md                  全局/项目目录布局 + 环境变量 + 迁移路径（即 §7 落地）
└── superpowers/specs/                 设计 spec 历史归档
```

**两个特别文档：**

- `PLATFORMS.md`：四张大表（skill 可用性、slash command 原生支持、自动 hook 支持、浏览器 driver 选择）
- `DRIVERS.md`：9 个动词完整定义 + 写一个新 driver 的步骤 + skill 作者契约

### 卸载

```
gpowers uninstall [选项]

选项：
  --keep-data    保留 ~/.gpowers/sessions、~/.gpowers/learnings 等用户数据
  --dry-run      只打印会删除的内容
  --platform=<p> 只从指定平台卸载（不删 ~/.gpowers）
  --remove-data  连用户数据一并删
```

**执行：**
1. 删除各平台软链（`~/.claude/plugins/gpowers` 等、`~/.kimi/skills/gpowers-*`）
2. 从 CLAUDE.md / AGENTS.md / GEMINI.md 撤销 gpowers 注入段（用 markers 包围便于精确剪除）
3. 询问是否删 `~/.gpowers/` 真源（默认保留）
4. 不动用户数据除非传 `--remove-data`

### 迁移路径

`gpowers migrate` 命令：

| 来源 | 行为 |
|---|---|
| 已装 gstack | 备份 learnings/sessions/config → 卸载 gstack → 安装 gpowers → 数据搬到 ~/.gpowers/ |
| 已装 superpowers plugin | 提示用户手动 `/plugin uninstall superpowers` → gpowers 接管 core/ |
| 两者都装 | 先迁 gstack（数据多），再处理 superpowers plugin |
| 都没装 | 全新安装路径 |

**slash command 兼容**：
- `/review`（gstack 旧名）→ 自动 alias 到 `/pr-review`，6 个月过渡期保留
- 过渡期内每次调 `/review` 提示一次改名

### 版本与发布

- 语义化版本 `vMAJOR.MINOR.PATCH`
- `MAJOR` bump：接口变更（drivers 动词变、模块边界变）
- `MINOR` bump：新 skill、新平台支持
- `PATCH` bump：bug fix / 上游同步
- Release artifact：`gpowers-vX.Y.Z.tar.gz` 直接解压可用 + git tag

---

## §7 运行时目录与数据布局

**问题**：gstack 用扁平 `~/.gstack/` 杂烩 40+ 子目录；superpowers 用 XDG `~/.config/superpowers/`。gpowers 必须统一，并且**项目相关数据下沉到项目目录**。

### 两层布局

**全局层 `~/.gpowers/`**——跨项目共享、不变更频繁、单机一份。

```
~/.gpowers/
├── core/  roles/  tools/  business/   ← 安装内容（git managed）
├── platforms/                          ← 安装内容
├── bin/                                ← 安装内容
├── manifest.json                       ← 已安装模块
├── upstream-sources.json               ← 上游版本
├── .gitignore                          ← 排除下面的运行时目录
│
├── config/                             ← 用户偏好（跨项目）
│   ├── compact.toml                    ← 取代 ~/.config/gstack/compact.toml
│   ├── compact-rules/
│   ├── builder-profile                 ← 你的构建者画像
│   ├── developer-profile               ← 开发者画像
│   ├── gbrain-repo-policy
│   └── plan-tune.toml
│
├── state/                              ← 易变状态
│   ├── installation-id
│   ├── last-update-check
│   ├── update-snoozed
│   ├── just-upgraded-from
│   ├── security/
│   │   ├── attempts
│   │   ├── device-salt
│   │   └── feedback
│   └── worktrees/<project-slug>/<branch>/   ← 全局 worktree
│
├── cache/                              ← 可重建
│   ├── browser/
│   │   └── chromium-profile/
│   ├── models/                         ← AI 模型（大文件，跨项目共享）
│   ├── repos/                          ← clone 镜像
│   └── browsesafe-bench-smoke/
│
├── data/                               ← 全局产物（跨项目）
│   ├── browser-skills/                 ← 用户自定义 browser skill
│   ├── global-domain-skills/
│   ├── retros/global/                  ← 跨项目周回顾
│   ├── learnings/global/               ← 跨项目 learn 沉淀
│   ├── sessions/                       ← 项目外 session 上下文（如全局 office-hours）
│   ├── investigate-sessions/           ← 项目外的 root cause 调查
│   └── benchmarks/global/              ← 跨项目 benchmark 对比
│
├── analytics/                          ← 遥测（默认开，可关）
│   ├── skill-usage
│   ├── spec-review
│   ├── browse-telemetry
│   └── ...
│
├── logs/                               ← 全局错误日志
└── tmp/                                ← 短期临时
```

**项目层 `<repo>/.gpowers/`**——绑定项目，跟项目同生命周期。

```
<repo>/
├── .gpowers/                           ← 项目运行时数据
│   ├── plans/                          ← ceo-plans, eng-plans, design-plans, devex-plans, autoplans
│   │   ├── ceo/<slug>.md
│   │   ├── eng/<slug>.md
│   │   ├── design/<slug>.md
│   │   ├── devex/<slug>.md
│   │   └── autoplan/<slug>.md
│   ├── designs/                        ← /design-shotgun /design-html 产物
│   ├── evals/                          ← 评测结果
│   ├── sessions/                       ← 项目内 session 上下文快照
│   ├── investigate/                    ← /investigate 根因分析记录
│   ├── retros/                         ← 项目级 retro
│   ├── learnings/                      ← 项目学到的（PROJECT.learn.md 等）
│   ├── canary/                         ← canary 历史
│   ├── health/                         ← /health 分数历史
│   ├── benchmark/                      ← 性能基线
│   ├── ship-queue.json                 ← /landing-report 状态
│   ├── browser-skills/                 ← 项目专属 browser skill
│   ├── logs/                           ← 项目级日志
│   └── README.md                       ← 给团队的说明（自动生成）
│
└── docs/gpowers/specs/                 ← 已确立约定（不在 .gpowers/ 内，可被审阅 + commit）
    └── YYYY-MM-DD-*-design.md
```

### 项目目录探测

按优先级查找"项目根"：

1. `GPOWERS_PROJECT_DIR` 环境变量（显式指定）
2. 从 `cwd` 往上找 `.gpowers/`（已初始化项目）
3. 从 `cwd` 往上找 `.git`（任何 git 仓库即视为项目根）
4. 都没有 → fallback 到全局 `~/.gpowers/`

`gpowers init` 可显式创建 `<repo>/.gpowers/` 并写入 `.gitignore` 模板。

### 该 commit 还是该 ignore？

`<repo>/.gpowers/` 大部分**应该 commit**（团队共享决策记忆），少数运行时垃圾 ignore。`gpowers init` 写入的 `.gpowers/.gitignore` 模板：

```
# gpowers 项目运行时数据
# 默认大部分 commit（团队共享决策记忆）
# 排除项：
logs/
tmp/
sessions/*.pid
sessions/*.lock
*.local.*
.cache/
ship-queue.lock
```

具体规则：

| 子目录 | commit？ | 理由 |
|---|---|---|
| `plans/` | ✅ commit | CEO/eng/design 决策应是团队共识 |
| `designs/` | ✅ commit | 设计稿是产出 |
| `evals/` | ✅ commit | 评测结论值得追溯 |
| `retros/` | ✅ commit | 周回顾是历史 |
| `learnings/` | ✅ commit | 项目学到的应跨人传 |
| `investigate/` | ✅ commit | 根因记录是宝贵债务 |
| `sessions/` | ⚠️ 看情况 | 默认 commit 摘要，ignore 完整 pty 转录 |
| `canary/` `health/` `benchmark/` | ✅ commit | 历史趋势有用 |
| `ship-queue.json` | ✅ commit | 但 `.lock` 文件 ignore |
| `logs/` `tmp/` | ❌ ignore | 噪音 |
| `browser-skills/` | ✅ commit | 项目专属浏览器交互值得共享 |

### 环境变量覆盖

主路径全部可被环境变量覆盖：

```bash
GPOWERS_HOME=~/.gpowers                      # 全局根
GPOWERS_CONFIG=$GPOWERS_HOME/config          # 配置
GPOWERS_STATE=$GPOWERS_HOME/state            # 状态
GPOWERS_CACHE=$GPOWERS_HOME/cache            # 缓存
GPOWERS_DATA=$GPOWERS_HOME/data              # 全局产物
GPOWERS_ANALYTICS=$GPOWERS_HOME/analytics    # 遥测（设 NULL 关闭）
GPOWERS_PROJECT_DIR=<auto-detected>          # 项目根，默认自动探测
GPOWERS_PROJECT_DATA=$GPOWERS_PROJECT_DIR/.gpowers   # 项目数据
GPOWERS_TMP=$GPOWERS_HOME/tmp                # 临时
```

**XDG 互操作**：如果用户已经设了 `XDG_DATA_HOME` / `XDG_CACHE_HOME` 等，并且想跟随 XDG，可以一次性配：

```bash
export GPOWERS_DATA="${XDG_DATA_HOME:-$HOME/.local/share}/gpowers"
export GPOWERS_CACHE="${XDG_CACHE_HOME:-$HOME/.cache}/gpowers"
export GPOWERS_STATE="${XDG_STATE_HOME:-$HOME/.local/state}/gpowers"
export GPOWERS_CONFIG="${XDG_CONFIG_HOME:-$HOME/.config}/gpowers"
```

### 路径解析 helper

所有 skill 通过 `tools/bin/gpowers-path` 解析路径，**不允许**直接拼 `~/.gpowers/`：

```bash
$(gpowers-path config)         # → $GPOWERS_CONFIG
$(gpowers-path project plans)  # → $GPOWERS_PROJECT_DIR/.gpowers/plans
                               #   如未检测到项目 → fallback 到 $GPOWERS_DATA/plans
$(gpowers-path cache models)   # → $GPOWERS_CACHE/models
```

这保证未来路径迁移只改一处。

### 迁移路径细化（更新 §6 的迁移）

`gpowers migrate` 在原有逻辑外，针对运行时数据多一步：

| 旧位置 | 新位置 |
|---|---|
| `~/.gstack/config`, `~/.config/gstack/compact*` | `~/.gpowers/config/` |
| `~/.gstack/installation-id`, `last-update-check`, `update-snoozed`, `security/*` | `~/.gpowers/state/` |
| `~/.gstack/browse/`, `chromium-profile`, `models/`, `repos/`, `cache/` | `~/.gpowers/cache/` |
| `~/.gstack/builder-profile`, `developer-profile`, `gbrain-repo-policy` | `~/.gpowers/config/` |
| `~/.gstack/analytics/` | `~/.gpowers/analytics/` |
| `~/.gstack/projects/<slug>/ceo-plans/`, `designs/`, `evals/` | 对应 git repo 的 `<repo>/.gpowers/plans/ceo/` 等（依靠 `<slug>` 反查 repo 路径；找不到的留在 `~/.gpowers/data/legacy-projects/<slug>/`） |
| `~/.gstack/sessions/`, `retros/`, `learnings`, `investigate-sessions/`（如果项目内 cwd 历史可追） | 优先迁到对应项目 `.gpowers/`；否则 `~/.gpowers/data/<类>/global/` |
| `~/.config/superpowers/worktrees/<proj>/<branch>` | `~/.gpowers/state/worktrees/<proj>/<branch>` |

**迁移是 dry-run-first**：先打印映射表，让用户确认再执行。无法定位项目的产物归到 `legacy-projects/` 不丢。

### 卸载路径更新（更新 §6 的卸载）

```
gpowers uninstall [选项]

选项：
  --keep-data            保留 ~/.gpowers/data/、~/.gpowers/config/、各项目 <repo>/.gpowers/
  --remove-all-data      连项目数据也删（危险：会修改 git 工作区）
  --remove-global-data   删 ~/.gpowers/，但不动 <repo>/.gpowers/
```

默认行为：删安装内容（core/roles/tools/business + platforms + bin）+ state/ + cache/ + tmp/；**保留** config/, data/, analytics/, logs/, 以及所有项目里的 `<repo>/.gpowers/`。

---



这些问题不会阻塞 brainstorming → writing-plans 的转化，但实现期需要解答：

1. **Playwright CLI 版本**：select-driver 检测的是 `@playwright/test` 还是 `playwright`？两者命令略不同。
2. **Cursor 的 .cursorrules 注入策略**：Cursor 不原生支持 slash command。需要确认 roles/ 的 20 个 skill 在 Cursor 上是用引导段还是 cursor-rules.md 注入。
3. **Kimi 的并发**：如果一个 session 同时调多个 gpowers-* skill，Preamble 重复执行的开销 — 测一下。
4. **upstream-source.json schema**：是否区分"完全跟随上游" vs "本地有 patch"。
5. **business/ 模块的"商业自动化"在某些公司环境可能违规**：是否在安装时给出 disclaimer。

## 附录 B — 与 brainstorming 决议的对应

每个决议都对应到设计中的一个具体段落：

| 决议 | 落点 |
|---|---|
| 形态：新建统一发行版 | §1 总体架构（单一 git repo） |
| 范围：全集 | §2-§5（全 4 模块都收） |
| 触发哲学：双轨 | §2 双轨触发的具体兑现 |
| 跨平台：全平台第一公民 | §5 跨平台 skill 暴露矩阵 + platforms/ |
| 浏览器抽象 | §4 9 动词接口 + drivers/browser/ |
| 项目名：gpowers | 全文 |
| 命名空间分明 | §3 命名冲突清单 + frontmatter `namespace:` |
| +Kimi CLI | §5 Kimi 适配生成 |
| 运行时目录统一 + 项目相关进项目目录 | §7 全局 vs 项目两层布局 |
| GPOWERS_* 环境变量覆盖 | §7 环境变量覆盖 + 路径 helper |
