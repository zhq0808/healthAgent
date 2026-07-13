-- ============================================================================
-- Baseline 000003 · 智能体情景记忆流水表（对话全量 append-only）
-- v2 变化：
--   * 新增 message_id UUID 作为唯一业务消息身份（由 Go 后端生成 UUIDv7）。
--   * parent_id BIGINT → parent_message_id UUID，外键引用 message_id。
--   * seq 改为 BIGINT，由 Session 的 next_message_seq 分配。
--   * 删除数据库 trace_id 列（追踪只留在结构化日志/追踪系统）。
--   * 新增 UNIQUE(message_id, user_id)，供记忆来源复合外键校验同一用户。
--   id BIGINT 仅作数据库内部行主键，不进入 API / 跨表业务关联 / 幂等 / 日志串联。
-- ============================================================================
CREATE TABLE agent_memory_episodic (
    id                BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    message_id        UUID         NOT NULL,             -- 唯一业务消息身份（后端 UUIDv7）
    session_id        VARCHAR(64)  NOT NULL,
    user_id           VARCHAR(64)  NOT NULL,             -- 数据归属用户
    agent_id          VARCHAR(64)  NOT NULL,
    seq               BIGINT       NOT NULL,             -- Session 内递增顺序号，由 next_message_seq 分配
    parent_message_id UUID,                              -- 父消息 message_id，支持 regenerate 分支

    role              VARCHAR(64)  NOT NULL,
    msg_type          VARCHAR(64)  NOT NULL DEFAULT 'text',
    status            VARCHAR(64)  NOT NULL DEFAULT 'completed', -- 流式状态

    content           TEXT,                              -- tool_call 等可为空
    thought           TEXT,                              -- ReAct 内部思考
    meta_data         JSONB,                             -- 图片URL/工具入参/其它可变元数据

    prompt_tokens     INTEGER,                           -- 成本核算（仅 assistant 有值）
    completion_tokens INTEGER,
    client_message_id UUID,                              -- 一次用户发送动作的幂等键（前端生成）

    sync_status       SMALLINT     NOT NULL DEFAULT 0,   -- 0未同步 1已同步 2失败
    deleted_at        TIMESTAMPTZ,                       -- 软删除（NULL=未删）
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT uk_ame_message_id      UNIQUE (message_id),
    CONSTRAINT uk_ame_message_user    UNIQUE (message_id, user_id),  -- 供来源复合外键 (message_id,user_id) 引用
    CONSTRAINT uk_ame_session_seq     UNIQUE (session_id, seq),
    CONSTRAINT ck_ame_role     CHECK (role     IN ('user','assistant','system','tool')),
    CONSTRAINT ck_ame_msg_type CHECK (msg_type IN ('text','image','file','tool_call','tool_result')),
    CONSTRAINT ck_ame_status   CHECK (status   IN ('pending','streaming','completed','failed')),
    CONSTRAINT ck_ame_sync     CHECK (sync_status IN (0,1,2)),
    CONSTRAINT fk_ame_parent_message FOREIGN KEY (parent_message_id)
        REFERENCES agent_memory_episodic(message_id) ON DELETE SET NULL,
    CONSTRAINT fk_ame_session_user FOREIGN KEY (session_id, user_id)
        REFERENCES agent_memory_session(session_id, user_id) ON DELETE RESTRICT
);

-- 索引：拉会话历史 (WHERE session_id=? ORDER BY seq) 由 uk_ame_session_seq 直接覆盖。
CREATE INDEX idx_ame_user_agent ON agent_memory_episodic (user_id, agent_id);
CREATE INDEX idx_ame_parent     ON agent_memory_episodic (parent_message_id);
-- 部分索引：后台向量化任务只扫"待同步/失败"的行，索引更小更快。
CREATE INDEX idx_ame_sync_pending ON agent_memory_episodic (sync_status)
    WHERE sync_status <> 1;
-- 同一用户同一会话内，一次用户发送动作幂等：重试命中同一条 user 消息，不重复插入。
CREATE UNIQUE INDEX uk_ame_user_session_client_message
    ON agent_memory_episodic (user_id, session_id, client_message_id)
    WHERE role = 'user' AND client_message_id IS NOT NULL;

CREATE TRIGGER trg_ame_updated_at
    BEFORE UPDATE ON agent_memory_episodic
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE  agent_memory_episodic IS '智能体情景记忆流水表';
COMMENT ON COLUMN agent_memory_episodic.id                IS '数据库内部行主键，仅限本库维护/排查，不对外';
COMMENT ON COLUMN agent_memory_episodic.message_id        IS '唯一业务消息身份，后端生成的 UUIDv7，所有跨表业务关联/API/记忆来源均用它';
COMMENT ON COLUMN agent_memory_episodic.session_id        IS '会话ID';
COMMENT ON COLUMN agent_memory_episodic.user_id           IS '数据归属用户ID';
COMMENT ON COLUMN agent_memory_episodic.agent_id          IS '处理该消息的智能体ID';
COMMENT ON COLUMN agent_memory_episodic.seq               IS 'Session 内递增顺序号，由 agent_memory_session.next_message_seq 分配，允许空洞';
COMMENT ON COLUMN agent_memory_episodic.parent_message_id IS '父消息 message_id（UUID），关联本轮 user 消息或为 regenerate 分支保留关系';
COMMENT ON COLUMN agent_memory_episodic.client_message_id IS '一次用户发送动作的幂等键，前端生成；assistant 消息为空是正常情况';
