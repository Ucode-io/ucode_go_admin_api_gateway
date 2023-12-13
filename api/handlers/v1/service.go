package v1

import (
	"ucode/ucode_go_api_gateway/services"
)

func (h *HandlerV1) GetService(namespace string) (services.ServiceManagerI, error) {
	return h.services.Get(namespace)
}

func (h *HandlerV1) RemoveService(namespace string) error {
	return h.services.Remove(namespace)
}

func (h *HandlerV1) IsServiceExists(namespace string) bool {
	_, err := h.services.Get(namespace)
	if err != nil {
		return false
	}

	return true
}
