package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/YusovID/pr-reviewer-service/internal/apperrors"
	"github.com/YusovID/pr-reviewer-service/internal/service"
	"github.com/YusovID/pr-reviewer-service/pkg/api"
	"github.com/YusovID/pr-reviewer-service/pkg/logger/sl"
)

type Server struct {
	log         *slog.Logger
	teamService service.TeamService
	userService service.UserService
	prService   service.PullRequestService
}

func NewServer(
	log *slog.Logger,
	ts service.TeamService,
	us service.UserService,
	prs service.PullRequestService,
) *Server {
	return &Server{
		log:         log,
		teamService: ts,
		userService: us,
		prService:   prs,
	}
}

func (s *Server) Routes() http.Handler {
	router := api.Handler(s)
	return s.logRequest(router)
}

func (s *Server) PostTeamAdd(w http.ResponseWriter, r *http.Request) {
	const op = "internal.transport.http.PostTeamAdd"

	var req api.Team
	if err := s.decode(r, &req); err != nil {
		s.handleServiceError(w, r, op, err)
		return
	}

	team, err := s.teamService.CreateTeam(r.Context(), req)
	if err != nil {
		s.handleServiceError(w, r, op, err)
		return
	}

	s.respond(w, http.StatusCreated, map[string]*api.Team{"team": team})
}

func (s *Server) GetTeamGet(w http.ResponseWriter, r *http.Request, params api.GetTeamGetParams) {
	const op = "internal.transport.http.GetTeamGet"

	team, err := s.teamService.GetTeam(r.Context(), params.TeamName)
	if err != nil {
		s.handleServiceError(w, r, op, err)
		return
	}

	s.respond(w, http.StatusOK, map[string]*api.Team{"team": team})
}

func (s *Server) PostUsersSetIsActive(w http.ResponseWriter, r *http.Request) {
	const op = "internal.transport.http.PostUsersSetIsActive"

	var req api.PostUsersSetIsActiveJSONBody
	if err := s.decode(r, &req); err != nil {
		s.handleServiceError(w, r, op, err)
		return
	}

	user, err := s.userService.SetIsActive(r.Context(), req.UserId, req.IsActive)
	if err != nil {
		s.handleServiceError(w, r, op, err)
		return
	}

	s.respond(w, http.StatusOK, map[string]*api.User{"user": user})
}

func (s *Server) PostPullRequestCreate(w http.ResponseWriter, r *http.Request) {
	const op = "internal.transport.http.PostPullRequestCreate"

	var req api.PostPullRequestCreateJSONBody
	if err := s.decode(r, &req); err != nil {
		s.handleServiceError(w, r, op, err)
		return
	}

	pr, err := s.prService.CreatePR(r.Context(), req.PullRequestId, req.PullRequestName, req.AuthorId)
	if err != nil {
		s.handleServiceError(w, r, op, err)
		return
	}

	s.respond(w, http.StatusCreated, map[string]*api.PullRequest{"pr": pr})
}

func (s *Server) PostPullRequestMerge(w http.ResponseWriter, r *http.Request) {
	const op = "internal.transport.http.PostPullRequestMerge"

	var req api.PostPullRequestMergeJSONBody
	if err := s.decode(r, &req); err != nil {
		s.handleServiceError(w, r, op, err)
		return
	}

	pr, err := s.prService.MergePR(r.Context(), req.PullRequestId)
	if err != nil {
		s.handleServiceError(w, r, op, err)
		return
	}

	s.respond(w, http.StatusOK, map[string]*api.PullRequest{"pr": pr})
}

func (s *Server) PostPullRequestReassign(w http.ResponseWriter, r *http.Request) {
	const op = "internal.transport.http.PostPullRequestReassign"

	var req api.PostPullRequestReassignJSONBody
	if err := s.decode(r, &req); err != nil {
		s.handleServiceError(w, r, op, err)
		return
	}

	resp, err := s.prService.ReassignReviewer(r.Context(), req.PullRequestId, req.OldUserId)
	if err != nil {
		s.handleServiceError(w, r, op, err)
		return
	}

	s.respond(w, http.StatusOK, resp)
}

func (s *Server) GetUsersGetReview(w http.ResponseWriter, r *http.Request, params api.GetUsersGetReviewParams) {
	const op = "internal.transport.http.GetUsersGetReview"

	resp, err := s.prService.GetReviewAssignments(r.Context(), params.UserId)
	if err != nil {
		s.handleServiceError(w, r, op, err)
		return
	}

	s.respond(w, http.StatusOK, resp)
}

func (s *Server) respond(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)

	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			s.log.Error("failed to encode response", sl.Err(err))
		}
	}
}

func (s *Server) respondError(w http.ResponseWriter, code int, message string) {
	s.respond(w, code, map[string]string{"error": message})
}

func (s *Server) respondAPIError(w http.ResponseWriter, code int, apiCode api.ErrorResponseErrorCode, message string) {
	errResp := api.ErrorResponse{
		Error: struct {
			Code    api.ErrorResponseErrorCode `json:"code"`
			Message string                     `json:"message"`
		}{
			Code:    apiCode,
			Message: message,
		},
	}
	s.respond(w, code, errResp)
}

func (s *Server) decode(r *http.Request, v interface{}) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return fmt.Errorf("%w: %w", apperrors.ErrInvalidRequest, err)
	}

	return nil
}

func (s *Server) handleServiceError(w http.ResponseWriter, _ *http.Request, op string, err error) {
	log := s.log.With(slog.String("op", op))
	log.Error("service error occurred", sl.Err(err))

	var teamExistsErr *apperrors.TeamAlreadyExistsError
	var prExistsErr *apperrors.PRAlreadyExistsError

	switch {
	case errors.Is(err, apperrors.ErrInvalidRequest):
		s.respondError(w, http.StatusBadRequest, "invalid request body")
	case errors.Is(err, apperrors.ErrNotFound):
		s.respondAPIError(w, http.StatusNotFound, api.NOTFOUND, "resource not found")
	case errors.As(err, &teamExistsErr):
		s.respondAPIError(w, http.StatusConflict, api.TEAMEXISTS, "team with this name already exists")
	case errors.As(err, &prExistsErr):
		s.respondAPIError(w, http.StatusConflict, api.PREXISTS, "pull request with this id already exists")
	case errors.Is(err, apperrors.ErrPRMerged):
		s.respondAPIError(w, http.StatusConflict, api.PRMERGED, apperrors.ErrPRMerged.Error())
	case errors.Is(err, apperrors.ErrReviewerNotAssigned):
		s.respondAPIError(w, http.StatusConflict, api.NOTASSIGNED, apperrors.ErrReviewerNotAssigned.Error())
	case errors.Is(err, apperrors.ErrNoCandidate):
		s.respondAPIError(w, http.StatusConflict, api.NOCANDIDATE, apperrors.ErrNoCandidate.Error())
	default:
		s.respondError(w, http.StatusInternalServerError, "internal server error")
	}
}
