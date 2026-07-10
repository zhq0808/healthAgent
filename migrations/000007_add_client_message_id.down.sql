DROP INDEX IF EXISTS uk_ame_user_session_client_message;

ALTER TABLE agent_memory_episodic
    DROP COLUMN IF EXISTS client_message_id;
