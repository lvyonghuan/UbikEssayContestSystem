package submission

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type submissionSwagger struct {
	Paths map[string]map[string]any `json:"paths"`
}

func TestSwaggerDeclaredAPIsAreReachable(t *testing.T) {
	setupSubmissionRouteMocks(t)

	wd, _ := os.Getwd()
	swaggerPath := filepath.Join(wd, "..", "docs", "API", "Submission", "Submission_swagger.json")
	tmp := t.TempDir()
	_ = os.Chdir(tmp)
	t.Cleanup(func() { _ = os.Chdir(wd) })

	router := buildSubmissionRouter()

	content, err := os.ReadFile(swaggerPath)
	if err != nil {
		t.Fatalf("read swagger file failed: %v", err)
	}

	var doc submissionSwagger
	if err = json.Unmarshal(content, &doc); err != nil {
		t.Fatalf("unmarshal swagger failed: %v", err)
	}
	if len(doc.Paths) == 0 {
		t.Fatal("swagger paths should not be empty")
	}

	requiredPaths := []string{
		"/author/register",
		"/author/login",
		"/author/refresh",
		"/author",
		"/author/submission",
		"/author/submission/{id}",
		"/author/submission/file",
	}
	for _, p := range requiredPaths {
		if _, ok := doc.Paths[p]; !ok {
			t.Fatalf("swagger missing required path: %s", p)
		}
	}

	replacer := strings.NewReplacer("{id}", "1")
	for path, methods := range doc.Paths {
		for method := range methods {
			httpMethod := strings.ToUpper(method)
			requestPath := "/api/v1" + replacer.Replace(path)
			auth := "Bearer author"
			var body []byte

			switch {
			case requestPath == "/api/v1/author/register":
				auth = ""
				body = []byte(`{"authorName":"u","password":"p"}`)
			case requestPath == "/api/v1/author/login":
				auth = ""
				body = []byte(`{"authorID":1,"password":"p"}`)
			case requestPath == "/api/v1/author/refresh":
				auth = "Bearer refresh-author"
			case requestPath == "/api/v1/author":
				body = []byte(`{"authorID":1,"authorName":"u2"}`)
			case requestPath == "/api/v1/author/submission" && (httpMethod == http.MethodPost || httpMethod == http.MethodPut || httpMethod == http.MethodDelete):
				body = []byte(`{"workID":1,"authorID":1,"trackID":2,"workTitle":"w"}`)
			}

			if requestPath == "/api/v1/author/submission/file" && httpMethod == http.MethodPost {
				w := doMultipartRequest(
					router,
					requestPath,
					auth,
					map[string]string{"work_id": "10"},
					"article_file",
					"paper.docx",
					[]byte("docx-content"),
				)
				if w.Code != http.StatusOK {
					t.Fatalf("unexpected http status for %s %s: %d", httpMethod, requestPath, w.Code)
				}
				if code := decodeRespCode(t, w.Body.Bytes()); code == 404 {
					t.Fatalf("swagger path is not reachable: %s %s", httpMethod, requestPath)
				}
				continue
			}

			w := doJSONRequest(router, httpMethod, requestPath, auth, body)
			if w.Code != http.StatusOK {
				t.Fatalf("unexpected http status for %s %s: %d", httpMethod, requestPath, w.Code)
			}
			if code := decodeRespCode(t, w.Body.Bytes()); code == 404 {
				t.Fatalf("swagger path is not reachable: %s %s", httpMethod, requestPath)
			}
		}
	}
}
