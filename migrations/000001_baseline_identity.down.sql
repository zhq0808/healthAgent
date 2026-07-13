-- Baseline 000001 回滚：先删依赖 agent_user 的凭证表，再删主体表，最后删共用函数。
DROP TABLE IF EXISTS guest_credential;
DROP TABLE IF EXISTS agent_user;
DROP FUNCTION IF EXISTS set_updated_at();
