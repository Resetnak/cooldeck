package views

import (
	"fmt"
	"testing"
	"time"

	"github.com/resetnak/cooldeck/internal/domain"
	"github.com/resetnak/cooldeck/internal/tui/theme"
)

// BenchmarkApplicationsReproject checks that filter+sort stays cheap at the
// dashboard target size (200 apps). Run: make bench
func BenchmarkApplicationsReproject(b *testing.B) {
	apps := make([]domain.Application, 200)
	for i := range apps {
		apps[i] = domain.Application{
			UUID:    fmt.Sprintf("app-%03d", i),
			Name:    fmt.Sprintf("service-%03d", i),
			Status:  domain.ParseStatus("running:healthy"),
			Project: domain.ResourceRef{Name: "proj"},
			Environment: domain.ResourceRef{
				Name: map[bool]string{true: "production", false: "staging"}[i%5 == 0],
			},
			Branch: "main",
		}
	}
	view := NewApplications()
	view.SetApplications(apps, nil, time.Now())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		view.SetFilter("status:running")
		view.SetSort(SortByName)
		view.SetFilter("")
		view.SetSort(SortByStatus)
	}
}

// BenchmarkApplicationsRender measures a full table paint at 200 rows.
func BenchmarkApplicationsRender(b *testing.B) {
	apps := make([]domain.Application, 200)
	for i := range apps {
		apps[i] = domain.Application{
			UUID:   fmt.Sprintf("app-%03d", i),
			Name:   fmt.Sprintf("service-%03d", i),
			Status: domain.ParseStatus("running:healthy"),
			Branch: "main",
		}
	}
	view := NewApplications()
	view.SetApplications(apps, nil, time.Now())
	th := theme.New(theme.Options{Mode: theme.ModeDark, ASCII: true})
	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = view.Render(th, 120, 40, true, now)
	}
}
