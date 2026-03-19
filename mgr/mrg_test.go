package mgr

import (
	"log/slog"
	"net/url"
	"os"
	"testing"

	"github.com/goforj/godump"
	"github.com/platform-engineering-labs/orbital/opm/security"
	"github.com/platform-engineering-labs/orbital/opm/tree"
	"github.com/platform-engineering-labs/orbital/ops"
	"github.com/platform-engineering-labs/orbital/platform"
)

func TestMgr(t *testing.T) {
	repo, _ := url.Parse("https://hub.platform.engineering/repos/platform.engineering/pel#stable")

	mgr, err := New(slog.New(slog.NewTextHandler(os.Stderr, nil)), "/Users/discountelf/.pel/ops/trees/default", &tree.Config{
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

	available, err := mgr.Available()
	if err != nil {
		t.Fatal(err)
	}

	godump.Dump(available)
}
