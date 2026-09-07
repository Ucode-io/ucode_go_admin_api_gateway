package models

type CallQualityTranscriptLine struct {
	Speaker string `json:"speaker" binding:"required,oneof=operator client unknown"`
	Text    string `json:"text" binding:"required"`
	From    int64  `json:"from"`
}

type CallQualityCriterion struct {
	ID            string    `json:"id" binding:"required"`
	Name          string    `json:"name" binding:"required"`
	Instructions  string    `json:"instructions" binding:"required"`
	MaxScore      float64   `json:"max_score" binding:"required,gte=0"`
	AllowedScores []float64 `json:"allowed_scores" binding:"required,min=1"`
}

type CallQualityEvaluateRequest struct {
	CallID           string                      `json:"call_id" binding:"required"`
	Language         string                      `json:"language"`
	TranscriptSource string                      `json:"transcript_source" binding:"required,oneof=stereo mono"`
	Transcript       []CallQualityTranscriptLine `json:"transcript" binding:"required,min=1,max=2000,dive"`
	CriteriaVersion  string                      `json:"criteria_version" binding:"required"`
	Criteria         []CallQualityCriterion      `json:"criteria" binding:"required,min=1,max=30,dive"`
	ViolationRules   string                      `json:"violation_rules"`
}

type CallQualityEvidence struct {
	Speaker string `json:"speaker"`
	Quote   string `json:"quote"`
	From    int64  `json:"from"`
}

type CallQualityCriterionResult struct {
	CriterionID    string                `json:"criterion_id"`
	Name           string                `json:"name"`
	Score          float64               `json:"score"`
	Status         string                `json:"status"`
	Explanation    string                `json:"explanation"`
	Recommendation string                `json:"recommendation,omitempty"`
	Evidence       []CallQualityEvidence `json:"evidence"`
}

type CallQualityViolation struct {
	CriterionID string                `json:"criterion_id"`
	Severity    string                `json:"severity"`
	Explanation string                `json:"explanation"`
	Evidence    []CallQualityEvidence `json:"evidence"`
	Penalty     float64               `json:"penalty"`
}

type CallQualityEvaluateResponse struct {
	CallID          string                       `json:"call_id"`
	CriteriaVersion string                       `json:"criteria_version"`
	Summary         string                       `json:"summary"`
	PositiveScore   float64                      `json:"positive_score"`
	Penalty         float64                      `json:"penalty"`
	TotalScore      float64                      `json:"total_score"`
	MaxScore        float64                      `json:"max_score"`
	ReviewRequired  bool                         `json:"review_required"`
	ReviewReasons   []string                     `json:"review_reasons"`
	Criteria        []CallQualityCriterionResult `json:"criteria"`
	Violations      []CallQualityViolation       `json:"violations"`
}
