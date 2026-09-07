package callquality

const (
	evaluationToolName = "submit_call_quality_evaluation"
	statusCompleted    = "completed"
	statusProtected    = "protected"
	statusUncertain    = "uncertain"
	severityModerate   = "moderate"
	severityMajor      = "major"
	maxPenaltyPerBlock = 2.0
	maxTotalPenalty    = 10.0
	maxTranscriptChars = 120000
	maxConfiguredScore = 100.0
)
