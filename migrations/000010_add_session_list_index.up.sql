CREATE INDEX idx_sess_user_last_message
    ON agent_memory_session (
        user_id,
        (COALESCE(last_message_at, created_at)) DESC,
        created_at DESC,
        session_id DESC
    )
    WHERE deleted_at IS NULL;