package models

type (
	GeneratePromptMCP struct {
		ProjectId     string `json:"project_id"`
		EnvironmentId string `json:"environment_id"`
		APIKey        string `json:"api_key"`
		UserPrompt    string `json:"user_prompt"`
		Method        string `json:"method"`
		BaseURL       string `json:"base_url"`
	}

	AnalysisRequest struct {
		UserRequest string         `json:"user_request"`
		FileGraph   map[string]any `json:"file_graph"`
		ProjectName string         `json:"project_name"`
	}

	UpdateRequest struct {
		UserRequest    string           `json:"user_request"`
		FilesToUpdate  []FileContent    `json:"files_to_update"`
		AnalysisResult AnalysisResponse `json:"analysis_result"`
		ProjectName    string           `json:"project_name"`
	}

	FileContent struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}

	AnalysisResponse struct {
		AnalysisSummary      string         `json:"analysis_summary"`
		FilesToModify        []FileToModify `json:"files_to_modify"`
		NewFilesNeeded       []FileToCreate `json:"new_files_needed"`
		FilesToDelete        []FileToDelete `json:"files_to_delete"`
		AffectedDependencies []string       `json:"affected_dependencies"`
		EstimatedComplexity  string         `json:"estimated_complexity"`
		Risks                []string       `json:"risks"`
	}

	FileToModify struct {
		Path       string `json:"path"`
		Reason     string `json:"reason"`
		ChangeType string `json:"change_type"`
		Priority   string `json:"priority"`
	}

	FileToCreate struct {
		Path       string `json:"path"`
		Reason     string `json:"reason"`
		ChangeType string `json:"change_type"`
	}

	FileToDelete struct {
		Path   string `json:"path"`
		Reason string `json:"reason"`
	}
)
