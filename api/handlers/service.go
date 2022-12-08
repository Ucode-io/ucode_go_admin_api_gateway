package handlers

import (
	"ucode/ucode_go_api_gateway/services"
)

func (h *Handler) GetService(namespace string) (services.ServiceManagerI, error) {
	return h.services.Get(namespace)
}

func (h *Handler) RemoveService(namespace string) error {
	return h.services.Remove(namespace)
}

func (h *Handler) IsServiceExists(namespace string) bool {
	_, err := h.services.Get(namespace)
	if err != nil {
		return false
	}

	return true
}
