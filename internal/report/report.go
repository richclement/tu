package report

const SchemaVersionV1 = "v1"

type ResultKind string

const (
	ResultKindFile    ResultKind = "file"
	ResultKindSummary ResultKind = "summary"
)

type Status string

const (
	StatusCounted Status = "counted"
	StatusSkipped Status = "skipped"
)

type Method string

const (
	MethodExact     Method = "exact"
	MethodHeuristic Method = "heuristic"
)

type Summary struct {
	FilesSeen    int64 `json:"files_seen"`
	FilesCounted int64 `json:"files_counted"`
	FilesSkipped int64 `json:"files_skipped"`
	TotalTokens  int64 `json:"total_tokens"`
}

type Result struct {
	Kind     ResultKind `json:"kind"`
	Path     string     `json:"path"`
	Tokens   *int64     `json:"tokens"`
	Method   *Method    `json:"method"`
	Provider *string    `json:"provider"`
	Status   Status     `json:"status"`
	Reason   *string    `json:"reason"`
}

type ScanReport struct {
	SchemaVersion    string   `json:"schema_version"`
	Target           string   `json:"target"`
	Root             string   `json:"root"`
	Recursive        bool     `json:"recursive"`
	RespectGitIgnore bool     `json:"respect_gitignore"`
	Sort             string   `json:"sort"`
	Summary          Summary  `json:"summary"`
	Results          []Result `json:"results"`
}
