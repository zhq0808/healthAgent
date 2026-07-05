-- P0 SQLite 建表。全部使用 IF NOT EXISTS，可重复执行（幂等迁移）。
-- 单用户阶段：user_id 使用固定常量，表结构已预留字段，P1 加账号零改表。
--
-- 时间约定：所有 *_at / *_date 字段存 UTC RFC3339Nano 文本（如 2026-07-05T02:03:04Z），
-- 由 Go 层统一格式化/解析；birth_date 只用日期部分 YYYY-MM-DD。

-- 指标标准化字典（解决 空腹血糖/FPG/GLU/葡萄糖 混写 + 单位换算 + 异常判定基准）
CREATE TABLE IF NOT EXISTS metric_definitions (
    code             TEXT PRIMARY KEY,      -- 标准编码，如 fasting_glucose
    display_name     TEXT NOT NULL,
    category         TEXT NOT NULL,         -- glucose / lipid / uric_acid / body
    canonical_unit   TEXT NOT NULL,
    aliases_json     TEXT,                  -- 别名 []
    default_ref_low  REAL,
    default_ref_high REAL,
    critical_high    REAL,
    critical_low     REAL,
    condition        TEXT,                  -- 关联健康状况，如 glucose_control
    created_at       TEXT NOT NULL,
    updated_at       TEXT NOT NULL
);

-- 个人静态画像
CREATE TABLE IF NOT EXISTS profiles (
    user_id          TEXT PRIMARY KEY,
    gender           TEXT CHECK (gender IN ('male','female','unknown')),
    birth_date       TEXT,                  -- YYYY-MM-DD
    height_cm        REAL,
    allergies_json   TEXT,                  -- 过敏源 []
    diseases_json    TEXT,                  -- 疾病史 []
    diet_pref_json   TEXT,                  -- 饮食偏好
    weight_goal_json TEXT,                  -- 减肥/体重目标
    created_at       TEXT NOT NULL,
    updated_at       TEXT NOT NULL
);

-- 身体指标时间序列（体重/体脂率/腰围等）
CREATE TABLE IF NOT EXISTS body_metrics (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id      TEXT NOT NULL,
    metric_type  TEXT NOT NULL,
    value        REAL NOT NULL,
    unit         TEXT NOT NULL,
    measured_at  TEXT NOT NULL,
    source       TEXT CHECK (source IN ('manual','file','text','voice','import')),
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_body_metrics_user_type
    ON body_metrics (user_id, metric_type, measured_at);
-- 幂等：同用户/类型/时间/来源不重复入库
CREATE UNIQUE INDEX IF NOT EXISTS uq_body_metrics_user_type_time_source
    ON body_metrics (user_id, metric_type, measured_at, COALESCE(source, ''));

-- 体检记录（一次体检）
CREATE TABLE IF NOT EXISTS medical_records (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     TEXT NOT NULL,
    hospital    TEXT,
    checked_at  TEXT NOT NULL,
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_medical_records_user_checked
    ON medical_records (user_id, checked_at);
CREATE UNIQUE INDEX IF NOT EXISTS uq_medical_records_user_checked_hospital
    ON medical_records (user_id, checked_at, COALESCE(hospital, ''));

-- 单项检验值（含标准化后的值、原始抽取、异常标记、规则版本）
CREATE TABLE IF NOT EXISTS lab_results (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id       TEXT NOT NULL,           -- 冗余，便于按用户查询/删除
    record_id     INTEGER NOT NULL,
    item_code     TEXT NOT NULL,           -- 标准化后的指标编码
    value         REAL NOT NULL,           -- 已换算为 canonical 单位
    unit          TEXT NOT NULL,
    ref_low       REAL,
    ref_high      REAL,
    ref_source    TEXT CHECK (ref_source IN ('report','system_default')),
    abnormal_flag TEXT NOT NULL DEFAULT 'normal'
                  CHECK (abnormal_flag IN ('normal','high','low','critical_high','critical_low')),
    raw_item_name TEXT,                     -- 原始名称，如 "空腹血糖"
    raw_value     TEXT,                     -- 原始值文本
    raw_unit      TEXT,                     -- 原始单位
    normalized_by_version TEXT,             -- 指标字典版本
    rules_version TEXT,                     -- 异常判定规则版本
    created_at    TEXT NOT NULL,
    updated_at    TEXT NOT NULL,
    FOREIGN KEY (record_id) REFERENCES medical_records (id) ON DELETE CASCADE
);
-- 幂等：同一体检记录下同一指标唯一
CREATE UNIQUE INDEX IF NOT EXISTS uq_lab_results_record_item
    ON lab_results (record_id, item_code);
-- 高频查询：按用户查某指标的异常趋势
CREATE INDEX IF NOT EXISTS idx_lab_results_user_item_flag
    ON lab_results (user_id, item_code, abnormal_flag);

-- LLM 抽取草稿态（两步录入：抽取 → 确认落库）
CREATE TABLE IF NOT EXISTS intake_drafts (
    id                    TEXT PRIMARY KEY,   -- draft_id
    user_id               TEXT NOT NULL,
    raw_text              TEXT NOT NULL,
    extracted_json        TEXT NOT NULL,
    status                TEXT NOT NULL DEFAULT 'pending'
                          CHECK (status IN ('pending','confirmed','discarded')),
    confirmed_result_json TEXT,              -- 确认后生成了哪些数据，供回溯
    error_message         TEXT,
    created_at            TEXT NOT NULL,
    confirmed_at          TEXT,
    discarded_at          TEXT
);
CREATE INDEX IF NOT EXISTS idx_intake_drafts_user_status
    ON intake_drafts (user_id, status, created_at);

-- 每日饮食建议（含版本快照，可追溯）
CREATE TABLE IF NOT EXISTS diet_advices (
    id                     INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id                TEXT NOT NULL,
    advice_json            TEXT NOT NULL,
    based_on_snapshot_json TEXT,             -- 当时基于哪些异常指标
    model                  TEXT,
    prompt_version         TEXT,
    rules_version          TEXT,
    created_at             TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_diet_advices_user
    ON diet_advices (user_id, created_at);

-- 吃了啥记录
CREATE TABLE IF NOT EXISTS meal_logs (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    TEXT NOT NULL,
    raw_text   TEXT NOT NULL,
    source     TEXT CHECK (source IN ('manual','file','text','voice','import')),
    items_json TEXT,
    logged_at  TEXT NOT NULL,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_meal_logs_user
    ON meal_logs (user_id, logged_at);
