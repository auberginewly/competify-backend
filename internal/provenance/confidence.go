package provenance

// ConfidenceFactors are the 4 dimensions feeding the weighted score.
// See docs/provenance.md for the formula and weight rationale.
type ConfidenceFactors struct {
	SourceScore float64 // 数据源可信度（域名权威度 + 历史准确率 + 时效性）
	CleanScore  float64 // 清洗质量（去重覆盖率 + 字段完整率 + 格式合规率）
	LLMScore    float64 // LLM 一致性（多采样一致性 + 与已知事实吻合度）
	ReviewScore float64 // 审查通过度（Cross-Reviewer 判定 + 辩论轮次 + 人工介入情况）
}

// Weights configures the 4-dimension weighted average. Defaults per design doc.
type Weights struct {
	Source float64
	Clean  float64
	LLM    float64
	Review float64
}

// DefaultWeights matches the design doc: 0.30 / 0.25 / 0.25 / 0.20
var DefaultWeights = Weights{Source: 0.30, Clean: 0.25, LLM: 0.25, Review: 0.20}

// CalculateConfidence computes C = w₁·c_source + w₂·c_clean + w₃·c_llm + w₄·c_review,
// clamped to [0, 1].
func CalculateConfidence(f ConfidenceFactors, w ...Weights) float64 {
	weights := DefaultWeights
	if len(w) > 0 {
		weights = w[0]
	}
	c := weights.Source*f.SourceScore +
		weights.Clean*f.CleanScore +
		weights.LLM*f.LLMScore +
		weights.Review*f.ReviewScore
	if c > 1.0 {
		c = 1.0
	}
	if c < 0.0 {
		c = 0.0
	}
	return c
}

// Label maps a score to a human-readable bucket. < 0.6 → SUSPICIOUS triggers human review.
func Label(score float64) string {
	switch {
	case score >= 0.9:
		return "HIGH"
	case score >= 0.7:
		return "MEDIUM"
	case score >= 0.6:
		return "LOW"
	default:
		return "SUSPICIOUS"
	}
}

// Anchored binds a confidence score to its Merkle Root, so the score itself is auditable.
type Anchored struct {
	Confidence   float64           `json:"confidence"`
	Label        string            `json:"label"`
	MerkleRoot   string            `json:"merkle_root"`
	ProvenanceID string            `json:"provenance_id"`
	Factors      ConfidenceFactors `json:"factors"`
}

// AnchorConfidence packages a confidence score with its Merkle anchor.
func AnchorConfidence(f ConfidenceFactors, merkleRoot, provenanceID string) *Anchored {
	score := CalculateConfidence(f)
	return &Anchored{
		Confidence:   score,
		Label:        Label(score),
		MerkleRoot:   merkleRoot,
		ProvenanceID: provenanceID,
		Factors:      f,
	}
}
