package go_program

import "time"

const (
	DefaultBuildTimeout   = 80 * time.Second
	DefaultExecuteTimeout = 20 * time.Second
	DefaultLoopTimeout    = 240 * time.Second
	DefaultStdoutLimit    = 512 * 1024
	DefaultStderrLimit    = 128 * 1024
	DefaultMaxAttempts    = 3
)

type Manifest struct {
	ProgramID       string            `json:"program_id"`
	DisplayName     string            `json:"display_name"`
	Purpose         string            `json:"purpose"`
	SourceDir       string            `json:"source_dir"`
	Permissions     Permissions       `json:"permissions"`
	InputSchema     ObjectSchema      `json:"input_schema"`
	OutputSchema    ObjectSchema      `json:"output_schema"`
	VendorAllowlist []string          `json:"vendor_allowlist,omitempty"`
	DataSources     []DataSource      `json:"data_sources,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

type Permissions struct {
	ReadAppData         bool     `json:"read_app_data"`
	ReadOutputs         bool     `json:"read_outputs"`
	WriteOutputsScratch bool     `json:"write_outputs_scratch"`
	ReadMountedPaths    []string `json:"read_mounted_paths,omitempty"`
	ReadDBAsJSON        bool     `json:"read_db_as_json"`
	Network             bool     `json:"network"`
	ShellSubprocess     bool     `json:"shell_subprocess"`
}

type DataSource struct {
	Kind string `json:"kind"` // xlsx, csv, builtin_db, weather_json, web_json
	Name string `json:"name"`
	Path string `json:"path,omitempty"`
}

type ObjectSchema struct {
	Required []string `json:"required,omitempty"`
}

type Limits struct {
	BuildTimeout   time.Duration `json:"build_timeout"`
	ExecuteTimeout time.Duration `json:"execute_timeout"`
	LoopTimeout    time.Duration `json:"loop_timeout"`
	StdoutBytes    int64         `json:"stdout_bytes"`
	StderrBytes    int64         `json:"stderr_bytes"`
	MaxAttempts    int           `json:"max_attempts"`
}

func DefaultLimits() Limits {
	return Limits{
		BuildTimeout:   DefaultBuildTimeout,
		ExecuteTimeout: DefaultExecuteTimeout,
		LoopTimeout:    DefaultLoopTimeout,
		StdoutBytes:    DefaultStdoutLimit,
		StderrBytes:    DefaultStderrLimit,
		MaxAttempts:    DefaultMaxAttempts,
	}
}

func (l Limits) Normalize() Limits {
	def := DefaultLimits()
	if l.BuildTimeout <= 0 {
		l.BuildTimeout = def.BuildTimeout
	}
	if l.ExecuteTimeout <= 0 {
		l.ExecuteTimeout = def.ExecuteTimeout
	}
	if l.LoopTimeout <= 0 {
		l.LoopTimeout = def.LoopTimeout
	}
	if l.StdoutBytes <= 0 {
		l.StdoutBytes = def.StdoutBytes
	}
	if l.StderrBytes <= 0 {
		l.StderrBytes = def.StderrBytes
	}
	if l.MaxAttempts <= 0 {
		l.MaxAttempts = def.MaxAttempts
	}
	return l
}

type Toolchain struct {
	GoBinary string `json:"go_binary"`
	Version  string `json:"version"`
}

type ReviewKind string

const (
	ReviewUnauthorizedPackage ReviewKind = "unauthorized_package"
	ReviewNetworkRequired     ReviewKind = "network_required"
	ReviewShellRequired       ReviewKind = "shell_subprocess_required"
	ReviewSandboxRequired     ReviewKind = "sandbox_required"
	ReviewDataSourceMissing   ReviewKind = "data_source_missing"
)

type ReviewRequest struct {
	Kind    ReviewKind `json:"kind"`
	Subject string     `json:"subject"`
	Reason  string     `json:"reason"`
}

type ValidationIssue struct {
	Kind   ReviewKind `json:"kind"`
	File   string     `json:"file,omitempty"`
	Import string     `json:"import,omitempty"`
	Reason string     `json:"reason"`
	Review bool       `json:"review"`
}

type ValidationResult struct {
	Hash           string            `json:"hash"`
	GoFiles        []string          `json:"go_files"`
	Issues         []ValidationIssue `json:"issues"`
	ReviewRequests []ReviewRequest   `json:"review_requests"`
}

func (r ValidationResult) HasIssues() bool {
	return len(r.Issues) > 0
}

type BuildResult struct {
	BinaryPath string `json:"binary_path"`
	Stdout     string `json:"stdout,omitempty"`
	Stderr     string `json:"stderr,omitempty"`
}

type ExecuteResult struct {
	Stdout       []byte `json:"stdout"`
	Stderr       []byte `json:"stderr,omitempty"`
	StdoutSHA256 string `json:"stdout_sha256"`
}

type ErrorClass string

const (
	ErrorModelFixable      ErrorClass = "model_fixable"
	ErrorNeedsUserDecision ErrorClass = "needs_user_decision"
	ErrorNoProgress        ErrorClass = "no_progress"
)

type ClassifiedError struct {
	Class  ErrorClass `json:"class"`
	Reason string     `json:"reason"`
}
