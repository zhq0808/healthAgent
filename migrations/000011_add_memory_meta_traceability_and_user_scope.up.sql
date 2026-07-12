-- 数据边界说明（不建表，只是约束本表的职责范围）：
-- agent_memory_meta 只存"对话记忆特征"（偏好/目标/生活习惯等自由文本画像），
-- 不存身高体重、检验指标、过敏、疾病、用药等结构化健康档案——
-- 这类数据字段固定、需要精确类型和单位，将来应建独立的结构化健康档案表，
-- 本次迁移不新增该表，只是明确"不往这张表里塞"。

-- entity_key 命名约定（应用层维护白名单，本表不用 DB 枚举强制，见迁移文件末尾说明）：
--   格式 "<主题>.<属性>"，如 diet.preference(饮食偏好/忌口)、goal.weight(目标体重)、
--   habit.exercise(运动习惯)。同一 (user_id, agent_id, category, entity_key) 只保留一条，
--   新陈述覆盖旧 memory_value/confidence/status/来源字段，不追加拼接冲突文本。

-- 1. 去重范围要从"同一会话"升级为"同一用户"：用户级特征应跨会话生效，
--    不能因为在新会话里又说了一遍同一句话就再插一条重复记录。
--    session_id 列本身不改名、不改含义、原 FK 不变：它继续代表"这条记忆最近一次
--    被确认/更新时所在的会话"，只是不再参与唯一性约束——原来它是"去重范围内的
--    会话标识"，现在降级为纯追溯字段，字面语义没变，只是不再承担去重职责。
ALTER TABLE agent_memory_meta DROP CONSTRAINT uk_mem_entity;
ALTER TABLE agent_memory_meta
    ADD CONSTRAINT uk_mem_entity_user UNIQUE (user_id, agent_id, category, entity_key);

COMMENT ON COLUMN agent_memory_meta.session_id IS '最近一次确认/更新该特征的来源会话ID（仅追溯用，不参与去重）';

-- 2. 补充可追溯字段：source_message_id 精确定位到触发本次写入/更新的那条 episodic 消息；
--    status/confidence/extractor_version 支撑"低风险自动确认、高风险人工/规则确认"的闭环。
ALTER TABLE agent_memory_meta
    ADD COLUMN source_message_id BIGINT,
    ADD COLUMN status            VARCHAR(20) NOT NULL DEFAULT 'pending',
    ADD COLUMN confidence        NUMERIC(3,2),
    ADD COLUMN extractor_version VARCHAR(32);

ALTER TABLE agent_memory_meta
    ADD CONSTRAINT fk_meta_source_message FOREIGN KEY (source_message_id)
        REFERENCES agent_memory_episodic(id) ON DELETE SET NULL,
    ADD CONSTRAINT ck_mem_status CHECK (status IN ('confirmed', 'pending', 'rejected')),
    ADD CONSTRAINT ck_mem_confidence CHECK (confidence IS NULL OR confidence BETWEEN 0 AND 1);

COMMENT ON COLUMN agent_memory_meta.source_message_id IS '产生/最近一次确认该特征的原始消息ID（关联 agent_memory_episodic.id）';
COMMENT ON COLUMN agent_memory_meta.status            IS '确认状态: confirmed=可信可注入上下文/pending=待确认，不注入/rejected=已否决，不注入';
COMMENT ON COLUMN agent_memory_meta.confidence         IS '抽取置信度 0.00-1.00，由 extractor 给出，人工确认的可为 NULL';
COMMENT ON COLUMN agent_memory_meta.extractor_version  IS '生成/最近一次更新该记录的抽取器版本号，用于回溯 Prompt 变更影响';
