package http

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/emzola/issuetracker/internal/model"
	"github.com/emzola/issuetracker/internal/service"
)

type tokenService interface {
	CreateActivationToken(ctx context.Context, user *model.User) error
	CreateAuthenticationToken(ctx context.Context, email, password string) ([]byte, error)
}

// CreateActivationToken godoc
// @Summary Create a new activation token
// @Description This endpoint creates a new activation token
// @Tags tokens
// @Accept  json
// @Produce json
// @Param payload body createActivationTokenPayload true "Request payload"
// @Success 200
// @Failure 400
// @Failure 422
// @Failure 500
// @Router /v1/tokens/activation [post]
func (h *Handler) createActivationToken(w http.ResponseWriter, r *http.Request) {
	var requestPayload createActivationTokenPayload
	err := h.decodeJSON(w, r, &requestPayload)
	if err != nil {
		h.badRequestResponse(w, r, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	user, err := h.service.GetUserByEmail(ctx, requestPayload.Email)
	if err != nil {
		switch {
		case errors.Is(err, context.Canceled):
			return
		case errors.Is(err, service.ErrFailedValidation):
			h.failedValidationResponse(w, r, err)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	err = h.service.CreateActivationToken(ctx, user)
	if err != nil {
		switch {
		case errors.Is(err, context.Canceled):
			return
		case errors.Is(err, service.ErrActivated):
			h.alreadyActivatedResponse(w, r)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	err = h.encodeJSON(w, http.StatusOK, envelop{"message": "an email will be sent to you containing activation instructions"}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// CreateAuthenticationToken godoc
// @Summary Create JWT authentication token
// @Description This endpoint creates JWT token
// @Tags tokens
// @Accept  json
// @Produce json
// @Param payload body createAuthenticationTokenPayload true "Request payload"
// @Success 201 {object} model.Token
// @Failure 400
// @Failure 401
// @Failure 422
// @Failure 500
// @Router /v1/tokens/authentication [post]
func (h *Handler) createAuthenticationToken(w http.ResponseWriter, r *http.Request) {
	var requestPayload createAuthenticationTokenPayload
	err := h.decodeJSON(w, r, &requestPayload)
	if err != nil {
		h.badRequestResponse(w, r, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	jwtBytes, err := h.service.CreateAuthenticationToken(ctx, requestPayload.Email, requestPayload.Password)
	if err != nil {
		switch {
		case errors.Is(err, context.Canceled):
			return
		case errors.Is(err, service.ErrFailedValidation):
			h.failedValidationResponse(w, r, err)
		case errors.Is(err, service.ErrInvalidCredentials):
			h.invalidCredentialsResponse(w, r)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	err = h.encodeJSON(w, http.StatusCreated, envelop{"authentication_token": string(jwtBytes)}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}
