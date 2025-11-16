package http

type createTeamRequest struct {
	TeamName string `json:"team_name" validate:"required,min=3,max=50"`
	Members  []struct {
		UserID   string `json:"user_id" validate:"required,custom_id,min=1,max=100"`
		Username string `json:"username" validate:"required,min=2,max=100"`
		IsActive bool   `json:"is_active"`
	} `json:"members" validate:"omitempty,dive"`
}

type createPRRequest struct {
	PullRequestID   string `json:"pull_request_id" validate:"required,custom_id,min=1,max=100"`
	PullRequestName string `json:"pull_request_name" validate:"required,min=5,max=255"`
	AuthorID        string `json:"author_id" validate:"required,custom_id,min=1,max=100"`
}

type setUserActiveRequest struct {
	UserID   string `json:"user_id" validate:"required,custom_id,min=1,max=100"`
	IsActive bool   `json:"is_active"`
}

type mergePRRequest struct {
	PullRequestID string `json:"pull_request_id" validate:"required,custom_id,min=1,max=100"`
}

type reassignRequest struct {
	PullRequestID string `json:"pull_request_id" validate:"required,custom_id,min=1,max=100"`
	OldUserID     string `json:"old_user_id" validate:"required,custom_id,min=1,max=100"`
}
