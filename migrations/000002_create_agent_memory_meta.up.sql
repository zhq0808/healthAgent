-- 长期记忆元数据表（提炼后的事实/画像，长期保留）
CREATE TABLE agent_memory_meta (
    memory_id        VARCHAR(64)  PRIMARY KEY,
    session_id       VARCHAR(64)  NOT NULL,           -- 会话线程，与 episodic 一致
    user_id          VARCHAR(64)  NOT NULL,           -- 数据归属用户
    agent_id         VARCHAR(64)  NOT NULL,
    category         VARCHAR(30)  NOT NULL,           -- user_preference/factual/bio
    entity_key       VARCHAR(100),                    -- 关联实体关键词，方便传统索引提取
    memory_value     TEXT         NOT NULL,           -- 提炼后的记忆内容，如"用户对花生过敏"

    -- 长期记忆管理三要素（重要性 + 频率 + 新近度，用于召回排序）
    importance       SMALLINT     NOT NULL DEFAULT 3, -- 重要性评分 1-10
    visit_count      INTEGER      NOT NULL DEFAULT 1, -- 被唤醒/点击频次
    last_accessed_at TIMESTAMPTZ  NOT NULL DEFAULT now(),

    deleted_at       TIMESTAMPTZ,                     -- 软删除（NULL=未删）
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT ck_mem_importance CHECK (importance BETWEEN 1 AND 10),
    CONSTRAINT ck_mem_category   CHECK (category IN ('user_preference','factual','bio')),
    -- 去重：同一归属下同类同实体只留一条，重复抽取时 UPSERT 而非插重。
    -- 注意：entity_key 为 NULL 的行不参与去重（PG 视多个 NULL 为不同值）。
    CONSTRAINT uk_mem_entity     UNIQUE (session_id, category, entity_key)
);

CREATE INDEX idx_mem_user_category ON agent_memory_meta (user_id, agent_id, category);
CREATE INDEX idx_mem_entity        ON agent_memory_meta (entity_key);

-- 复用 000001 定义的 set_updated_at() 触发器函数
CREATE TRIGGER trg_mem_updated_at
    BEFORE UPDATE ON agent_memory_meta
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE  agent_memory_meta IS '长期记忆元数据表（提炼后的事实/画像）';
COMMENT ON COLUMN agent_memory_meta.memory_id        IS '记忆唯一ID';
COMMENT ON COLUMN agent_memory_meta.session_id       IS '会话线程ID，与 episodic 一致';
COMMENT ON COLUMN agent_memory_meta.user_id          IS '数据归属用户ID';
COMMENT ON COLUMN agent_memory_meta.category         IS '分类: user_preference/factual/bio';
COMMENT ON COLUMN agent_memory_meta.entity_key       IS '关联实体关键词，方便传统索引提取';
COMMENT ON COLUMN agent_memory_meta.memory_value     IS '提炼后的记忆内容';
COMMENT ON COLUMN agent_memory_meta.importance       IS '重要性评分 1-10';
COMMENT ON COLUMN agent_memory_meta.visit_count      IS '被唤醒/点击频次';
COMMENT ON COLUMN agent_memory_meta.last_accessed_at IS '最后一次被唤醒时间';
COMMENT ON COLUMN agent_memory_meta.deleted_at       IS '软删除时间，NULL表示未删除';
