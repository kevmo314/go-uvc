package transfers

import (
	"strings"
	"testing"
)

func TestClaimFrameReaderWithProbeCommitNil(t *testing.T) {
	si := &StreamingInterface{}
	_, err := si.ClaimFrameReaderWithProbeCommit(nil)
	if err == nil {
		t.Fatal("expected error for nil probe/commit control")
	}
	if !strings.Contains(err.Error(), "probe/commit control is nil") {
		t.Fatalf("unexpected error: %v", err)
	}
}
