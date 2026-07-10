-- 用户身份主体表。认证凭证（Guest token、密码、OAuth）存放在独立凭证表中。
CREATE TABLE agent_user (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id      VARCHAR(64)  NOT NULL,              -- 服务端内部稳定业务ID，由应用使用crypto/rand生成
    username     VARCHAR(64),                        -- 用户可感知的自定义名称，Guest阶段允许为空
    user_type    SMALLINT     NOT NULL DEFAULT 0,   -- 0=guest 1=registered
    status       SMALLINT     NOT NULL DEFAULT 0,   -- 0=active 1=suspended 2=deactivated
    auth_version INTEGER      NOT NULL DEFAULT 1,   -- 全设备认证版本，修改密码/退出全部设备时递增

    deleted_at   TIMESTAMPTZ,                       -- 软删除时间，NULL表示未删除
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT uk_agent_user_user_id UNIQUE (user_id),
    CONSTRAINT uk_agent_user_username UNIQUE (username),
    CONSTRAINT ck_agent_user_type CHECK (user_type IN (0, 1)),
    CONSTRAINT ck_agent_user_status CHECK (status IN (0, 1, 2)),
    CONSTRAINT ck_agent_user_auth_version CHECK (auth_version > 0),
    CONSTRAINT ck_agent_user_user_id_not_blank CHECK (btrim(user_id) <> ''),
    CONSTRAINT ck_agent_user_username_not_blank CHECK (username IS NULL OR btrim(username) <> '')
);

-- 复用 000001 定义的 set_updated_at() 触发器函数。
CREATE TRIGGER trg_agent_user_updated_at
    BEFORE UPDATE ON agent_user
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- 仅索引仍然有效的用户；认证与业务查询通常会同时过滤软删除状态。
CREATE INDEX idx_agent_user_active_type ON agent_user (user_type, status)
    WHERE deleted_at IS NULL;

COMMENT ON TABLE  agent_user IS '用户身份主体表，Guest转正式账号时保留同一user_id';
COMMENT ON COLUMN agent_user.id           IS '数据库内部自增主键，不对客户端暴露';
COMMENT ON COLUMN agent_user.user_id      IS '服务端内部稳定业务ID，应用使用密码学安全随机数生成，用户无感知';
COMMENT ON COLUMN agent_user.username     IS '用户自定义名称，Guest阶段可为空';
COMMENT ON COLUMN agent_user.user_type    IS '账号类型: 0=guest, 1=registered';
COMMENT ON COLUMN agent_user.status       IS '账号状态: 0=active, 1=suspended, 2=deactivated';
COMMENT ON COLUMN agent_user.auth_version IS '全设备认证版本，递增后旧版本Token全部失效';
COMMENT ON COLUMN agent_user.deleted_at   IS '软删除时间，NULL表示未删除';
