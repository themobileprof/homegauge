package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ErrorBody struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details any    `json:"details,omitempty"`
}

func JSONError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, ErrorBody{Error: message, Code: code})
}

func BadRequest(c *gin.Context, message string) {
	JSONError(c, http.StatusBadRequest, "bad_request", message)
}

func Unauthorized(c *gin.Context, message string) {
	JSONError(c, http.StatusUnauthorized, "unauthorized", message)
}

func Forbidden(c *gin.Context, message string) {
	JSONError(c, http.StatusForbidden, "forbidden", message)
}

func NotFound(c *gin.Context, message string) {
	JSONError(c, http.StatusNotFound, "not_found", message)
}

func Conflict(c *gin.Context, message string) {
	JSONError(c, http.StatusConflict, "conflict", message)
}

func Internal(c *gin.Context, message string) {
	JSONError(c, http.StatusInternalServerError, "internal_error", message)
}
