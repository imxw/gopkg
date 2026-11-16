package ginx

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ParseIDParam extracts an int64 path parameter, sending a 400 envelope on
// failure. invalidMsg is caller-owned text (localized freely); (0,false)
// means the response is already written — handlers just return.
func ParseIDParam(c *gin.Context, param, invalidMsg string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(param), 10, 64)
	if err != nil {
		BadRequest(c, invalidMsg)
		return 0, false
	}
	return id, true
}

// ParseIDQuery extracts an optional int64 query parameter; empty or absent
// returns (0, true) — callers treat 0 as the "no filter" sentinel. Negative
// values are rejected (400).
func ParseIDQuery(c *gin.Context, param, invalidMsg string) (int64, bool) {
	raw := c.Query(param)
	if raw == "" {
		return 0, true
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id < 0 {
		BadRequest(c, invalidMsg)
		return 0, false
	}
	return id, true
}

// ParseUUIDParam extracts a UUID path parameter, sending a 400 envelope on
// failure. uuid.Nil with false = response already written.
func ParseUUIDParam(c *gin.Context, param, invalidMsg string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(param))
	if err != nil {
		BadRequest(c, invalidMsg)
		return uuid.Nil, false
	}
	return id, true
}
