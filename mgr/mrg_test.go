package mgr

import (
	"log/slog"
	"net/url"
	"os"
	"testing"

	"github.com/platform-engineering-labs/orbital/opm/security"
	"github.com/platform-engineering-labs/orbital/opm/tree"
	"github.com/platform-engineering-labs/orbital/ops"
	"github.com/platform-engineering-labs/orbital/platform"
	"github.com/stretchr/testify/assert"
)

func TestMgr(t *testing.T) {
	repo, _ := url.Parse("https://hub.platform.engineering/repos/platform.engineering/pel#stable")

	mgr, err := New(slog.New(slog.NewTextHandler(os.Stderr, nil)), "/opt/pel", &tree.Config{
		OS:       platform.Current().OS,
		Arch:     platform.Current().Arch,
		Security: security.Default,
		Repositories: []ops.Repository{
			{
				Uri:      *repo,
				Priority: 0,
				Enabled:  true,
				Prune:    0,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	assert.False(t, mgr.Ready())
}
