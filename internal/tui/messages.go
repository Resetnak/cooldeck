package tui

import (
	"time"

	"github.com/resetnak/cooldeck/internal/app"
	"github.com/resetnak/cooldeck/internal/domain"
)

// Every asynchronous result carries the sequence number of the request that
// produced it. A response whose sequence is older than the model's current one
// is dropped, so a slow request that lands after a newer one cannot overwrite
// fresher data. Sequence numbers are per request kind, because a dashboard
// refresh and a detail load progress independently.

type connectedMsg struct {
	Seq        uint64
	Connection app.Connection
}

type connectFailedMsg struct {
	Seq uint64
	Err *domain.Error
}

type dashboardLoadedMsg struct {
	Seq      uint64
	Snapshot app.DashboardSnapshot
}

type dashboardFailedMsg struct {
	Seq uint64
	Err *domain.Error
}

type detailLoadedMsg struct {
	Seq     uint64
	AppUUID string
	Detail  app.ApplicationDetail
}

type detailFailedMsg struct {
	Seq     uint64
	AppUUID string
	Err     *domain.Error
}

type runtimeLogsLoadedMsg struct {
	Seq      uint64
	AppUUID  string
	Snapshot app.LogSnapshot
}

type runtimeLogsFailedMsg struct {
	Seq     uint64
	AppUUID string
	Err     *domain.Error
}

type deploymentLogsLoadedMsg struct {
	Seq            uint64
	DeploymentUUID string
	Snapshot       app.LogSnapshot
}

type deploymentLogsFailedMsg struct {
	Seq            uint64
	DeploymentUUID string
	Err            *domain.Error
}

type actionCompletedMsg struct {
	Seq    uint64
	Result app.OperationResult
}

type actionFailedMsg struct {
	Seq uint64
	Err *domain.Error
}

// tailTickMsg schedules one fleet-tail source. Unlike the other request kinds
// several are in flight at once, so the sequence number is what tells a live
// reply from one that belongs to a tail the user has already left.
type tailTickMsg struct {
	Seq     uint64
	AppUUID string
}

type tailLoadedMsg struct {
	Seq      uint64
	AppUUID  string
	Snapshot app.LogSnapshot
}

type tailFailedMsg struct {
	Seq     uint64
	AppUUID string
	Err     *domain.Error
}

type runtimeLogsTickMsg struct {
	AppUUID string
}

// refreshTickMsg drives the periodic dashboard refresh.
type refreshTickMsg time.Time

// frameTickMsg drives animation: the header spinner, relative timestamps and
// toast expiry. It runs at a low rate because nothing on screen needs to be
// smooth, and a fast tick over SSH is wasted bandwidth.
type frameTickMsg time.Time

// toastMsg adds a notification to the stack.
type toastMsg struct {
	Kind   int
	Text   string
	Detail string
}

// openURLFailedMsg reports that the OS refused to open a link.
type openURLFailedMsg struct{ Err error }

// instanceSwitchedMsg is delivered after OpenService succeeds.
type instanceSwitchedMsg struct {
	Service      app.Service
	InstanceID   string
	InstanceName string
}

// instanceSwitchFailedMsg is delivered when OpenService fails.
type instanceSwitchFailedMsg struct {
	InstanceID string
	Err        *domain.Error
}

// instanceRemovedMsg is delivered after a local instance is deleted from config.
// When NextID is set the model should switch onto that instance next.
type instanceRemovedMsg struct {
	Name   string
	NextID string
	// EmptyFleet is true when no configured instances remain.
	EmptyFleet bool
}
