# 健康管理 Agent — 开发计划（Plan）

> 北极星：**结合你的体检异常指标，告诉你今天该吃什么。**
> 拆分原则：**竖切片**。每个小需求都从「前端 → 接口 → 后端 →（存储）」整条打通，能在浏览器点一下看到真实效果。不做「先写完所有后端再接前端」的横切，避免坑堆到最后。
> 约定：`[x]` 已完成 · `[ ]` 待办 · `[~]` 进行中。

---

## 技术基线（已定）
- **前端**：`front/`（Figma Make 导出，React 18 + TS + shadcn/ui + Tailwind v4，Vite 6）。dev 端口 `5173`，代理 `/api`、`/health` → Go `8091`。
- **后端**：Go 1.25，HTTP `8091`，chi 路由，统一响应 `{code,message,data,trace_id}`。
- **存储**：**MySQL 8.0（Docker 启动）**，见 `docker-compose.yml`（宿主机 `3308`，库 `health_db`）。驱动 `go-sql-driver/mysql`。~~不用 SQLite~~。
- **LLM**：DeepSeek（OpenAI 兼容），key 走 `.env`。

---

## Phase 0 · 骨架 ✅ 基本完成
- [x] 分层脚手架（`cmd/server` + `internal/{http,config,logger,store,model}`）
- [x] 配置加载（yaml + env，API key 只从 env 读）
- [x] 结构化日志（slog JSON + `Redact()`）
- [x] HTTP 骨架（chi + traceID/recover/access-log/body-limit 中间件）
- [x] 统一响应 + 错误码 + `404`/`405` JSON 处理器
- [x] `GET /health` 探活（含 DB ping）
- [x] 真优雅关闭 + 完整 server 超时
- [x] 领域数据结构 `model`
- [x] 前端 `front/` 跑通（Vite dev + 代理到后端）
- [x] `docker-compose.yml`（MySQL 8.0，宿主机 3308）+ `config.yaml` 改 MySQL DSN
- [ ] **存储切到 MySQL**：store 驱动 `sqlite → mysql`、schema 转 MySQL 方言（放在 S2 一起做，S1 用不到 DB）

---

## 里程碑总览（竖切片）

| 里程碑 | 小需求 | 验收（浏览器可见） | 练到 |
|---|---|---|---|
| **M1 裸对话** | S1 真聊天打通 | 输入任意话 → Go 调 DeepSeek → 显示真 AI 回复（无 DB、无规则） | Go handler、调外部 API、超时/降级 |
| **M2 记住你** | S2 档案存取（MySQL） | 档案页填的信息 → 存 MySQL → 刷新还在 | Go+MySQL CRUD、Docker 起库 |
| | S3 档案注入对话 | AI 回复结合你的档案（"你 BMI 24，建议…"） | prompt 拼装、上下文注入 |
| **M3 杀手锏** | S4 体检指标录入 | 贴体检文本 → LLM 抽结构化草稿 → 确认入库 | LLM 结构化抽取、草稿确认流 |
| | S5 指标驱动饮食 ⭐ | 「今天吃什么」结合**异常指标**给建议（血糖高→控糖） | 业务规则+prompt、**命根子** |
| **M4 闭环** | S6 膳食卡片接真数据 | 膳食推荐卡片来自后端而非写死 | 结构化返回、前端渲染 |
| | S7 打卡持久化 | 打卡/饮水勾选 → 存库 → 明天看连续天数 | 状态持久化、按天聚合 |
| | S8 数据总览接真值 | 步数/睡眠/热量卡片来自库 | 聚合查询 |
| **M5 进阶** | S9 多轮对话记忆 | AI 记得上文 | 会话历史、上下文窗口 |
| | S10 语音输入 | 麦克风 → STT → 对话 | 音频流、STT 接入 |

## 每个小需求内部固定四步
1. **定接口约定**：请求/响应 JSON、错误码。
2. **前端接线**：把对应 mock（`resolveIntent`/写死卡片）换成 `fetch`。
3. **后端 handler**（练裸编码内核）：路由 → 校验 → 业务 → 返回。
4. **验收**：浏览器点一下看真效果 + 看 Go 日志。

---

## ▶ 当前开工：M1 / S1 裸对话
- [ ] **S1-a 管道打通**：Go 加 `POST /api/v1/chat` 空 handler，先返回写死 `{"reply":"pong"}` → 前端 `send()` 改调它 → 验证前后端通
- [ ] **S1-b 接 DeepSeek**：handler 真调 DeepSeek（读 `.env` key、`context` 超时、错误降级）→ 返回真回复
- [ ] **S1-c 前端态**：把假的 `setTimeout` 打字动画接到真请求的 loading/错误态上

> S1 不碰数据库、不做规则、不做抽取。就是把「聊天」从假变真。

---

## Phase 1 · 录入闭环（细化待办，进入 M3 时再展开）
> 体检文本 → 带异常标记的结构化数据，脏值被拦。字典标准化 / 异常判定 / LLM 抽取 / 两步录入+幂等 / 数据展示。（原横切细目保留在 git 历史，进入 S4/S5 时按需搬回。）

## Phase 2 · 建议闭环 ⭐（进入 M3/S5 时展开）
> 血糖高+海鲜过敏用户，建议不含高糖/海鲜、有理由、命中 critical 提示就医。禁忌规则字典+三档校验 / 上下文组装 / 单次 DeepSeek + 重写兜底 / 建议落库版本快照。

## Phase 3 · 生产加固（后置，量大了再做）
- [ ] 账号鉴权（`user_id` 常量 → 真实用户）
- [ ] PHI 字段级加密 + 审计表
- [ ] LLM 熔断/降级（抖动→规则模板）/ 限流
- [ ] Redis 会话
- [ ] 夜间批量预生成：worker pool + errgroup 按 user_id 分片（同用户串行、异用户并行）
- [ ] OTel 可观测（LLM 耗时/失败率/token、endpoint P99）

## Phase 4 · 增强（对齐 PRD P1/P2）
- [ ] 营养数据集 → 热量精算
- [ ] RAG（权威膳食指南向量库降幻觉）
- [ ] 语音闭环 `stt`（STTProvider 接口 → 国内语音大模型）
- [ ] 打卡 / 千人千面 / 导入导出 / 锻炼计划 / 拍照 / 点外卖
