package v1

import chat_prompts2 "ucode/ucode_go_api_gateway/api/handlers/ai/chat_prompts"

func capacitorPromptAddendum() string {
	return chat_prompts2.PromptCapacitorMobileAddendum
}

func usesWebAppGenerator(projectType string) bool {
	return projectType == "webapp" || projectType == mobileProjectType
}
