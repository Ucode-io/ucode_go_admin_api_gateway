package v2

import "strings"

// normalizeProfessionalCRMLegacyDeal keeps integrations using the original
// Professional CRM contract compatible with the renamed Udevs pipeline.
func normalizeProfessionalCRMLegacyDeal(projectID, collection string, data map[string]any) {
	if projectID != professionalCRMProjectID || collection != professionalCRMDealsCollection || data == nil {
		return
	}

	pipeline, ok := legacyPipelineReplacement(data["pipeline"])
	if !ok {
		return
	}
	data["pipeline"] = pipeline

	stage := scalarString(data[legacyProfessionalCRMStageField])
	if stage == "" {
		stage = scalarString(data["stage"])
	}
	if stage != "" {
		data[professionalCRMStageField] = stage
	}
}

func legacyPipelineReplacement(value any) (any, bool) {
	switch typed := value.(type) {
	case string:
		if strings.TrimSpace(typed) == legacyProfessionalCRMPipeline {
			return professionalCRMPipeline, true
		}
	case []any:
		if len(typed) == 1 && scalarString(typed[0]) == legacyProfessionalCRMPipeline {
			return []any{professionalCRMPipeline}, true
		}
	case []string:
		if len(typed) == 1 && strings.TrimSpace(typed[0]) == legacyProfessionalCRMPipeline {
			return []string{professionalCRMPipeline}, true
		}
	}

	return value, false
}

func scalarString(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case []any:
		if len(typed) == 1 {
			return scalarString(typed[0])
		}
	case []string:
		if len(typed) == 1 {
			return strings.TrimSpace(typed[0])
		}
	}

	return ""
}
