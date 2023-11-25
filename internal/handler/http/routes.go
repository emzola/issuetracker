package http

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (h *Handler) Routes() http.Handler {
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(h.notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(h.methodNotAllowedResponse)

	router.HandlerFunc(http.MethodGet, "/v1/projects", h.requireActivatedUser(h.getAllProjects))
	router.HandlerFunc(http.MethodPost, "/v1/projects", h.requireActivatedUser(h.createProject))
	router.HandlerFunc(http.MethodGet, "/v1/projects/:project_id", h.requireActivatedUser(h.getProject))
	router.HandlerFunc(http.MethodPatch, "/v1/projects/:project_id", h.requireActivatedUser(h.updateProject))
	router.HandlerFunc(http.MethodDelete, "/v1/projects/:project_id", h.requireActivatedUser(h.deleteProject))

	router.HandlerFunc(http.MethodPost, "/v1/users", h.createUser)
	router.HandlerFunc(http.MethodPut, "/v1/users/activated", h.activateUser)

	router.HandlerFunc(http.MethodPost, "/v1/tokens/activation", h.createActivationToken)
	router.HandlerFunc(http.MethodPost, "/v1/tokens/authentication", h.createAuthenticationToken)

	return h.authenticate(router)
}
