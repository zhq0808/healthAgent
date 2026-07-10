-- 智能体情景记忆流水表（对话全量 append-only）
CREATE TABLE agent_memory_episodic (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    session_id    VARCHAR(64)  NOT NULL,
    user_id       VARCHAR(64),                          -- 匿名期为 NULL，登录后回填
    agent_id      VARCHAR(64)  NOT NULL,
    seq           INTEGER      NOT NULL,
    parent_id     BIGINT,                               -- 消息树父节点，支持 regenerate 分支

    role          VARCHAR(64)  NOT NULL,
    msg_type      VARCHAR(64)  NOT NULL DEFAULT 'text',
    status        VARCHAR(64)  NOT NULL DEFAULT 'completed', -- 流式状态

    content       TEXT,                                 -- tool_call 等可为空
    thought       TEXT,                                 -- ReAct 内部思考
    meta_data     JSONB,                                -- 图片URL/工具入参/其它可变元数据

    prompt_tokens      INTEGER,                         -- 成本核算（仅 assistant 有值）
    completion_tokens  INTEGER,
    trace_id      VARCHAR(64),                          -- 关联请求日志

    sync_status   SMALLINT     NOT NULL DEFAULT 0,      -- 0未同步 1已同步 2失败
    deleted_at    TIMESTAMPTZ,                          -- 软删除（NULL=未删）
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT uk_session_seq  UNIQUE (session_id, seq),
    CONSTRAINT ck_ame_role     CHECK (role     IN ('user','assistant','system','tool')),
    CONSTRAINT ck_ame_msg_type CHECK (msg_type IN ('text','image','file','tool_call','tool_result')),
    CONSTRAINT ck_ame_status   CHECK (status   IN ('pending','streaming','completed','failed')),
    CONSTRAINT ck_ame_sync     CHECK (sync_status IN (0,1,2)),
    CONSTRAINT fk_ame_parent   FOREIGN KEY (parent_id)
        REFERENCES agent_memory_episodic(id) ON DELETE SET NULL
);

-- 索引：拉会话历史 (WHERE session_id=? ORDER BY seq) 由 uk_session_seq 直接覆盖。
CREATE INDEX idx_ame_user_agent ON agent_memory_episodic (user_id, agent_id);
CREATE INDEX idx_ame_parent     ON agent_memory_episodic (parent_id);
-- 部分索引：后台向量化任务只扫"待同步/失败"的行，索引更小更快。
CREATE INDEX idx_ame_sync_pending ON agent_memory_episodic (sync_status)
    WHERE sync_status <> 1;

-- updated_at 自动维护（PG 没有 MySQL 的 ON UPDATE，用触发器）
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_ame_updated_at
    BEFORE UPDATE ON agent_memory_episodic
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- 字段注释
COMMENT ON TABLE  agent_memory_episodic IS '智能体情景记忆流水表';
COMMENT ON COLUMN agent_memory_episodic.session_id  IS '会话ID';
COMMENT ON COLUMN agent_memory_episodic.user_id     IS '用户ID，匿名期为NULL，登录后回填';
COMMENT ON COLUMN agent_memory_episodic.agent_id    IS '处理该消息的智能体ID';
COMMENT ON COLUMN agent_memory_episodic.seq         IS '会话内消息序号，配合uk防并发串号';
COMMENT ON COLUMN agent_memory_episodic.parent_id   IS '消息树父节点，支持regenerate分支';
COMMENT ON COLUMN agent_memory_episodic.role        IS '角色: user/assistant/system/tool';
COMMENT ON COLUMN agent_memory_episodic.msg_type    IS '类型: text/image/file/tool_call/tool_result';
COMMENT ON COLUMN agent_memory_episodic.status      IS '流式状态: pending/streaming/completed/failed';
COMMENT ON COLUMN agent_memory_episodic.content     IS '消息主体内容，tool_call类可为空';
COMMENT ON COLUMN agent_memory_episodic.thought     IS '模型内部思考过程(ReAct Thought)';
COMMENT ON COLUMN agent_memory_episodic.meta_data   IS '结构化元数据(图片URL/工具入参等)';
COMMENT ON COLUMN agent_memory_episodic.sync_status IS '向量库同步状态: 0未同步/1已同步/2失败';
COMMENT ON COLUMN agent_memory_episodic.deleted_at  IS '软删除时间，NULL表示未删除';
