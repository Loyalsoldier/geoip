package singbox

import (
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestNewSRSOutUsesDefaultOutputDir(t *testing.T) {
	converter := NewSRSOut(lib.ActionOutput)
	output, ok := converter.(*srs_out)
	if !ok {
		t.Fatalf("NewSRSOut() returned %T, want *srs_out", converter)
	}
	if output.OutputDir != defaultOutputDir {
		t.Fatalf("OutputDir = %q, want %q", output.OutputDir, defaultOutputDir)
	}
}
