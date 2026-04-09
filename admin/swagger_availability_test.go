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
		"/admin/authors",
		"/admin/authors/{author_id}",
		"/admin/scripts",
		"/admin/scripts/{script_id}",
		"/admin/scripts/{script_id}/status",
		"/admin/scripts/{script_id}/versions",
		"/admin/scripts/{script_id}/versions/upload",
		"/admin/scripts/{script_id}/versions/{version_id}/activate",
		"/admin/script-flows",
		"/admin/script-flows/{flow_id}",
		"/admin/script-flows/{flow_id}/status",
		"/admin/script-flows/{flow_id}/steps",
		"/admin/script-flows/{flow_id}/mounts",
		"/admin/script-flows/mounts",
		"/admin/script-flows/mounts/{mount_id}",
		"/admin/works",
		"/admin/works/{work_id}",
		"/admin/works/{work_id}/file",
		"/admin/sub-admins",
		"/admin/sub-admins/batch",
		"/admin/sub-admins/{admin_id}",
		"/admin/sub-admins/{admin_id}/permissions",
		"/admin/sub-admins/{admin_id}/disable",
		"/admin/sub-admins/handover-super",
	}
	for _, required := range requiredPaths {
		if _, ok := doc.Paths[required]; !ok {
			t.Fatalf("swagger missing required path: %s", required)
		}
	}

	replacer := strings.NewReplacer(
		"{contest_id}", "1",
		"{track_id}", "1",
		"{author_id}", "1",
		"{script_id}", "1",
		"{version_id}", "1",
		"{flow_id}", "1",
		"{mount_id}", "1",
		"{work_id}", "1",
		"{admin_id}", "1",
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
			case strings.Contains(requestPath, "/api/v1/admin/authors/") && httpMethod == http.MethodPut:
				body = []byte(`{"authorName":"a","penName":"p","authorEmail":"a@example.com"}`)
			case requestPath == "/api/v1/admin/sub-admins" && httpMethod == http.MethodPost:
				body = []byte(`{"adminEmail":"sub@example.com","permissionNames":["works.read"]}`)
			case requestPath == "/api/v1/admin/sub-admins/batch" && httpMethod == http.MethodPost:
				body = []byte(`{"emails":["a@example.com"],"permissionNames":["works.read"]}`)
			case strings.Contains(requestPath, "/api/v1/admin/sub-admins/") && strings.HasSuffix(requestPath, "/permissions") && httpMethod == http.MethodPut:
				body = []byte(`{"permissionNames":["works.read"]}`)
			case requestPath == "/api/v1/admin/sub-admins/handover-super" && httpMethod == http.MethodPost:
				body = []byte(`{"newSuperAdminID":1}`)
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
