package scriptflow

import "time"

const (
	ScopeSubmission = "submission"
	ScopeSystem     = "system"
	ScopeJudge      = "judge"
)

const (
	EventSubmissionPre       = "submission_pre"
	EventSubmissionUpdatePre = "submission_update_pre"
	EventSubmissionDeletePre = "submission_delete_pre"
	EventFilePre             = "file_pre"
	EventFilePost            = "file_post"
	EventContestEnd          = "contest_end"
)

type ExecuteInput struct {
	Scope    string         `json:"scope"`
	EventKey string         `json:"eventKey"`
	FlowKey  string         `json:"flowKey"`
	TraceID  string         `json:"traceID"`
	NowUnix  int64          `json:"nowUnix"`
	Context  map[string]any `json:"context"`
	Payload  map[string]any `json:"payload"`
}

type ExecuteOutput struct {
	Allow   bool           `json:"allow"`
	Message string         `json:"message,omitempty"`
	Patch   map[string]any `json:"patch,omitempty"`
	Metrics map[string]any `json:"metrics,omitempty"`
}

type StepConfig struct {
	StepName        string         `json:"stepName"`
	Interpreter     string         `json:"interpreter"`
	ScriptPath      string         `json:"scriptPath"`
	Timeout         time.Duration  `json:"timeout"`
	FailureStrategy string         `json:"failureStrategy"`
	RetryCount      int            `json:"retryCount"`
	InputTemplate   map[string]any `json:"inputTemplate"`
}

type ChainConfig struct {
	Scope    string       `json:"scope"`
	EventKey string       `json:"eventKey"`
	FlowKey  string       `json:"flowKey"`
	Steps    []StepConfig `json:"steps"`
}

type StepResult struct {
	StepName   string         `json:"stepName"`
	Success    bool           `json:"success"`
	DurationMs int64          `json:"durationMs"`
	Message    string         `json:"message,omitempty"`
	Patch      map[string]any `json:"patch,omitempty"`
}

type ChainResult struct {
	Allowed bool           `json:"allowed"`
	Reason  string         `json:"reason,omitempty"`
	Patch   map[string]any `json:"patch,omitempty"`
	Steps   []StepResult   `json:"steps"`
}
