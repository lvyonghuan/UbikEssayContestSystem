package system

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
)

type systemSwagger struct {
	Paths map[string]map[string]interface{} `json:"paths"`
}

func TestSwaggerDeclaredAPIsAreReachable(t *testing.T) {
	mockSystemAuthAndSources(t)
	router := buildGlobalInfoRouter()

	content, err := os.ReadFile("../docs/API/System/GlobalInfo_swagger.json")
	if err != nil {
		t.Fatalf("read swagger file failed: %v", err)
	}

	var doc systemSwagger
	if err = json.Unmarshal(content, &doc); err != nil {
		t.Fatalf("unmarshal swagger failed: %v", err)
	}
	if len(doc.Paths) == 0 {
		t.Fatal("swagger paths should not be empty")
	}

	requiredPaths := []string{
		"/contests",
		"/contests/{contest_id}",
		"/tracks/{contest_id}",
		"/tracks/detail/{track_id}",
	}
	for _, required := range requiredPaths {
		if _, ok := doc.Paths[required]; !ok {
			t.Fatalf("swagger missing required path: %s", required)
		}
	}

	replacer := strings.NewReplacer(
		"{contest_id}", "1",
		"{track_id}", "1",
	)

	for path, methods := range doc.Paths {
		for method := range methods {
			httpMethod := strings.ToUpper(method)
			requestPath := "/api/v1" + replacer.Replace(path)

			w := doRequest(router, httpMethod, requestPath, nil, "Bearer ok")
			if w.Code != http.StatusOK {
				t.Fatalf("unexpected http status for %s %s: %d", httpMethod, requestPath, w.Code)
			}

			if got := reqCode(t, w.Body.Bytes()); got == 404 {
				t.Fatalf("swagger path is not reachable: %s %s", httpMethod, requestPath)
			}
		}
	}
}
