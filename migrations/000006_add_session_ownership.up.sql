-- 会话必须属于真实用户；消息和记忆中的 user_id 必须与会话所有者一致。
ALTER TABLE agent_memory_session
    ADD CONSTRAINT uk_session_user UNIQUE (session_id, user_id),
    ADD CONSTRAINT fk_session_user FOREIGN KEY (user_id)
        REFERENCES agent_user(user_id) ON DELETE RESTRICT;

ALTER TABLE agent_memory_episodic
    ADD CONSTRAINT fk_episodic_session_user FOREIGN KEY (session_id, user_id)
        REFERENCES agent_memory_session(session_id, user_id) ON DELETE RESTRICT;

ALTER TABLE agent_memory_meta
    ADD CONSTRAINT fk_meta_session_user FOREIGN KEY (session_id, user_id)
        REFERENCES agent_memory_session(session_id, user_id) ON DELETE RESTRICT;
