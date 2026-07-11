DROP INDEX IF EXISTS uk_turn_lease_result_message;
DROP INDEX IF EXISTS uk_turn_lease_user_message;

ALTER TABLE agent_turn_lease
    DROP CONSTRAINT IF EXISTS ck_turn_lease_completed_result,
    DROP CONSTRAINT IF EXISTS fk_turn_lease_result_message,
    DROP CONSTRAINT IF EXISTS fk_turn_lease_user_message,
    DROP CONSTRAINT IF EXISTS ck_turn_lease_attempt_no,
    DROP COLUMN IF EXISTS result_message_id,
    DROP COLUMN IF EXISTS user_message_id,
    DROP COLUMN IF EXISTS attempt_no;