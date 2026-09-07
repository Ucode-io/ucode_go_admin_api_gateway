package callquality

import (
	"fmt"
	"strings"

	"ucode/ucode_go_api_gateway/api/models"
)

func validateRequest(req models.CallQualityEvaluateRequest) error {
	characters := 0
	for _, line := range req.Transcript {
		characters += len([]rune(line.Text))
		if characters > maxTranscriptChars {
			return fmt.Errorf("transcript exceeds %d characters", maxTranscriptChars)
		}
	}

	ids := make(map[string]struct{}, len(req.Criteria))
	for _, criterion := range req.Criteria {
		id := strings.TrimSpace(criterion.ID)
		if id == "" || id != criterion.ID || strings.TrimSpace(criterion.Name) == "" || strings.TrimSpace(criterion.Instructions) == "" {
			return fmt.Errorf("criterion id, name and instructions are required")
		}
		if _, exists := ids[id]; exists {
			return fmt.Errorf("duplicate criterion id: %s", id)
		}
		ids[id] = struct{}{}
		if criterion.MaxScore <= 0 || criterion.MaxScore > maxConfiguredScore {
			return fmt.Errorf("criterion %s max_score must be between 0 and %.0f", id, maxConfiguredScore)
		}
		includesMaximum := false
		for _, score := range criterion.AllowedScores {
			if score < 0 || score > criterion.MaxScore {
				return fmt.Errorf("criterion %s contains an out-of-range score", id)
			}
			if scoreAllowed(score, []float64{criterion.MaxScore}) {
				includesMaximum = true
			}
		}
		if !includesMaximum {
			return fmt.Errorf("criterion %s allowed_scores must include max_score", id)
		}
	}
	return nil
}
