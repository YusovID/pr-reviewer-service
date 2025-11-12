package http

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/YusovID/pr-reviewer-service/pkg/api"
	"github.com/jmoiron/sqlx"
)

type Server struct {
	db  *sqlx.DB
	log *slog.Logger
	//
}

func NewServer(db *sqlx.DB, log *slog.Logger) *Server {
	return &Server{
		db:  db,
		log: log,
	}
}

func (s *Server) Routes() http.Handler {
	// r := chi.NewRouter()

	return api.Handler(s)
}

// Создать PR и автоматически назначить до 2 ревьюверов из команды автора
// (POST /pullRequest/create)
func (s *Server) PostPullRequestCreate(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Пометить PR как MERGED (идемпотентная операция)
// (POST /pullRequest/merge)
func (s *Server) PostPullRequestMerge(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Переназначить конкретного ревьювера на другого из его команды
// (POST /pullRequest/reassign)
func (s *Server) PostPullRequestReassign(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Создать команду с участниками (создаёт/обновляет пользователей)
// (POST /team/add)
func (s *Server) PostTeamAdd(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Получить команду с участниками
// (GET /team/get)
func (s *Server) GetTeamGet(w http.ResponseWriter, r *http.Request, params api.GetTeamGetParams) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Получить PR'ы, где пользователь назначен ревьювером
// (GET /users/getReview)
func (s *Server) GetUsersGetReview(w http.ResponseWriter, r *http.Request, params api.GetUsersGetReviewParams) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Установить флаг активности пользователя
// (POST /users/setIsActive)
func (s *Server) PostUsersSetIsActive(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func encode[T any](w http.ResponseWriter, status int, v T) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	return nil
}

func decode[T any](r *http.Request) (T, error) {
	var v T
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		return v, fmt.Errorf("decode json: %w", err)
	}
	return v, nil
}
