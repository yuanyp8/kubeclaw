package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func parseIDParam(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "path parameter is not a valid numeric id")
		return 0, false
	}
	return id, true
}
