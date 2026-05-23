package models

type (
	RouterInput struct {
		UserMessage   string
		FileGraphJSON string
		HasImages     bool
		History       []ChatMessage
	}

	ArchitectInput struct {
		Clarified         string
		ExistingSchemaCtx string
		Images            []string
		History           []ChatMessage
	}

	ManifestInput struct {
		Plan    *ArchitectPlan
		History []ChatMessage
	}

	PlannerInput struct {
		Clarified     string
		FileGraphJSON string
		HasImages     bool
		History       []ChatMessage
	}

	InspectorInput struct {
		Question     string
		FilesContext string
		Images       []string
		History      []ChatMessage
	}

	EditorInput struct {
		Clarified        string
		Plan             *SonnetPlanResult
		FilesContext     string
		Images           []string
		History          []ChatMessage
		HasMatchingFiles bool
	}

	LLMUsage struct {
		InputTokens  int
		OutputTokens int
	}

	// VisualEditInput carries pre-built messages for the visual edit tool call.
	// The caller (v1) is responsible for building prompt and resolving file contexts.
	VisualEditInput struct {
		Messages []ChatMessage
	}

	// RepairFileInput carries the file to repair and the pre-built user prompt
	// (errors + available exports + rules). The agent applies the repair tool.
	RepairFileInput struct {
		File       ProjectFile
		UserPrompt string
	}

	// DatabaseQueryInput carries data for the database assistant agent call.
	DatabaseQueryInput struct {
		Clarified   string
		SchemaText  string
		DataContext string
		History     []ChatMessage
	}
)
