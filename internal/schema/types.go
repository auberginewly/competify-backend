// Package schema defines all data contracts flowing between Agents.
// This file is the SINGLE SOURCE OF TRUTH — tygo generates frontend TS types from it.
// Run `make sync-types` after any change here.
package schema

import "time"

// ============ Agent Runtime ============

// AgentStatus is the runtime state of an Agent in the DAG.
// Mirrors the frontend status color encoding (see frontend docs/styling-guide.md).
type AgentStatus string

const (
	AgentStatusPending AgentStatus = "pending"
	AgentStatusRunning AgentStatus = "running"
	AgentStatusReview  AgentStatus = "review"
	AgentStatusError   AgentStatus = "error"
	AgentStatusDone    AgentStatus = "done"
)

// ============ Orchestrator ============

// UserQuery 用户输入请求
type UserQuery struct {
	CompetitorName string   `json:"competitor_name"`
	TargetURL      string   `json:"target_url"`
	Dimensions     []string `json:"dimensions"` // pricing / feature / tech / market
	Priority       int      `json:"priority"`   // 1-5
	RequestedBy    string   `json:"requested_by"`
}

// TaskDAGPlan Orchestrator 输出的任务执行计划
type TaskDAGPlan struct {
	TaskID            string    `json:"task_id"`
	CompetitorName    string    `json:"competitor_name"`
	Dimensions        []string  `json:"dimensions"`
	RequiresWebScrape bool      `json:"requires_web_scrape"`
	RequiresSocial    bool      `json:"requires_social"`
	RequiresFinancial bool      `json:"requires_financial"`
	RequiresReview    bool      `json:"requires_review"`
	RequiresAPI       bool      `json:"requires_api"`
	TargetSchema      string    `json:"target_schema"`
	CreatedAt         time.Time `json:"created_at"`
	ExpiresAt         time.Time `json:"expires_at"`
}

// ============ Collector & Cleaner ============

// RawDataPack 采集器原始输出
type RawDataPack struct {
	TaskID      string            `json:"task_id"`
	SourceType  string            `json:"source_type"` // web / social / financial / review / api
	SourceURL   string            `json:"source_url"`
	RawContent  string            `json:"raw_content"`
	Headers     map[string]string `json:"headers"`
	StatusCode  int               `json:"status_code"`
	CapturedAt  time.Time         `json:"captured_at"`
	CollectorID string            `json:"collector_id"`
	Fingerprint string            `json:"fingerprint"` // SHA-256 of raw content
}

// NormalizedDataset Cleaner 清洗后输出
type NormalizedDataset struct {
	TaskID       string                 `json:"task_id"`
	VikingURI    string                 `json:"viking_uri"`
	SourceType   string                 `json:"source_type"`
	CleanedText  string                 `json:"cleaned_text"`
	Structured   map[string]interface{} `json:"structured"`
	SimHashValue uint64                 `json:"sim_hash_value"`
	Confidence   float64                `json:"confidence"`
	CleanedAt    time.Time              `json:"cleaned_at"`
	CleanerID    string                 `json:"cleaner_id"`
}

// ============ Analyzer ============

// AnalysisResult 分析师输出
type AnalysisResult struct {
	TaskID     string    `json:"task_id"`
	Dimension  string    `json:"dimension"` // pricing / feature / tech / market
	Payload    string    `json:"payload"`
	CoTReason  string    `json:"cot_reason"`
	Score      float64   `json:"score"` // confidence 0-1
	SourceURIs []string  `json:"source_uris"`
	AnalyzerID string    `json:"analyzer_id"`
	ModelName  string    `json:"model_name"`
	AnalyzedAt time.Time `json:"analyzed_at"`
}

// ============ Cross-Reviewer ============

// ReviewReport 交叉审查输出
type ReviewReport struct {
	TaskID       string     `json:"task_id"`
	IsApproved   bool       `json:"is_approved"`
	Conflicts    []Conflict `json:"conflicts"`
	NextAction   string     `json:"next_action"` // APPROVE / RETRY_AUTO / REJECT_HUMAN
	ReviewedAt   time.Time  `json:"reviewed_at"`
	ReviewerID   string     `json:"reviewer_id"`
	DebateRounds int        `json:"debate_rounds"`
}

// Conflict 单条冲突描述
type Conflict struct {
	Dimension    string   `json:"dimension"`
	AgentIDs     []string `json:"agent_ids"`
	Description  string   `json:"description"`
	Severity     string   `json:"severity"` // low / medium / high
	SuggestedFix string   `json:"suggested_fix"`
}

// ============ Writer ============

// DraftReport 撰写员输出
type DraftReport struct {
	TaskID          string     `json:"task_id"`
	Title           string     `json:"title"`
	MarkdownContent string     `json:"markdown_content"`
	Footnotes       []Footnote `json:"footnotes"`
	MerkleRootHash  string     `json:"merkle_root_hash"`
	Confidence      float64    `json:"confidence"`
	GeneratedAt     time.Time  `json:"generated_at"`
	WriterID        string     `json:"writer_id"`
}

// Footnote 溯源脚注
type Footnote struct {
	ID           string  `json:"id"`
	Conclusion   string  `json:"conclusion"`
	Confidence   float64 `json:"confidence"`
	ProvenanceID string  `json:"provenance_id"`
	VikingURI    string  `json:"viking_uri"`
}

// ============ Final-Reviewer ============

// FinalReviewInput 终审员输入
type FinalReviewInput struct {
	TaskID         string    `json:"task_id"`
	ReportID       string    `json:"report_id"`
	Content        string    `json:"content"`
	MerkleRootHash string    `json:"merkle_root_hash"`
	Confidence     float64   `json:"confidence"`
	GeneratedAt    time.Time `json:"generated_at"`
}

// FinalReviewOutput 终审员输出
type FinalReviewOutput struct {
	ReportID   string    `json:"report_id"`
	Content    string    `json:"content"`
	Status     string    `json:"status"` // APPROVED / REJECTED / NEEDS_REVISION
	Signature  string    `json:"signature"`
	Comment    string    `json:"comment"`
	ApprovedAt time.Time `json:"approved_at"`
	ApprovedBy string    `json:"approved_by"`
}

// ============ Final Output ============

// FinalReport 系统最终输出
type FinalReport struct {
	TaskID     string    `json:"task_id"`
	ReportID   string    `json:"report_id"`
	Content    string    `json:"content"`
	Status     string    `json:"status"` // published / draft / rejected
	Signature  string    `json:"signature"`
	ApprovedBy string    `json:"approved_by"`
	ApprovedAt time.Time `json:"approved_at"`
	MerkleRoot string    `json:"merkle_root"`
}

// ============ Ontology Types ============
// 下面这段由 schema/generator.go 从 schema.yaml 生成，不要手改。
// 改 ontology 请改 schema.yaml + 跑 make schema-gen。

// AUTOGEN:ONTOLOGY:START

// Competitor 竞品实体
type Competitor struct {
	UID          string        `json:"uid,omitempty"`
	CompanyName  string        `json:"company_name,omitempty"`
	FoundedDate  *time.Time    `json:"founded_date,omitempty"`
	FundingStage string        `json:"funding_stage,omitempty"`
	Headquarters string        `json:"headquarters,omitempty"`
	TeamSize     int           `json:"team_size,omitempty"`
	ThreatLevel  int           `json:"threat_level,omitempty"`
	Website      string        `json:"website,omitempty"`
	CompetesWith []*Competitor `json:"competes_with,omitempty"`
	Develops     []*Product    `json:"develops,omitempty"`
}

// Feature 功能实体
type Feature struct {
	UID          string     `json:"uid,omitempty"`
	Availability bool       `json:"availability,omitempty"`
	Category     string     `json:"category,omitempty"`
	Description  string     `json:"description,omitempty"`
	EvidenceURL  string     `json:"evidence_url,omitempty"`
	FeatureName  string     `json:"feature_name,omitempty"`
	LastUpdated  *time.Time `json:"last_updated,omitempty"`
	Maturity     int        `json:"maturity,omitempty"`
	PartOf       *Product   `json:"part_of,omitempty"`
}

// MarketEvent 市场事件实体
type MarketEvent struct {
	UID         string     `json:"uid,omitempty"`
	Confidence  float64    `json:"confidence,omitempty"`
	Description string     `json:"description,omitempty"`
	EventDate   *time.Time `json:"event_date,omitempty"`
	EventType   string     `json:"event_type,omitempty"`
	Impact      string     `json:"impact,omitempty"`
	SourceURL   string     `json:"source_url,omitempty"`
}

// PricingTier 定价实体
type PricingTier struct {
	UID           string     `json:"uid,omitempty"`
	BillingModel  string     `json:"billing_model,omitempty"`
	Currency      string     `json:"currency,omitempty"`
	LastUpdated   *time.Time `json:"last_updated,omitempty"`
	Limitations   string     `json:"limitations,omitempty"`
	Price         float64    `json:"price,omitempty"`
	TargetSegment string     `json:"target_segment,omitempty"`
	TierName      string     `json:"tier_name,omitempty"`
}

// Product 产品实体
type Product struct {
	UID           string         `json:"uid,omitempty"`
	Category      string         `json:"category,omitempty"`
	CoreTech      string         `json:"core_tech,omitempty"`
	Description   string         `json:"description,omitempty"`
	LaunchDate    *time.Time     `json:"launch_date,omitempty"`
	ProductName   string         `json:"product_name,omitempty"`
	ReviewCount   int            `json:"review_count,omitempty"`
	UserRating    float64        `json:"user_rating,omitempty"`
	BelongsTo     *Competitor    `json:"belongs_to,omitempty"`
	HasPricing    []*PricingTier `json:"has_pricing,omitempty"`
	OffersFeature []*Feature     `json:"offers_feature,omitempty"`
}

// TechComponent 技术栈实体
type TechComponent struct {
	UID           string     `json:"uid,omitempty"`
	Architecture  string     `json:"architecture,omitempty"`
	ComponentName string     `json:"component_name,omitempty"`
	GithubStars   int        `json:"github_stars,omitempty"`
	LastCommit    *time.Time `json:"last_commit,omitempty"`
	OpenSource    bool       `json:"open_source,omitempty"`
	Technology    string     `json:"technology,omitempty"`
}

// UserSegment 用户群体实体
type UserSegment struct {
	UID               string   `json:"uid,omitempty"`
	CorePainPoints    []string `json:"core_pain_points,omitempty"`
	Description       string   `json:"description,omitempty"`
	SatisfactionScore float64  `json:"satisfaction_score,omitempty"`
	SegmentName       string   `json:"segment_name,omitempty"`
	TypicalUseCases   []string `json:"typical_use_cases,omitempty"`
}

// AUTOGEN:ONTOLOGY:END

//go:generate go run -tags=generator generator.go
