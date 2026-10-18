package dto

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newTestContext(url string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", url, nil)
	return c
}

func TestParseListQuery_LimitAboveMax_ClampsToMax(t *testing.T) {
	q := ParseListQuery(newTestContext("/?limit=200"))

	if q.Limit != 100 {
		t.Fatalf("expected limit above 100 to clamp down to the max (100), got %d", q.Limit)
	}
}

func TestParseListQuery_LimitWithinRange_KeptAsIs(t *testing.T) {
	q := ParseListQuery(newTestContext("/?limit=50"))

	if q.Limit != 50 {
		t.Fatalf("expected a valid in-range limit to pass through unchanged, got %d", q.Limit)
	}
}

func TestParseListQuery_LimitOmitted_DefaultsToTen(t *testing.T) {
	q := ParseListQuery(newTestContext("/"))

	if q.Limit != 10 {
		t.Fatalf("expected default limit of 10 when omitted, got %d", q.Limit)
	}
}
