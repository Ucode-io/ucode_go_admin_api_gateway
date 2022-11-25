package handlers

import (
	"errors"
	"ucode/ucode_go_api_gateway/services"
)

func (h *Handler) GetService(namespace string) (services.ServiceManagerI, error) {
	h.services.Mu.Lock()
	defer h.services.Mu.Unlock()

	services, ok := h.services.Services[namespace]
	if !ok {
		return nil, errors.New("error while getting nil service:" + namespace)
	}
	return services, nil
}

func (h *Handler) RemoveService(namespace string) {
	h.services.Mu.Lock()
	defer h.services.Mu.Unlock()

	delete(h.services.Services, namespace)
}

func (h *Handler) IsServiceExists(namespace string) bool {
	h.services.Mu.Lock()
	defer h.services.Mu.Unlock()

	_, ok := h.services.Services[namespace]
	return ok
}
