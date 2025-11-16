package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/YusovID/pr-reviewer-service/internal/apperrors"
	"github.com/YusovID/pr-reviewer-service/internal/service"
	"github.com/YusovID/pr-reviewer-service/internal/validation"
	"github.com/YusovID/pr-reviewer-service/pkg/api"
	"github.com/YusovID/pr-reviewer-service/pkg/logger/sl"
	"github.com/YusovID/pr-reviewer-service/swagger"
	"github.com/go-chi/chi/v5"
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
	mux := chi.NewRouter()

	mux.Use(s.requestID)
	mux.Use(s.logRequest)

	swaggerHandler, err := swagger.GetHandler()
	if err != nil {
		s.log.Error("failed to get swagger handler", sl.Err(err))
	} else {
		mux.Mount("/swagger", http.StripPrefix("/swagger", swaggerHandler))
	}

	mux.Mount("/", api.Handler(s))

	return mux
}

func (s *Server) PostTeamAdd(w http.ResponseWriter, r *http.Request) {
	const op = "internal.transport.http.PostTeamAdd"

	var req createTeamRequest
	if err := s.decodeAndValidate(r, &req); err != nil {
		s.handleServiceError(w, r, op, err)
		return
	}

	apiMembers := make([]api.TeamMember, len(req.Members))
	for i, m := range req.Members {
		apiMembers[i] = api.TeamMember{
			UserId:   m.UserID,
			Username: m.Username,
			IsActive: m.IsActive,
		}
	}

	apiTeam := api.Team{
		TeamName: req.TeamName,
		Members:  apiMembers,
	}

	team, err := s.teamService.CreateTeamWithUsers(r.Context(), apiTeam)
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

	var req setUserActiveRequest
	if err := s.decodeAndValidate(r, &req); err != nil {
		s.handleServiceError(w, r, op, err)
		return
	}

	user, err := s.userService.SetIsActive(r.Context(), req.UserID, req.IsActive)
	if err != nil {
		s.handleServiceError(w, r, op, err)
		return
	}

	s.respond(w, http.StatusOK, map[string]*api.User{"user": user})
}

func (s *Server) PostPullRequestCreate(w http.ResponseWriter, r *http.Request) {
	const op = "internal.transport.http.PostPullRequestCreate"

	var req createPRRequest
	if err := s.decodeAndValidate(r, &req); err != nil {
		s.handleServiceError(w, r, op, err)
		return
	}

	pr, err := s.prService.CreatePR(r.Context(), req.PullRequestID, req.PullRequestName, req.AuthorID)
	if err != nil {
		s.handleServiceError(w, r, op, err)
		return
	}

	s.respond(w, http.StatusCreated, map[string]*api.PullRequest{"pr": pr})
}

func (s *Server) PostPullRequestMerge(w http.ResponseWriter, r *http.Request) {
	const op = "internal.transport.http.PostPullRequestMerge"

	var req mergePRRequest
	if err := s.decodeAndValidate(r, &req); err != nil {
		s.handleServiceError(w, r, op, err)
		return
	}

	pr, err := s.prService.MergePR(r.Context(), req.PullRequestID)
	if err != nil {
		s.handleServiceError(w, r, op, err)
		return
	}

	s.respond(w, http.StatusOK, map[string]*api.PullRequest{"pr": pr})
}

func (s *Server) PostPullRequestReassign(w http.ResponseWriter, r *http.Request) {
	const op = "internal.transport.http.PostPullRequestReassign"

	var req reassignRequest
	if err := s.decodeAndValidate(r, &req); err != nil {
		s.handleServiceError(w, r, op, err)
		return
	}

	resp, err := s.prService.ReassignReviewer(r.Context(), req.PullRequestID, req.OldUserID)
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

func (s *Server) GetStats(w http.ResponseWriter, r *http.Request) {
	const op = "internal.transport.http.GetStats"

	stats, err := s.prService.GetStats(r.Context())
	if err != nil {
		s.handleServiceError(w, r, op, err)
		return
	}

	s.respond(w, http.StatusOK, stats)
}

func (s *Server) PostTeamDeactivate(w http.ResponseWriter, r *http.Request) {
	const op = "internal.transport.http.PostTeamDeactivate"

	var req deactivateTeamRequest
	if err := s.decodeAndValidate(r, &req); err != nil {
		s.handleServiceError(w, r, op, err)
		return
	}

	deactivatedCount, reassignedCount, err := s.userService.DeactivateTeam(r.Context(), req.TeamName)
	if err != nil {
		s.handleServiceError(w, r, op, err)
		return
	}

	s.respond(w, http.StatusOK, map[string]int{
		"deactivated_users_count": deactivatedCount,
		"reassigned_prs_count":    reassignedCount,
	})
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

func (s *Server) decodeAndValidate(r *http.Request, v interface{}) error {
	if err := s.decode(r.Body, v); err != nil {
		return err
	}

	if err := validation.ValidateStruct(v); err != nil {
		return err
	}

	return nil
}

func (s *Server) decode(body io.ReadCloser, v interface{}) error {
	defer body.Close()

	if err := json.NewDecoder(body).Decode(v); err != nil {
		return fmt.Errorf("%w: %w", apperrors.ErrInvalidRequest, err)
	}

	return nil
}

func (s *Server) handleServiceError(w http.ResponseWriter, _ *http.Request, op string, err error) {
	log := s.log.With(slog.String("op", op))
	log.Error("service error occurred", sl.Err(err))

	var (
		teamExistsErr *apperrors.TeamAlreadyExistsError
		prExistsErr   *apperrors.PRAlreadyExistsError
		validationErr *validation.ValidationError
	)

	switch {
	case errors.As(err, &validationErr):
		wrappedErr := fmt.Errorf("%w: %s", apperrors.ErrValidation, validationErr.Error())
		s.respondError(w, http.StatusBadRequest, wrappedErr.Error())
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
