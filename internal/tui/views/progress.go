package views

import (
	"time"

	"github.com/resetnak/cooldeck/internal/domain"
	"github.com/resetnak/cooldeck/internal/tui/components"
	"github.com/resetnak/cooldeck/internal/tui/theme"
)

// baselineSample is how many finished deployments the expected duration is
// taken from. Five is a compromise: enough that one slow build does not move
// the median, few enough that the estimate follows a project that genuinely
// grows heavier over time.
const baselineSample = 5

// progressColumn is the table column the bar lives in. It is only added when
// something is actually in flight - a column that is blank on every row is
// wasted width, and reserving space for content that is not there is a mistake
// this codebase has already made once.
var progressColumn = components.Column{
	Title:      "Progress",
	ShortTitle: "Prog",
	MinWidth:   components.DurationBarWidth(),
	Priority:   2,
}

// activeFirst orders in-flight deployments ahead of settled ones. It is only
// ever used with a stable sort, so everything else keeps the order Coolify
// returned it in - newest first.
func activeFirst(a, b domain.Deployment) int {
	switch {
	case a.Status.IsActive() == b.Status.IsActive():
		return 0
	case a.Status.IsActive():
		return -1
	default:
		return 1
	}
}

// anyDeploymentActive reports whether the Progress column would carry anything.
func anyDeploymentActive(deployments []domain.Deployment) bool {
	for _, d := range deployments {
		if d.Status.IsActive() {
			return true
		}
	}
	return false
}

// deploymentProgressCell renders one Progress cell. history is the deployment
// list the baseline is taken from; it may be a single application's history or
// a mixed list, since BaselineDuration filters by application.
func deploymentProgressCell(
	th *theme.Theme,
	deployment domain.Deployment,
	history []domain.Deployment,
	now time.Time,
) string {
	if !deployment.Status.IsActive() {
		return ""
	}
	baseline := domain.BaselineDuration(history, deployment.ApplicationUUID, baselineSample)
	return components.DurationBar(th, deployment.Duration(now), baseline)
}
