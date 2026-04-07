package admin

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
)

type adminSwagger struct {
	Paths map[string]map[string]interface{} `json:"paths"`
}

func TestSwaggerDeclaredAPIsAreReachable(t *testing.T) {
	mockAuthAndSources(t)
	router := buildAdminRouter()

	content, err := os.ReadFile("../docs/API/Admin/Admin_swagger.json")
	if err != nil {
		t.Fatalf("read swagger file failed: %v", err)
	}

	var doc adminSwagger
	if err := json.Unmarshal(content, &doc); err != nil {
		t.Fatalf("unmarshal swagger failed: %v", err)
	}
	if len(doc.Paths) == 0 {
		t.Fatal("swagger paths should not be empty")
	}

	requiredPaths := []string{
		"/admin/works/{work_id}",
		"/admin/works/{work_id}/file",
		"/admin/works/track/{track_id}",
		"/admin/works/author/{author_id}",
	}
	for _, required := range requiredPaths {
		if _, ok := doc.Paths[required]; !ok {
			t.Fatalf("swagger missing required path: %s", required)
		}
	}

	replacer := strings.NewReplacer(
		"{contest_id}", "1",
		"{track_id}", "1",
		"{work_id}", "1",
		"{author_id}", "1",
	)

	for path, methods := range doc.Paths {
		for method := range methods {
			httpMethod := strings.ToUpper(method)
			requestPath := "/api/v1" + replacer.Replace(path)

			var body []byte
			auth := "Bearer admin"

			switch {
			case requestPath == "/api/v1/admin/login":
				auth = ""
				body = []byte(`{"adminName":"a","password":"b"}`)
			case requestPath == "/api/v1/admin/refresh":
				auth = "Bearer refresh-admin"
			case strings.Contains(requestPath, "/api/v1/admin/contest") && (httpMethod == http.MethodPost || httpMethod == http.MethodPut):
				body = []byte(`{"contestName":"c"}`)
			case strings.Contains(requestPath, "/api/v1/admin/track") && (httpMethod == http.MethodPost || httpMethod == http.MethodPut):
				body = []byte(`{"trackName":"t"}`)
			}

			w := doRequest(router, httpMethod, requestPath, body, auth)
			if w.Code != http.StatusOK {
				t.Fatalf("unexpected http status for %s %s: %d", httpMethod, requestPath, w.Code)
			}

			if strings.Contains(requestPath, "/file") && httpMethod == http.MethodGet {
				if len(w.Body.Bytes()) == 0 {
					t.Fatalf("expected file response for %s %s", httpMethod, requestPath)
				}
				continue
			}

			if got := reqCode(t, w.Body.Bytes()); got == 404 {
				t.Fatalf("swagger path is not reachable: %s %s", httpMethod, requestPath)
			}
		}
	}
}
