package response

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func decodeResponseBody(t *testing.T, body []byte) map[string]interface{} {
	t.Helper()
	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	return resp
}

func TestRespSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	RespSuccess(c, gin.H{"x": "y"})
	resp := decodeResponseBody(t, w.Body.Bytes())
	if int(resp["code"].(float64)) != 200 {
		t.Fatalf("unexpected code: %+v", resp)
	}
}

func TestRespError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	RespError(c, 500, "err")
	resp := decodeResponseBody(t, w.Body.Bytes())
	if int(resp["code"].(float64)) != 500 {
		t.Fatalf("unexpected code: %+v", resp)
	}
	if resp["msg"] != "err" {
		t.Fatalf("unexpected msg: %+v", resp)
	}
}
