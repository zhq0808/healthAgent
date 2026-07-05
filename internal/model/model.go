package model

import "time"

// UserID 是用户标识。
//
// 注意这里没直接用 string，而是 `type UserID string`——这叫“定义命名类型”。
// PHP 没有这个概念（PHP 里 string 就是 string）。好处：函数签名写成
// f(uid UserID) 时，你不小心把一个普通字符串（比如 hospital 名）传进去，
// 编译器会直接报错。等于用类型给“用户 ID”这个概念上了一道锁。
type UserID string

// DefaultUserID 是 P0 单用户阶段的固定用户 ID。
// P0 不做账号系统，所有数据都挂在这个固定用户下；表里已预留 user_id 字段，
// P1 加账号时把这里换成真实 ID 即可，不用改表。
const DefaultUserID UserID = "local"

// Gender 性别枚举。
//
// 【为什么不用裸 string？】这是本文件最值得学的 Go 模式：
// 用 `type Gender string` + 一组常量，模拟其他语言的 enum。
//   - 裸 string：值可以是任意乱七八糟的字符串，"男"/"M"/"male" 全混进来。
//   - 命名类型 + 常量：约定只用下面这几个常量，代码里一眼知道合法值有哪些；
//     配合 schema.sql 里的 CHECK 约束，DB 再兜一道底。
// Go 的 enum 是“君子协定”——编译器不强制只能用这几个常量（不像 Java enum），
// 所以 DB 层的 CHECK 约束是必须的第二道防线。
type Gender string

const (
	GenderMale    Gender = "male"
	GenderFemale  Gender = "female"
	GenderUnknown Gender = "unknown" // 未知/未填，避免用空字符串表达“不知道”
)

// InputSource 数据来源枚举：这条数据是怎么进来的。
// 用途：审计（谁录的）、幂等（body_metrics 的唯一索引把 source 也算进去）、
// 以及日后按来源做质量分析（语音抽取错得多不多）。
type InputSource string

const (
	SourceManual InputSource = "manual" // 用户手动填表
	SourceFile   InputSource = "file"   // 上传结构化文件
	SourceText   InputSource = "text"   // 自由文本经 LLM 抽取
	SourceVoice  InputSource = "voice"  // 语音转文字后抽取
	SourceImport InputSource = "import" // 批量导入
)

// MetricType 身体指标类型枚举（时间序列里存的是哪种指标）。
type MetricType string

const (
	MetricWeight  MetricType = "weight"   // 体重
	MetricBodyFat MetricType = "body_fat" // 体脂率
	MetricWaist   MetricType = "waist"    // 腰围
)

// AbnormalFlag 体检指标异常标记枚举。
//
// 【为什么把它落库而不是每次现算？】读多写少——录入时算一次存进去，
// 之后无数次查询/生成建议都直接读，不重算。critical_high/low 是“危急值”，
// 命中后 rules 层不发挥饮食建议，直接提示就医（合规硬约束）。
type AbnormalFlag string

const (
	AbnormalNormal       AbnormalFlag = "normal"        // 正常
	AbnormalHigh         AbnormalFlag = "high"          // 偏高
	AbnormalLow          AbnormalFlag = "low"           // 偏低
	AbnormalCriticalHigh AbnormalFlag = "critical_high" // 危急偏高 → 提示就医
	AbnormalCriticalLow  AbnormalFlag = "critical_low"  // 危急偏低 → 提示就医
)

// RefSource 参考区间来源枚举：这条指标的“正常范围”是哪来的。
// 优先用体检报告原文的区间（report）；报告没给才退回系统字典默认值
// （system_default）——并标出来，不假装自己的默认值很权威。
type RefSource string

const (
	RefFromReport  RefSource = "report"         // 来自体检报告原文
	RefFromDefault RefSource = "system_default" // 来自系统内置字典
)

// DraftStatus 录入草稿状态枚举，配合“两步录入”用（抽取→确认）。
type DraftStatus string

const (
	DraftPending   DraftStatus = "pending"   // LLM 抽完了，等用户确认
	DraftConfirmed DraftStatus = "confirmed" // 用户确认，已落库
	DraftDiscarded DraftStatus = "discarded" // 用户丢弃
)

// WeightGoal 减肥/体重目标。
//
// 【注意这里的字段带 json tag，前面的实体大多不带——为什么？】
// WeightGoal / DietPreference 是要被序列化成 JSON 存进 profiles 表的
// weight_goal_json / diet_pref_json 列的（整块存 JSON，不拆列）。
// 带 json tag 是为了控制 JSON 里的字段名（蛇形 target_weight_kg）。
// 而 Profile/BodyMetric 那些是按列存的，不整体序列化，就不需要 tag。
//
// omitempty：字段为零值（nil/0/""）时，序列化的 JSON 里直接不出现这个 key，
// 省空间也更干净。
type WeightGoal struct {
	TargetWeightKG  *float64 `json:"target_weight_kg,omitempty"` // 指针=可不填
	Deadline        *string  `json:"deadline,omitempty"`         // YYYY-MM-DD
	DailyCalorieGap *int     `json:"daily_calorie_gap,omitempty"`
}

// DietPreference 饮食偏好。同样整块存 JSON，故带 json tag。
type DietPreference struct {
	Likes    []string `json:"likes,omitempty"`
	Dislikes []string `json:"dislikes,omitempty"`
	Style    string   `json:"style,omitempty"` // 如 清淡 / 川菜 / 素食
}

// Profile 个人静态画像（一个用户一条）。
//
// 【为什么有的字段是指针有的不是？】
//   - HeightCM *float64、BirthDate *time.Time：可能没填，用指针的 nil 表达“缺失”。
//   - Allergies []string：切片本身零值就是 nil，不需要再套指针。
//   - CreatedAt/UpdatedAt time.Time：一定有值，不用指针。
type Profile struct {
	UserID     UserID
	Gender     Gender
	BirthDate  *time.Time // 仅日期部分有意义；可空
	HeightCM   *float64   // 可空
	Allergies  []string   // 过敏源，rules 层永远硬过滤（跟疾病规则独立）
	Diseases   []string   // 疾病史
	DietPref   DietPreference
	WeightGoal WeightGoal
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// BodyMetric 身体指标时间序列里的“一条”测量。
//
// 【为什么做时间序列不做单值？】饮食建议的价值在“趋势”——近一个月胖了 3kg
// 和体重一直稳，建议完全不同。单值会丢历史，也没法回答“这条建议基于什么数据”。
// 所以每次测量都新增一行，不覆盖。
type BodyMetric struct {
	ID         int64 // 自增主键。Go 里数据库整型主键惯用 int64
	UserID     UserID
	Type       MetricType
	Value      float64
	Unit       string
	MeasuredAt time.Time // 测量发生的时间（业务时间，用户给的）
	Source     InputSource
	CreatedAt  time.Time // 这行记录写入 DB 的时间（系统时间）
	UpdatedAt  time.Time
}

// MedicalRecord 一次体检（一次体检含多个检验项）。
type MedicalRecord struct {
	ID        int64
	UserID    UserID
	Hospital  string
	CheckedAt time.Time
	CreatedAt time.Time
	UpdatedAt time.Time

	// Labs 是“组装用”字段，不是数据库里的列。
	// 从 DB 读出来时，service 层把关联的 LabResult 填进来，方便一次性把
	// “一次体检 + 它的所有检验项”当一个整体传递。存的时候它们是分表存的
	// （medical_records 一行 + lab_results 多行），不会把 Labs 序列化进某列。
	Labs []LabResult
}

// LabResult 单项检验值（比如“空腹血糖 7.2 mmol/L”）。
//
// 【本结构体的核心设计：canonical 值 + raw 原文 双存】
//   - Value/Unit：标准化后的“干净值”，单位已换算成 canonical（如统一到 mmol/L）。
//     异常判定、趋势、生成建议都用这套。
//   - Raw*：LLM/文件抽取时的原始文本原样保留。
//     为什么留？①出问题能回溯“到底原文写的啥、是不是我换算错了” ②以后字典
//     升级重新标准化时，能拿原文重算，而不是拿已经被我处理过的值二次加工。
//   - NormalizedByVersion/RulesVersion：记录“这条数据是用哪个版本的字典/规则算的”。
//     AI 产品最容易返工——字典一改，历史数据的判定就可能变，版本号让你能解释
//     “当时为什么判成异常”。
type LabResult struct {
	ID           int64
	UserID       UserID // 冗余存一份，便于按用户直接查/删（不用先 join 体检表）
	RecordID     int64  // 外键：属于哪次体检（medical_records.id）
	ItemCode     string // 标准化编码，如 fasting_glucose
	Value        float64
	Unit         string
	RefLow       *float64 // 参考区间下限，可空
	RefHigh      *float64 // 参考区间上限，可空
	RefSource    RefSource
	AbnormalFlag AbnormalFlag

	RawItemName string // 原始名称，如 "空腹血糖"/"FPG"
	RawValue    string // 原始值文本（原样，不转 float）
	RawUnit     string // 原始单位

	NormalizedByVersion string // 指标字典版本
	RulesVersion        string // 异常判定规则版本

	CreatedAt time.Time
	UpdatedAt time.Time
}

// IntakeDraft LLM 抽取草稿。
//
// 【为什么要“草稿”这一步？】LLM 抽取会出错（数值抽错、单位搞混）。不能抽完直接
// 落库，得让用户先确认。所以先把抽取结果存成 pending 草稿，用户点“确认”才真正
// 写进正式表。ID（draft_id）还用来做幂等：用户手抖点两次确认，靠同一个 draft_id
// 保证只落一次库。
type IntakeDraft struct {
	ID                  string // draft_id（字符串主键，通常是随机 ID）
	UserID              UserID
	RawText             string // 用户原始输入
	ExtractedJSON       string // LLM 抽取出的结构化结果（JSON 文本）
	Status              DraftStatus
	ConfirmedResultJSON string     // 确认后生成了哪些数据，供回溯
	ErrorMessage        string     // 确认失败时的原因
	CreatedAt           time.Time  //
	ConfirmedAt         *time.Time // 可空：还没确认时为 nil
	DiscardedAt         *time.Time // 可空：还没丢弃时为 nil
}

// DietAdvice 每日饮食建议，含版本快照可追溯。
//
// AdviceJSON 存整块建议 JSON；BasedOnSnapshot/Model/PromptVersion/RulesVersion
// 记录“这条建议是基于当时哪些异常指标、用哪个模型和哪版 prompt/规则生成的”，
// 保证历史建议可解释、可复盘。
type DietAdvice struct {
	ID              int64
	UserID          UserID
	AdviceJSON      string
	BasedOnSnapshot string // 当时基于哪些异常指标（快照）
	Model           string // 用的哪个 LLM 模型
	PromptVersion   string
	RulesVersion    string
	CreatedAt       time.Time
}

// MealLog 吃了啥记录。
// P0 只存 raw_text（“今天中午吃了红烧肉+米饭”原文），不强行抽结构化 items，
// ItemsJSON 预留给 P1 精算热量时用。
type MealLog struct {
	ID        int64
	UserID    UserID
	RawText   string
	Source    InputSource
	ItemsJSON string // P1 才填：结构化后的食物项
	LoggedAt  time.Time
	CreatedAt time.Time
}

// MetricDefinition 指标标准化字典的一条定义（存在 metric_definitions 表）。
//
// 【它解决什么问题？】自由文本里同一个指标有好几种叫法：
// “空腹血糖 / FPG / 葡萄糖 / GLU”。不统一，异常判定/趋势/建议关联全乱。
// 这张字典把别名（Aliases）都映射到一个标准 Code，并带上换算基准和参考区间，
// 是标准化和异常判定的“真值来源”。
type MetricDefinition struct {
	Code           string   // 标准编码，如 fasting_glucose
	DisplayName    string   // 展示名，如 “空腹血糖”
	Category       string   // glucose / lipid / uric_acid / body ...
	CanonicalUnit  string   // 标准单位，所有值最终换算到它
	Aliases        []string // 别名，如 ["空腹血糖","FPG","葡萄糖","GLU"]
	DefaultRefLow  *float64 // 默认参考下限（报告没给区间时用），可空
	DefaultRefHigh *float64 // 默认参考上限，可空
	CriticalHigh   *float64 // 危急高阈值，命中→提示就医，可空
	CriticalLow    *float64 // 危急低阈值，可空
	Condition      string   // 关联的健康状况，如 glucose_control
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
