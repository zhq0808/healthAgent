-- ============================================================================
-- Baseline 000005 · 用户长期记忆（v2 开放原子记忆模型）
-- 四张表职责拆开：
--   agent_memory_meta            当前有效快照（聊天只读这张，不扫历史算当前状态）
--   agent_memory_history         追加式变更历史（ADD/UPDATE/DELETE，只追加不改旧）
--   agent_memory_history_source  每次变更引用的全部来源消息 + 已验证原文片段
--   agent_memory_extraction_state 每个 Session 一行的持久化抽取游标 + 租约
-- 彻底删除旧 category / entity_key / session_id 当前记忆模型。
-- ============================================================================

-- ---------------------------------------------------------------------------
-- 5.1 当前有效记忆快照
-- ---------------------------------------------------------------------------
CREATE TABLE agent_memory_meta (
    memory_id                UUID         PRIMARY KEY,          -- 后端 UUIDv7，业务主键
    user_id                  VARCHAR(64)  NOT NULL,             -- 归属用户，由可信上下文写入
    agent_id                 VARCHAR(64)  NOT NULL,             -- 当前固定 interview-agent
    memory_type              VARCHAR(20)  NOT NULL,             -- 可容错标签，允许 other
    memory_value             TEXT         NOT NULL,             -- 一条原子自然语言记忆
    confidence               NUMERIC(3,2),                      -- LLM 自评置信度，仅辅助
    latest_source_message_id UUID,                              -- 最近一次变更中 seq 最大的来源 user 消息
    extractor_model          VARCHAR(100),                      -- 抽取模型名
    extractor_version        VARCHAR(50),                       -- Prompt/Schema/规则版本
    version                  BIGINT       NOT NULL DEFAULT 1,   -- 乐观锁版本，初始1，每次变更递增

    deleted_at               TIMESTAMPTZ,                       -- 软删除时间
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT ck_meta_type       CHECK (memory_type IN ('preference','goal','habit','context','other')),
    CONSTRAINT ck_meta_value      CHECK (btrim(memory_value) <> ''),
    CONSTRAINT ck_meta_confidence CHECK (confidence IS NULL OR confidence BETWEEN 0 AND 1),
    CONSTRAINT ck_meta_version    CHECK (version > 0),
    -- 供 history 用 (memory_id,user_id,agent_id) 复合外键约束记忆归属。
    CONSTRAINT uk_meta_memory_owner UNIQUE (memory_id, user_id, agent_id),
    -- 来源消息必须与记忆属于同一用户（latest_source_message_id 可空，MATCH SIMPLE 下空值不校验）。
    CONSTRAINT fk_meta_latest_source FOREIGN KEY (latest_source_message_id, user_id)
        REFERENCES agent_memory_episodic(message_id, user_id) ON DELETE SET NULL
);

-- 召回主查询：某用户某 agent 的未删除记忆，按更新时间倒序。
CREATE INDEX idx_meta_user_agent_updated
    ON agent_memory_meta (user_id, agent_id, updated_at DESC)
    WHERE deleted_at IS NULL;

CREATE TRIGGER trg_meta_updated_at
    BEFORE UPDATE ON agent_memory_meta
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE  agent_memory_meta IS '用户长期记忆当前有效快照（开放原子记忆），聊天召回只读这张';
COMMENT ON COLUMN agent_memory_meta.memory_id                IS '记忆业务主键，后端 UUIDv7';
COMMENT ON COLUMN agent_memory_meta.user_id                  IS '归属用户ID，由可信上下文写入';
COMMENT ON COLUMN agent_memory_meta.agent_id                 IS '业务 Agent，当前固定 interview-agent';
COMMENT ON COLUMN agent_memory_meta.memory_type              IS '可容错标签: preference/goal/habit/context/other，不参与唯一约束/定位';
COMMENT ON COLUMN agent_memory_meta.memory_value             IS '一条原子自然语言记忆，一条只表达一个可独立更新的意思';
COMMENT ON COLUMN agent_memory_meta.confidence               IS 'LLM 自评置信度 0.00-1.00，仅作辅助门槛';
COMMENT ON COLUMN agent_memory_meta.latest_source_message_id IS '最近一次变更中 Session 内 seq 最大的来源 user 消息 message_id，便于快速追溯';
COMMENT ON COLUMN agent_memory_meta.extractor_model          IS '产生/更新该记忆的抽取模型名';
COMMENT ON COLUMN agent_memory_meta.extractor_version        IS '产生/更新该记忆的抽取 Prompt/Schema/规则版本';
COMMENT ON COLUMN agent_memory_meta.version                  IS '乐观锁版本，初始1，ADD/UPDATE/DELETE 每次递增';
COMMENT ON COLUMN agent_memory_meta.deleted_at               IS '软删除时间，NULL表示有效';

-- ---------------------------------------------------------------------------
-- 5.2 追加式变更历史（只追加，不修改旧历史）
-- ---------------------------------------------------------------------------
CREATE TABLE agent_memory_history (
    history_id        UUID         PRIMARY KEY,           -- 后端 UUIDv7，历史事件 ID
    memory_id         UUID         NOT NULL,              -- 对应当前记忆
    user_id           VARCHAR(64)  NOT NULL,              -- 归属用户
    agent_id          VARCHAR(64)  NOT NULL,              -- 业务 Agent
    action            VARCHAR(10)  NOT NULL,              -- ADD/UPDATE/DELETE
    from_version      BIGINT       NOT NULL,              -- 变更前版本，ADD 时为 0
    to_version        BIGINT       NOT NULL,              -- 变更后版本
    old_memory_value  TEXT,                               -- 变更前值，ADD 时为空
    new_memory_value  TEXT,                               -- 变更后值，DELETE 时为空
    memory_type       VARCHAR(20),                        -- 本次变更后的标签
    confidence        NUMERIC(3,2),                       -- 本次抽取置信度
    extractor_model   VARCHAR(100),                       -- 本次模型
    extractor_version VARCHAR(50),                        -- 本次抽取版本
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT ck_hist_action       CHECK (action IN ('ADD','UPDATE','DELETE')),
    CONSTRAINT ck_hist_from_version CHECK (from_version >= 0),
    CONSTRAINT ck_hist_to_version   CHECK (to_version > 0),
    CONSTRAINT ck_hist_confidence   CHECK (confidence IS NULL OR confidence BETWEEN 0 AND 1),
    -- 同一记忆同一版本只能产生一次历史；网络重试不会追加第二条同版本历史。
    CONSTRAINT uk_hist_memory_version UNIQUE (memory_id, to_version),
    -- 供 history_source 用 (history_id,user_id) 复合外键校验同一用户。
    CONSTRAINT uk_hist_history_user   UNIQUE (history_id, user_id),
    CONSTRAINT fk_hist_memory_owner FOREIGN KEY (memory_id, user_id, agent_id)
        REFERENCES agent_memory_meta(memory_id, user_id, agent_id) ON DELETE RESTRICT
);

-- 按记忆时间线回放：某记忆的全部历史按版本/时间排序。
CREATE INDEX idx_hist_memory ON agent_memory_history (memory_id, to_version);

COMMENT ON TABLE  agent_memory_history IS '记忆追加式变更历史，只追加不改旧，用于审计与恢复';
COMMENT ON COLUMN agent_memory_history.history_id        IS '历史事件ID，后端 UUIDv7';
COMMENT ON COLUMN agent_memory_history.action            IS '变更类型: ADD/UPDATE/DELETE';
COMMENT ON COLUMN agent_memory_history.from_version      IS '变更前版本，ADD 时为 0';
COMMENT ON COLUMN agent_memory_history.to_version        IS '变更后版本';
COMMENT ON COLUMN agent_memory_history.old_memory_value  IS '变更前记忆值，ADD 时为空';
COMMENT ON COLUMN agent_memory_history.new_memory_value  IS '变更后记忆值，DELETE 时为空';

-- ---------------------------------------------------------------------------
-- 5.3 每次变更的完整来源（多来源 + 已验证原文证据）
-- ---------------------------------------------------------------------------
CREATE TABLE agent_memory_history_source (
    history_id        UUID         NOT NULL,              -- 对应一次 memory history 变更
    source_order      SMALLINT     NOT NULL,              -- 本次来源顺序，从 1 开始
    user_id           VARCHAR(64)  NOT NULL,              -- 来源用户，用于数据库归属校验
    source_message_id UUID         NOT NULL,              -- 来源 user 消息的 message_id
    evidence_quote    TEXT         NOT NULL,              -- LLM 指出的原文片段，必须是来源消息原文子串

    CONSTRAINT pk_hist_source PRIMARY KEY (history_id, source_order),
    CONSTRAINT ck_hist_source_order CHECK (source_order >= 1),
    CONSTRAINT ck_hist_source_quote CHECK (btrim(evidence_quote) <> '' AND char_length(evidence_quote) <= 1000),
    -- 删除 history 时级联删除其来源关系。
    CONSTRAINT fk_hist_source_history FOREIGN KEY (history_id, user_id)
        REFERENCES agent_memory_history(history_id, user_id) ON DELETE CASCADE,
    -- 来源消息必须与本次变更属于同一用户。
    CONSTRAINT fk_hist_source_message FOREIGN KEY (source_message_id, user_id)
        REFERENCES agent_memory_episodic(message_id, user_id) ON DELETE RESTRICT
);

COMMENT ON TABLE  agent_memory_history_source IS '每次记忆变更引用的全部来源消息与已验证原文片段';
COMMENT ON COLUMN agent_memory_history_source.source_order      IS '本次来源顺序，从 1 开始';
COMMENT ON COLUMN agent_memory_history_source.source_message_id IS '来源 user 消息的 message_id（UUID）';
COMMENT ON COLUMN agent_memory_history_source.evidence_quote    IS 'LLM 指出的原文片段，后端必须校验它确实是来源消息原文的子串';

-- ---------------------------------------------------------------------------
-- 5.4 每个 Session 的持久化抽取游标 + 租约（每个 Session 一行）
-- ---------------------------------------------------------------------------
CREATE TABLE agent_memory_extraction_state (
    session_id                VARCHAR(64)  PRIMARY KEY,          -- 关联 Session
    user_id                   VARCHAR(64)  NOT NULL,             -- Session 归属用户
    last_extracted_result_seq BIGINT       NOT NULL DEFAULT 0,   -- 已处理 completed turn 的最大 assistant 结果 seq
    lease_token               UUID,                              -- 本次执行随机令牌；空表示当前无人处理
    lease_until               TIMESTAMPTZ,                       -- 本次执行权失效时间；崩溃后允许接管
    consecutive_failures      INTEGER      NOT NULL DEFAULT 0,   -- 连续失败次数；成功清零，用于退避/告警
    next_retry_at             TIMESTAMPTZ,                       -- 失败后最早重试时间
    last_error_code           VARCHAR(50),                       -- 最近一次非敏感错误分类，不存错误原文

    created_at                TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at                TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT ck_extract_seq      CHECK (last_extracted_result_seq >= 0),
    CONSTRAINT ck_extract_failures CHECK (consecutive_failures >= 0),
    -- lease_token 与 lease_until 必须同时为空或同时非空。
    CONSTRAINT ck_extract_lease_pair
        CHECK ((lease_token IS NULL AND lease_until IS NULL)
            OR (lease_token IS NOT NULL AND lease_until IS NOT NULL)),
    -- 复用 agent_memory_session 的 UNIQUE(session_id,user_id)，保证 Session 归属真实。
    CONSTRAINT fk_extract_session_user FOREIGN KEY (session_id, user_id)
        REFERENCES agent_memory_session(session_id, user_id) ON DELETE RESTRICT
);

-- 补扫积压：找"到重试时间、当前无未过期租约"的 Session。
CREATE INDEX idx_extract_retry ON agent_memory_extraction_state (next_retry_at);
CREATE INDEX idx_extract_lease ON agent_memory_extraction_state (lease_until)
    WHERE lease_until IS NOT NULL;

CREATE TRIGGER trg_extract_updated_at
    BEFORE UPDATE ON agent_memory_extraction_state
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE  agent_memory_extraction_state IS '每个 Session 的持久化抽取游标与租约，channel 只是提醒，这里才是可恢复真实进度';
COMMENT ON COLUMN agent_memory_extraction_state.last_extracted_result_seq IS '已处理 completed turn 的最大 assistant 结果 seq，初始 0';
COMMENT ON COLUMN agent_memory_extraction_state.lease_token          IS '本次执行随机令牌，空表示无人处理';
COMMENT ON COLUMN agent_memory_extraction_state.lease_until          IS '本次执行权失效时间，崩溃后允许其他 Worker 接管';
COMMENT ON COLUMN agent_memory_extraction_state.consecutive_failures IS '连续失败次数，成功清零，用于计算退避与告警';
COMMENT ON COLUMN agent_memory_extraction_state.next_retry_at        IS '失败后最早重试时间，防止故障时无间隔反复调用 LLM';
COMMENT ON COLUMN agent_memory_extraction_state.last_error_code      IS '最近一次非敏感错误分类，用于排错/指标，不存错误原文';
