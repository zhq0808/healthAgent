// Package model 定义核心领域数据结构。
//
// 约定：
//   - 所有时间字段在 Go 层统一用 time.Time（UTC），由 store 层负责与 SQLite 的
//     RFC3339Nano 文本互转。
//   - 枚举使用自定义 string 类型 + 常量，DB 层用 TEXT + CHECK 约束兜底。
//   - 可空字段使用指针（如 *float64），区分“0”与“未填写”。
package model

import "time"

// UserID 是用户标识。P0 单用户阶段使用固定常量。
type UserID string

// DefaultUserID 是 P0 单用户阶段的固定用户 ID。
const DefaultUserID UserID = "local"

// Gender 性别枚举。
type Gender string

const (
	GenderMale    Gender = "male"
	GenderFemale  Gender = "female"
	GenderUnknown Gender = "unknown"
)

// InputSource 数据来源枚举。
type InputSource string

const (
	SourceManual InputSource = "manual"
	SourceFile   InputSource = "file"
	SourceText   InputSource = "text"
	SourceVoice  InputSource = "voice"
	SourceImport InputSource = "import"
)

// MetricType 身体指标类型枚举。
type MetricType string

const (
	MetricWeight  MetricType = "weight"
	MetricBodyFat MetricType = "body_fat"
	MetricWaist   MetricType = "waist"
)

// AbnormalFlag 体检指标异常标记枚举。
type AbnormalFlag string

const (
	AbnormalNormal       AbnormalFlag = "normal"
	AbnormalHigh         AbnormalFlag = "high"
	AbnormalLow          AbnormalFlag = "low"
	AbnormalCriticalHigh AbnormalFlag = "critical_high"
	AbnormalCriticalLow  AbnormalFlag = "critical_low"
)

// RefSource 参考区间来源枚举。
type RefSource string

const (
	RefFromReport  RefSource = "report"         // 来自体检报告原文
	RefFromDefault RefSource = "system_default" // 来自系统内置字典
)

// DraftStatus 录入草稿状态枚举。
type DraftStatus string

const (
	DraftPending   DraftStatus = "pending"
	DraftConfirmed DraftStatus = "confirmed"
	DraftDiscarded DraftStatus = "discarded"
)

// WeightGoal 减肥/体重目标。
type WeightGoal struct {
	TargetWeightKG  *float64 `json:"target_weight_kg,omitempty"`
	Deadline        *string  `json:"deadline,omitempty"` // YYYY-MM-DD
	DailyCalorieGap *int     `json:"daily_calorie_gap,omitempty"`
}

// DietPreference 饮食偏好。
type DietPreference struct {
	Likes    []string `json:"likes,omitempty"`
	Dislikes []string `json:"dislikes,omitempty"`
	Style    string   `json:"style,omitempty"` // 如 清淡 / 川菜 / 素食
}

// Profile 个人静态画像（聚合根）。
type Profile struct {
	UserID     UserID
	Gender     Gender
	BirthDate  *time.Time // 仅日期部分有意义
	HeightCM   *float64
	Allergies  []string // 过敏源，永远硬过滤
	Diseases   []string // 疾病史
	DietPref   DietPreference
	WeightGoal WeightGoal
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// BodyMetric 身体指标时间序列的一条测量。
type BodyMetric struct {
	ID         int64
	UserID     UserID
	Type       MetricType
	Value      float64
	Unit       string
	MeasuredAt time.Time
	Source     InputSource
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// MedicalRecord 一次体检记录。
type MedicalRecord struct {
	ID        int64
	UserID    UserID
	Hospital  string
	CheckedAt time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
	Labs      []LabResult // 组装用，非持久化字段
}

// LabResult 单项检验值。value/unit 为标准化后的 canonical 值，
// raw_* 保留 LLM/文件原始抽取，供标准化规则修正时回溯。
type LabResult struct {
	ID           int64
	UserID       UserID // 冗余，便于按用户查询与删除
	RecordID     int64
	ItemCode     string // 标准化编码，如 fasting_glucose
	Value        float64
	Unit         string
	RefLow       *float64
	RefHigh      *float64
	RefSource    RefSource
	AbnormalFlag AbnormalFlag

	RawItemName string // 原始名称，如 "空腹血糖"/"FPG"
	RawValue    string // 原始值文本
	RawUnit     string // 原始单位

	NormalizedByVersion string // 指标字典版本
	RulesVersion        string // 异常判定规则版本

	CreatedAt time.Time
	UpdatedAt time.Time
}

// IntakeDraft LLM 抽取草稿。两步录入：抽取 → 用户确认后落库。
type IntakeDraft struct {
	ID                  string // draft_id
	UserID              UserID
	RawText             string
	ExtractedJSON       string
	Status              DraftStatus
	ConfirmedResultJSON string
	ErrorMessage        string
	CreatedAt           time.Time
	ConfirmedAt         *time.Time
	DiscardedAt         *time.Time
}

// DietAdvice 每日饮食建议，含版本快照可追溯。
type DietAdvice struct {
	ID              int64
	UserID          UserID
	AdviceJSON      string
	BasedOnSnapshot string // 当时基于哪些异常指标
	Model           string
	PromptVersion   string
	RulesVersion    string
	CreatedAt       time.Time
}

// MealLog 吃了啥记录。P0 只存 raw_text，不强抽结构化。
type MealLog struct {
	ID        int64
	UserID    UserID
	RawText   string
	Source    InputSource
	ItemsJSON string
	LoggedAt  time.Time
	CreatedAt time.Time
}

// MetricDefinition 指标标准化字典的一条定义。
type MetricDefinition struct {
	Code           string // 标准编码，如 fasting_glucose
	DisplayName    string
	Category       string // glucose / lipid / uric_acid / body ...
	CanonicalUnit  string
	Aliases        []string
	DefaultRefLow  *float64
	DefaultRefHigh *float64
	CriticalHigh   *float64
	CriticalLow    *float64
	Condition      string // 关联的健康状况，如 glucose_control
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
