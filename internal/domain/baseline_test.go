package domain

import (
	"testing"
	"time"
)

// finished builds a completed deployment of the given app that ran for d,
// started `ago` before a fixed reference point.
func finished(app string, ago, d time.Duration) Deployment {
	start := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC).Add(-ago)
	end := start.Add(d)
	return Deployment{
		ApplicationUUID: app,
		Status:          DeploymentFinished,
		StartedAt:       start,
		FinishedAt:      &end,
	}
}

func TestBaselineDurationTakesTheMedianOfTheNewestRuns(t *testing.T) {
	deployments := []Deployment{
		// Deliberately out of order: the caller's ordering is not guaranteed,
		// and "most recent" has to mean most recent for the estimate to track a
		// project that grows heavier.
		finished("app", 3*time.Hour, 60*time.Second),
		finished("app", 1*time.Hour, 40*time.Second),
		finished("app", 2*time.Hour, 50*time.Second),
		// Older than the sample window, and wildly slow: must not count.
		finished("app", 99*time.Hour, 90*time.Minute),
	}

	if got := BaselineDuration(deployments, "app", 3); got != 50*time.Second {
		t.Fatalf("baseline = %s, want 50s", got)
	}
}

func TestBaselineDurationIgnoresOtherApplications(t *testing.T) {
	deployments := []Deployment{
		finished("app", time.Hour, 30*time.Second),
		finished("other", time.Hour, 30*time.Minute),
	}

	if got := BaselineDuration(deployments, "app", 5); got != 30*time.Second {
		t.Fatalf("baseline = %s, want 30s", got)
	}
}

func TestBaselineDurationCountsOnlySuccesses(t *testing.T) {
	start := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	end := start.Add(2 * time.Second)

	// A build that died after two seconds says nothing about how long a
	// successful one takes; counting it would make every normal build look
	// overdue.
	deployments := []Deployment{
		finished("app", time.Hour, 60*time.Second),
		{ApplicationUUID: "app", Status: DeploymentFailed, StartedAt: start, FinishedAt: &end},
		{ApplicationUUID: "app", Status: DeploymentCancelled, StartedAt: start, FinishedAt: &end},
		{ApplicationUUID: "app", Status: DeploymentInProgress, StartedAt: start},
	}

	if got := BaselineDuration(deployments, "app", 5); got != 60*time.Second {
		t.Fatalf("baseline = %s, want 60s", got)
	}
}

func TestBaselineDurationAveragesTheMiddlePairWhenEven(t *testing.T) {
	deployments := []Deployment{
		finished("app", time.Hour, 10*time.Second),
		finished("app", 2*time.Hour, 20*time.Second),
		finished("app", 3*time.Hour, 30*time.Second),
		finished("app", 4*time.Hour, 60*time.Second),
	}

	if got := BaselineDuration(deployments, "app", 4); got != 25*time.Second {
		t.Fatalf("baseline = %s, want 25s", got)
	}
}

func TestBaselineDurationIsZeroWithoutUsableHistory(t *testing.T) {
	cases := map[string][]Deployment{
		"no deployments at all": nil,
		"none finished": {
			{ApplicationUUID: "app", Status: DeploymentInProgress},
		},
		"none for this application": {
			finished("other", time.Hour, time.Minute),
		},
	}

	for name, deployments := range cases {
		t.Run(name, func(t *testing.T) {
			// Zero means "no expectation to show". A caller that treats it as a
			// duration would draw a full bar the instant a build starts.
			if got := BaselineDuration(deployments, "app", 5); got != 0 {
				t.Fatalf("baseline = %s, want 0", got)
			}
		})
	}
}

func TestBaselineDurationHistoricalRunsDoNotDriftWithTheClock(t *testing.T) {
	deployments := []Deployment{finished("app", time.Hour, 45*time.Second)}

	first := BaselineDuration(deployments, "app", 5)
	// Duration() measures an unfinished deployment against now; a finished one
	// must be measured against its own end, or the baseline would grow forever.
	time.Sleep(2 * time.Millisecond)
	if second := BaselineDuration(deployments, "app", 5); second != first {
		t.Fatalf("baseline moved from %s to %s", first, second)
	}
	if first != 45*time.Second {
		t.Fatalf("baseline = %s, want 45s", first)
	}
}
