package azure

type errorResponse struct {
	Error azureError `json:"error"`
}

type azureError struct {
	Message string `json:"message"`
	Code    any    `json:"code"`
}

type deploymentsResponse struct {
	Data []deploymentResponse `json:"data"`
}

type deploymentResponse struct {
	ID        string `json:"id"`
	Object    string `json:"object"`
	CreatedAt int64  `json:"created_at"`
	Model     string `json:"model"`
}
