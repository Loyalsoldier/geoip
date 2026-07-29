package singbox

import (
	"testing"

	"github.com/Loyalsoldier/geoip/lib"
)

func TestNewSRSOutUsesDefaultOutputDir(t *testing.T) {
	output := NewSRSOut(lib.ActionOutput).(*srs_out)
	if output.OutputDir != defaultOutputDir {
		t.Fatalf("OutputDir = %q, want %q", output.OutputDir, defaultOutputDir)
	}
}
