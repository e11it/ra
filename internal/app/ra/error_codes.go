package ra

import "net/http"

// RA error codes are grouped by nature/range:
//   - 4xxxx: request/auth/domain validation issues
//   - 5xxxx: runtime/upstream/internal issues
//
// Keep these constants as a single source of truth for API contract.
const (
	ErrorCodeAuthDenied      = 40301
	ErrorCodeMalformedBody   = 40010
	ErrorCodePayloadValidate = 42230
	ErrorCodeReloadForbidden = 40310
	ErrorCodeReloadFailed    = 40020
)

type ErrorNature string

const (
	ErrorNatureAuth       ErrorNature = "auth"
	ErrorNatureMalformed  ErrorNature = "malformed_request"
	ErrorNatureValidation ErrorNature = "payload_validation"
	ErrorNatureConfig     ErrorNature = "config_reload"
	ErrorNatureUnknown    ErrorNature = "unknown"
)

type ErrorCodeMeta struct {
	HTTPStatus int
	Nature     ErrorNature
	Meaning    string
}

var errorCodeMeta = map[int]ErrorCodeMeta{
	ErrorCodeAuthDenied: {
		HTTPStatus: http.StatusForbidden,
		Nature:     ErrorNatureAuth,
		Meaning:    "request denied by ACL/auth checks",
	},
	ErrorCodeMalformedBody: {
		HTTPStatus: http.StatusBadRequest,
		Nature:     ErrorNatureMalformed,
		Meaning:    "request body cannot be parsed/read",
	},
	ErrorCodePayloadValidate: {
		HTTPStatus: http.StatusUnprocessableEntity,
		Nature:     ErrorNatureValidation,
		Meaning:    "payload failed domain validation checks",
	},
	ErrorCodeReloadForbidden: {
		HTTPStatus: http.StatusForbidden,
		Nature:     ErrorNatureConfig,
		Meaning:    "reload endpoint access forbidden by admin policy",
	},
	ErrorCodeReloadFailed: {
		HTTPStatus: http.StatusBadRequest,
		Nature:     ErrorNatureConfig,
		Meaning:    "config reload failed due to invalid input/state",
	},
}

func DescribeErrorCode(code int) ErrorCodeMeta {
	if meta, ok := errorCodeMeta[code]; ok {
		return meta
	}
	return ErrorCodeMeta{
		HTTPStatus: http.StatusInternalServerError,
		Nature:     ErrorNatureUnknown,
		Meaning:    "unknown error code",
	}
}
