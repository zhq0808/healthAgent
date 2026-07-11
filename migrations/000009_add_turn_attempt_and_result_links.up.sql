ALTER TABLE agent_turn_lease
    ADD COLUMN attempt_no BIGINT NOT NULL DEFAULT 1,
    ADD COLUMN user_message_id BIGINT,
    ADD COLUMN result_message_id BIGINT;

UPDATE agent_turn_lease AS turn
SET user_message_id = message.id
FROM agent_memory_episodic AS message
WHERE message.session_id = turn.session_id
  AND message.user_id = turn.user_id
  AND message.client_message_id = turn.client_message_id
  AND message.role = 'user';

UPDATE agent_turn_lease AS turn
SET result_message_id = reply.id
FROM agent_memory_episodic AS user_message
JOIN agent_memory_episodic AS reply
  ON reply.session_id = user_message.session_id
 AND reply.seq = user_message.seq + 1
 AND reply.role = 'assistant'
 AND reply.status = 'completed'
 AND reply.deleted_at IS NULL
WHERE turn.user_message_id = user_message.id
  AND turn.status = 'completed';

ALTER TABLE agent_turn_lease
    ADD CONSTRAINT ck_turn_lease_attempt_no CHECK (attempt_no > 0),
    ADD CONSTRAINT fk_turn_lease_user_message FOREIGN KEY (user_message_id)
        REFERENCES agent_memory_episodic(id) ON DELETE RESTRICT,
    ADD CONSTRAINT fk_turn_lease_result_message FOREIGN KEY (result_message_id)
        REFERENCES agent_memory_episodic(id) ON DELETE RESTRICT,
    ADD CONSTRAINT ck_turn_lease_completed_result
        CHECK (status <> 'completed' OR (user_message_id IS NOT NULL AND result_message_id IS NOT NULL));

CREATE UNIQUE INDEX uk_turn_lease_user_message
    ON agent_turn_lease (user_message_id)
    WHERE user_message_id IS NOT NULL;

CREATE UNIQUE INDEX uk_turn_lease_result_message
    ON agent_turn_lease (result_message_id)
    WHERE result_message_id IS NOT NULL;

COMMENT ON COLUMN agent_turn_lease.attempt_no IS '执行代次；每次过期恢复或失败重试递增，用作fencing token';
COMMENT ON COLUMN agent_turn_lease.user_message_id IS '本turn对应的用户消息ID';
COMMENT ON COLUMN agent_turn_lease.result_message_id IS 'completed turn对应的assistant结果消息ID';