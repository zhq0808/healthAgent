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
