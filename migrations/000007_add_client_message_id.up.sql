ALTER TABLE agent_memory_episodic
    ADD COLUMN client_message_id UUID;

CREATE UNIQUE INDEX uk_ame_user_session_client_message
    ON agent_memory_episodic (user_id, session_id, client_message_id)
    WHERE role = 'user' AND client_message_id IS NOT NULL;
