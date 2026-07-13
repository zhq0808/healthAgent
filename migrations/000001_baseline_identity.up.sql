-- ============================================================================
-- Baseline 000001 · 身份主体
-- 项目未上线，开发期迁移已压缩为 v2 基线；本文件不再叠加补丁迁移。
-- 包含：全局 updated_at 触发器函数、用户主体表、Guest 设备凭证表。
-- ============================================================================

-- updated_at 自动维护函数（全库共用，后续所有表的 updated_at 触发器都复用它）。
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

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

-- Guest 设备凭证表。浏览器只持有 HttpOnly Cookie 中的明文 token，数据库仅保存 SHA-256 hash。
CREATE TABLE guest_credential (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id      VARCHAR(64) NOT NULL,
    token_hash   BYTEA       NOT NULL,
    expires_at   TIMESTAMPTZ NOT NULL,
    revoked_at   TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uk_guest_credential_token_hash UNIQUE (token_hash),
    CONSTRAINT fk_guest_credential_user FOREIGN KEY (user_id)
        REFERENCES agent_user(user_id) ON DELETE RESTRICT,
    CONSTRAINT ck_guest_credential_hash_length CHECK (octet_length(token_hash) = 32),
    CONSTRAINT ck_guest_credential_expiry CHECK (expires_at > created_at),
    CONSTRAINT ck_guest_credential_revoked_at CHECK (revoked_at IS NULL OR revoked_at >= created_at)
);

CREATE INDEX idx_guest_credential_user_active ON guest_credential (user_id, expires_at DESC)
    WHERE revoked_at IS NULL;
CREATE INDEX idx_guest_credential_expiry ON guest_credential (expires_at)
    WHERE revoked_at IS NULL;

CREATE TRIGGER trg_guest_credential_updated_at
    BEFORE UPDATE ON guest_credential
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE  guest_credential IS 'Guest设备认证凭证，仅保存opaque token的SHA-256 hash';
COMMENT ON COLUMN guest_credential.user_id      IS '凭证所属的稳定用户业务ID';
COMMENT ON COLUMN guest_credential.token_hash   IS 'Guest opaque token的SHA-256 hash，禁止保存明文token';
COMMENT ON COLUMN guest_credential.expires_at   IS '凭证过期时间';
COMMENT ON COLUMN guest_credential.revoked_at   IS '凭证撤销时间，NULL表示未撤销';
COMMENT ON COLUMN guest_credential.last_used_at IS '最近一次成功识别该设备的时间';
