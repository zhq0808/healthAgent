ALTER TABLE agent_memory_meta
    DROP CONSTRAINT IF EXISTS fk_meta_session_user;

ALTER TABLE agent_memory_episodic
    DROP CONSTRAINT IF EXISTS fk_episodic_session_user;

ALTER TABLE agent_memory_session
    DROP CONSTRAINT IF EXISTS fk_session_user,
    DROP CONSTRAINT IF EXISTS uk_session_user;