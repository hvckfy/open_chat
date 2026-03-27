package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ServiceResponse struct {
	Response string               `json:"response"`
	Err      ServiceResponseError `json:"error"`
}

type ServiceResponseError struct {
	Exists  string `json:"error_exists"`
	Message string `json:"error_message"`
}

func RespondError(c *gin.Context, status int, message string) {
	resp := ServiceResponse{
		Err: ServiceResponseError{
			Exists:  "true",
			Message: message,
		},
	}

	c.JSON(status, resp)
}

func RespondSuccess(c *gin.Context, status int, body interface{}) {

	jsonBody, err := json.Marshal(body)
	if err != nil {
		resp := ServiceResponse{
			Err: ServiceResponseError{
				Exists:  "true",
				Message: "Internal Error",
			},
		}

		c.JSON(http.StatusInternalServerError, resp)
		return
	}

	resp := ServiceResponse{
		Response: string(jsonBody),
		Err: ServiceResponseError{
			Exists: "false",
		},
	}

	c.JSON(status, resp)
}
