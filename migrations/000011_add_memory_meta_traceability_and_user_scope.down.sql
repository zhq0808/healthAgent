-- 回滚顺序与 up 相反：先撤销去重范围变更，再撤销新增字段。
-- session_id 本身在 up 中未被改名/改约束归属，这里不需要做任何还原。
ALTER TABLE agent_memory_meta DROP CONSTRAINT uk_mem_entity_user;
ALTER TABLE agent_memory_meta
    ADD CONSTRAINT uk_mem_entity UNIQUE (session_id, category, entity_key);

COMMENT ON COLUMN agent_memory_meta.session_id IS '会话线程ID，与 episodic 一致';

ALTER TABLE agent_memory_meta
    DROP CONSTRAINT ck_mem_confidence,
    DROP CONSTRAINT ck_mem_status,
    DROP CONSTRAINT fk_meta_source_message;

ALTER TABLE agent_memory_meta
    DROP COLUMN extractor_version,
    DROP COLUMN confidence,
    DROP COLUMN status,
    DROP COLUMN source_message_id;