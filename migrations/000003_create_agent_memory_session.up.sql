-- 会话元信息 + 滚动摘要表（一个会话一行，会话的"目录页"）
CREATE TABLE agent_memory_session (
    session_id      VARCHAR(64)  PRIMARY KEY,
    user_id         VARCHAR(64)  NOT NULL,            -- 数据归属用户
    title           VARCHAR(255),                    -- 会话标题（自动生成或用户自定义）
    status          VARCHAR(20)  NOT NULL DEFAULT 'active', -- active/archived/ended

    -- 滚动摘要（喂模型的上下文压缩，不替换 episodic 原文）
    summary         TEXT,                            -- 滑出窗口的老对话压缩成的一段话
    summarized_seq  INTEGER      NOT NULL DEFAULT 0, -- 水位线：episodic.seq 已压缩到第几条

    message_count   INTEGER      NOT NULL DEFAULT 0, -- 消息计数（冗余，方便列表/统计）
    meta_data       JSONB,                           -- 会话级元数据（模型配置等，可扩展）
    last_message_at TIMESTAMPTZ,                      -- 最后活跃时间（排序/TTL 判断）

    deleted_at      TIMESTAMPTZ,                      -- 软删除（NULL=未删）
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT ck_sess_status CHECK (status IN ('active','archived','ended'))
);

-- 列"某用户的会话，按最近活跃排序"：部分索引只覆盖未删的行。
CREATE INDEX idx_sess_user_updated ON agent_memory_session (user_id, updated_at DESC)
    WHERE deleted_at IS NULL;

-- 复用 000001 定义的 set_updated_at() 触发器函数
CREATE TRIGGER trg_sess_updated_at
    BEFORE UPDATE ON agent_memory_session
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE  agent_memory_session IS '会话元信息 + 滚动摘要表';
COMMENT ON COLUMN agent_memory_session.session_id      IS '会话ID';
COMMENT ON COLUMN agent_memory_session.user_id         IS '数据归属用户ID';
COMMENT ON COLUMN agent_memory_session.title           IS '会话标题（自动生成或用户自定义）';
COMMENT ON COLUMN agent_memory_session.status          IS '状态: active/archived/ended';
COMMENT ON COLUMN agent_memory_session.summary         IS '滚动摘要（滑出窗口的老对话压缩）';
COMMENT ON COLUMN agent_memory_session.summarized_seq  IS '摘要水位线：episodic.seq 已压缩到第几条';
COMMENT ON COLUMN agent_memory_session.message_count   IS '消息计数（冗余，方便列表/统计）';
COMMENT ON COLUMN agent_memory_session.meta_data       IS '会话级元数据（模型配置等，可扩展）';
COMMENT ON COLUMN agent_memory_session.last_message_at IS '最后活跃时间（排序/TTL判断）';
COMMENT ON COLUMN agent_memory_session.deleted_at      IS '软删除时间，NULL表示未删除';
