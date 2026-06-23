package go_program

import "strings"

func ClassifyError(err error, issues []ValidationIssue, repeated bool) ClassifiedError {
	if repeated {
		return ClassifiedError{Class: ErrorNoProgress, Reason: "same code hash or same error repeated"}
	}
	for _, issue := range issues {
		switch issue.Kind {
		case ReviewUnauthorizedPackage, ReviewNetworkRequired, ReviewShellRequired, ReviewSandboxRequired, ReviewDataSourceMissing:
			return ClassifiedError{Class: ErrorNeedsUserDecision, Reason: issue.Reason}
		}
	}
	if err == nil {
		return ClassifiedError{}
	}
	text := strings.ToLower(err.Error())
	switch {
	case strings.Contains(text, "syntax") ||
		strings.Contains(text, "build failed") ||
		strings.Contains(text, "json invalid") ||
		strings.Contains(text, "missing required field") ||
		strings.Contains(text, "panic") ||
		strings.Contains(text, "execute failed"):
		return ClassifiedError{Class: ErrorModelFixable, Reason: err.Error()}
	case strings.Contains(text, "timeout") ||
		strings.Contains(text, "permission") ||
		strings.Contains(text, "sandbox"):
		return ClassifiedError{Class: ErrorNeedsUserDecision, Reason: err.Error()}
	default:
		return ClassifiedError{Class: ErrorModelFixable, Reason: err.Error()}
	}
}
