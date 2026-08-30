package application

import (
	"context"
	"net/http"

	"github.com/nickemma/tessera/internal/modules/inference/domain"
	"github.com/nickemma/tessera/internal/modules/inference/ports"
	"github.com/nickemma/tessera/internal/platform/apperrors"
)

type Service struct {
	provider ports.Provider
}

func NewService(provider ports.Provider) *Service {
	return &Service{provider: provider}
}

func (s *Service) Complete(ctx context.Context, request domain.Request, emit func(string) error) (domain.Response, error) {
	if err := request.Validate(); err != nil {
		return domain.Response{}, apperrors.New("invalid_request", http.StatusUnprocessableEntity, err.Error(), err)
	}

	response, err := s.provider.Complete(ctx, request, emit)
	if err != nil {
		return domain.Response{}, err
	}
	return response, nil
}
