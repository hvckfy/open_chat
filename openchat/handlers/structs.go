package handlers

type ServiceResponse struct {
	Response []byte               `json:"response"`
	Err      ServiceResponseError `json:"error"`
}

type ServiceResponseError struct {
	Exists  string `json:"error_exists"`
	Message string `json:"error_message"`
}
