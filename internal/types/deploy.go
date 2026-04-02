package types

type DeployResponse struct {
	Message string `json:"message"`
}

type DockerMessage struct {
	Stream string `json:"stream"`
	Status string `json:"status"`
	ID     string `json:"id"`
	Error  string `json:"error"`

	Progress string `json:"progress"`
}
