-- 会话活跃 turn 租约表：一个 Session 在任意时刻最多一个进行中的 turn（一问一答的完整处理周期）。
--
-- 为什么不能只靠 agent_memory_session 行锁：一次 turn 要跨"调用 LLM + 写 SSE"这段可能长达数十秒的
-- 时间，不能整段持有数据库事务/行锁；因此把"占用状态"单独落成一张带 TTL 的租约表，
-- 获取/释放/续期各自只用短事务，租约本身靠过期时间兜底恢复（进程崩溃/网络异常不会永久卡死会话）。
CREATE TABLE agent_turn_lease (
    id                 BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    session_id         VARCHAR(64)  NOT NULL,
    user_id            VARCHAR(64)  NOT NULL,            -- 数据归属用户，配合 FK 校验会话一致性
    client_message_id  UUID         NOT NULL,            -- 触发本次 turn 的用户消息幂等键

    status             VARCHAR(20)  NOT NULL DEFAULT 'active', -- active/completed/failed
    lease_expires_at   TIMESTAMPTZ  NOT NULL,             -- 租约到期时间，过期视为可被重新获取

    created_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT ck_turn_lease_status CHECK (status IN ('active', 'completed', 'failed')),
    -- 同一用户消息重试只应命中同一条租约记录，不产生新行（配合结果恢复协议）。
    CONSTRAINT uk_turn_lease_session_client UNIQUE (session_id, client_message_id),
    CONSTRAINT fk_turn_lease_session_user FOREIGN KEY (session_id, user_id)
        REFERENCES agent_memory_session(session_id, user_id) ON DELETE RESTRICT
);

-- 核心约束：一个 Session 同一时刻最多一条 active 记录，跨进程/跨实例也由数据库强制，不依赖内存锁。
CREATE UNIQUE INDEX uk_turn_lease_active_session ON agent_turn_lease (session_id)
    WHERE status = 'active';

-- 供后台巡检/下一次获取时扫描"已过期但仍标记 active"的租约，尽快判定可回收。
CREATE INDEX idx_turn_lease_expires ON agent_turn_lease (lease_expires_at)
    WHERE status = 'active';

-- 复用 000001 定义的 set_updated_at() 触发器函数。
CREATE TRIGGER trg_turn_lease_updated_at
    BEFORE UPDATE ON agent_turn_lease
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE  agent_turn_lease IS '会话活跃 turn 租约表：保证一个 Session 同一时刻最多一个进行中的 turn';
COMMENT ON COLUMN agent_turn_lease.session_id        IS '会话ID';
COMMENT ON COLUMN agent_turn_lease.user_id           IS '数据归属用户ID';
COMMENT ON COLUMN agent_turn_lease.client_message_id IS '触发本次turn的用户消息幂等键，用于重试恢复';
COMMENT ON COLUMN agent_turn_lease.status            IS '租约状态: active/completed/failed';
COMMENT ON COLUMN agent_turn_lease.lease_expires_at  IS '租约到期时间，过期后允许被重新获取';
COMMENT ON COLUMN agent_turn_lease.created_at        IS '租约首次创建时间';
COMMENT ON COLUMN agent_turn_lease.updated_at        IS '租约最后一次状态变更时间';
