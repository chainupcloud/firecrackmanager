# 项目
> 该项目是基于 Github dtouzeau/firecrackmanager，进行二次开发。
主要目标
- 完善原项目网络相关的配置，尤其是port-forward
- 基于适配最新的firecrack，是否有新特性适配

## 语言与风格

- 代码英文、注释中文、文档中文、回答中文。
- **说人话**：用通用词和业内标准术语，不自造词；确需内部代号时，第一次出现先用一句话解释。
- **结论先行，过程从简**：先给答案或建议，再给必要依据；出处放句末括号，不展开引用。
- **请 owner 决策用固定格式**：一句话说清要决什么 + 2~3 个选项（每项一句利弊）+ 推荐哪个。不铺陈分析过程。


## CLAUDE.md 层级

根=全局规则（本文件）；`services/CLAUDE.md` / `apps/CLAUDE.md`=端内公用（前后端都可能多服务）；具体服务/应用目录下=该服务特有。每处以 CLAUDE.md 为主、建对应 `AGENTS.md` 软链（Codex 入口）。skills 同理双端：`.claude/skills/`（Claude）与 `.agents/skills/`（Codex）为同一协议，字节一致，改一处必须同步另一处。

## 编码原则（1–4 沿用 pm 约定；5–6 针对 agent 历史问题新增）

1. **Think Before Coding**：假设说出来；歧义呈现多解不默默选；更简单做法就说；困惑停下问。
2. **Simplicity First**：最少代码解决本次问题；不为单次使用造抽象；不写未要求的灵活性；不防御不可能场景。
3. **Surgical Changes**：每行改动可追溯到本次需求；不顺手重构；匹配现有风格；孤儿代码自清。
4. **Goal-Driven Execution**：任务转可验证目标（加验证→先写失败测试；修 bug→先写复现测试）；多步任务先给"步骤→验证点"。
5. **No-Fallback**：不建 fallback 机制掩盖故障，失败就显性失败（fail-fast）；确需 fallback（代码降级路径、模型替补皆算）必须先经 owner 审核，禁自作主张。
6. **长期方案优先**：一律做长期/最终方案，禁临时 hack；唯紧急止血例外，且止血后立即补最终方案卡（止血处注明关联 issue）。

琐碎改动自行判断松紧；非琐碎全条适用。


## worktree

- 每张卡一个独立工作区，固定在 `<repo>/.worktrees/<卡号>`（非卡工作区 `x-<名字>`），用 `bash script/wt.sh new <卡号>` 建；盘点用 `bash script/wt.sh list`。
- **只清理自己的**：自己的卡 merge 后立即 `git worktree remove <路径> && git branch -d <分支>`，两条都不带 force——被拒说明里面还有东西，看一眼再定。别人的工作区不碰不删（可能有未提交成果，删了拿不回来，本仓为此抢救过 3 次）；禁 `git clean -xdff`、禁 `git checkout --` 还原未提交文件。
- 操作细节与已踩的坑见 `docs/retros/issue-loop-protocol-history.md` §worktree 操作细节。



## 受护面（动之前先分两类）

- **要 owner 先拍板的（政策类，清单封闭）**：资金动作（主网写操作、真实资金划转、生产配置与限额）· 显式风险接受签字 · 增删一整个门禁或放宽其适用面 · `domain:*` 语义变更（映射表在 skill）· DB migration/schema。
- **其余受护面目录**（`verify/` 门内实现、`.github/`、`docs/design-docs/`、`.claude/`、`.agents/`、`testdata/golden/`、`docs/reference/`、`design/app.css`、`design/tokens.css`、`design/app.js`）：**可以直接做**，但 merge 前须过两道——`verify/gate-integrity.sh` 绿 + **异族评审（一轮封顶，owner 2026-08-06 定）**：开发按模型表派（多数落 GPT 系），评审由 Claude 编排会话直接做，作者与评审天然不同族、无需额外派发；Claude 系开发的卡，每批开工时问 owner 一次"评审派谁"；评审须给可复现的反例，不收"看起来没问题"；修复自证后即 merge，仅当修复改动了门禁判别逻辑才复审一次。
- 分类一句话：改「什么算完成 / 什么风险可接受」= 政策类；改「怎么实现已定政策」= 实现类。改制缘由与依据见 `docs/retros/issue-loop-protocol-history.md` §受护面改制缘由。


## 远端机器
@.env.hw
