package deployer

import "time"

type DeployRequest struct {
	RepoURL string `json:"repo_url" binding:"required,url"`
}

type DeploymentResponse struct {
	ID          string    `json:"id"`
	RepoURL     string    `json:"repo_url"`
	Status      string    `json:"status"`
	DeployedURL string    `json:"deployed_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
