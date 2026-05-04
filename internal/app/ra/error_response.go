package ra

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/e11it/ra/pkg/validate"
)

const payloadValidationMessageBase = "Ra: payload validation errors"

// FormatPayloadValidationMessage формирует корневой message для ошибки payload validation:
// базовый текст, уникальные code из errors (порядок первого появления), trace_id в конце.
func FormatPayloadValidationMessage(details RAErrorDetails) string {
	var b strings.Builder
	b.WriteString(payloadValidationMessageBase)
	seen := make(map[string]struct{})
	var codes []string
	for _, e := range details.Errors {
		c := strings.TrimSpace(e.Code)
		if c == "" {
			continue
		}
		if _, ok := seen[c]; ok {
			continue
		}
		seen[c] = struct{}{}
		codes = append(codes, c)
	}
	if len(codes) > 0 {
		b.WriteString(". Problems: [")
		b.WriteString(strings.Join(codes, ", "))
		b.WriteString("]")
	}
	if tid := strings.TrimSpace(details.TraceID); tid != "" {
		b.WriteString(". Trace ID: ")
		b.WriteString(tid)
		b.WriteByte('.')
	}
	return b.String()
}

type RAErrorResponse struct {
	ErrorCode int            `json:"error_code"`
	Message   string         `json:"message"`
	Details   RAErrorDetails `json:"details"`
}

type RAErrorDetails struct {
	TraceID string                  `json:"trace_id,omitempty"`
	Reason  string                  `json:"reason,omitempty"`
	Errors  []RAValidationErrorItem `json:"errors,omitempty"`
}

type RAValidationErrorItem struct {
	RecordIndex int    `json:"record_index"`
	Path        string `json:"path,omitempty"`
	Code        string `json:"code,omitempty"`
	Message     string `json:"message,omitempty"`
}

func BuildValidationDetails(rep *validate.Report, traceID string) RAErrorDetails {
	details := RAErrorDetails{
		TraceID: traceID,
		Errors:  make([]RAValidationErrorItem, 0),
	}
	if rep == nil {
		return details
	}
	for _, issue := range rep.Issues() {
		if issue.Severity != validate.SeverityError {
			continue
		}
		details.Errors = append(details.Errors, RAValidationErrorItem{
			RecordIndex: issue.RecordIndex,
			Path:        issue.Path,
			Code:        issue.Code,
			Message:     issue.Message,
		})
	}
	return details
}

func WriteJSONErrorGin(
	c *gin.Context,
	statusCode, errorCode int,
	message, headerSummary string,
	details RAErrorDetails,
) {
	summary := strings.TrimSpace(headerSummary)
	if summary == "" {
		summary = message
	}
	if summary != "" {
		c.Header("X-RA-ERROR", summary)
	}
	c.JSON(statusCode, RAErrorResponse{
		ErrorCode: errorCode,
		Message:   message,
		Details:   details,
	})
	c.Abort()
}

func GinTraceID(c *gin.Context) string {
	if rid, ok := c.Get("x-request-id"); ok {
		if s, ok := rid.(string); ok {
			return s
		}
	}
	return c.GetHeader("X-Request-ID")
}

func StatusForErrorCode(errorCode int) int {
	return DescribeErrorCode(errorCode).HTTPStatus
}

func DetailsWithReason(traceID string, err error) RAErrorDetails {
	details := RAErrorDetails{TraceID: traceID}
	if err == nil {
		return details
	}
	details.Reason = err.Error()
	return details
}

func IsPayloadValidationError(status int, errorCode int) bool {
	return status == http.StatusUnprocessableEntity && errorCode == ErrorCodePayloadValidate
}

func IsAuthDenyError(errorCode int) bool {
	return errorCode == ErrorCodeAuthDenied || errorCode == ErrorCodeReloadForbidden
}

func WrapErrorMessage(prefix string, err error) string {
	if err == nil {
		return prefix
	}
	return prefix + ": " + err.Error()
}

func AnyError(values ...error) error {
	for _, err := range values {
		if err != nil {
			return err
		}
	}
	return nil
}

func IsAny(err error, targets ...error) bool {
	for _, target := range targets {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}
