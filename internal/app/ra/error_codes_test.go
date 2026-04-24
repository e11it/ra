package ra

import (
	"net/http"
	"testing"
)

func TestDescribeErrorCode(t *testing.T) {
	meta := DescribeErrorCode(ErrorCodePayloadValidate)
	if meta.HTTPStatus != http.StatusUnprocessableEntity {
		t.Fatalf("status mismatch: got=%d", meta.HTTPStatus)
	}
	if meta.Nature != ErrorNatureValidation {
		t.Fatalf("nature mismatch: got=%s", meta.Nature)
	}

	unknown := DescribeErrorCode(99999)
	if unknown.Nature != ErrorNatureUnknown {
		t.Fatalf("unknown nature mismatch: got=%s", unknown.Nature)
	}
}
