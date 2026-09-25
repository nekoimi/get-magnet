package script

import (
	"context"
	"testing"
)

func TestExecuteRejectsDynamicCode(t *testing.T) {
	if _, err := Execute(context.Background(), Request{Script: "return eval('1 + 1');"}); err == nil {
		t.Fatal("expected eval to be unavailable")
	}
}
