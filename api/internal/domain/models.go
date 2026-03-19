package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	RoleOwner  = "OWNER"
	RoleEditor = "EDITOR"
	RoleViewer = "VIEWER"

	TaskStatusTodo       = "todo"
	TaskStatusInProgress = "in_progress"
	TaskStatusDone       = "done"

	ApprovalStateNotRequired      = "not_required"
	ApprovalStatePending          = "pending"
	ApprovalStatePartiallyApprove = "partially_approved"
	ApprovalStateApproved         = "approved"

	MeetingStatusPlanned   = "planned"
	MeetingStatusCompleted = "completed"
	MeetingStatusCancelled = "cancelled"

	AnnouncementStatusDraft            = "draft"
	AnnouncementStatusPublishRequested = "publish_requested"
	AnnouncementStatusPublished        = "published"
	AnnouncementStatusPublishFailed    = "publish_failed"
)

type Actor struct {
	UserID         uuid.UUID
	OrganizationID uuid.UUID
	Role           string
	Subject        string
	Email          string
	Name           string
}

type Principal struct {
	Subject string
	Email   string
	Name    string
}

type Organization struct {
	ID   uuid.UUID
	Slug string
	Name string
}

type Member struct {
	ID    uuid.UUID
	Email string
	Name  string
	Role  string
}

type Meeting struct {
	ID          uuid.UUID
	Year        int32
	Month       int32
	ScheduledAt *time.Time
	MeetingURL  *string
	Notes       *string
	Status      string
}

type Task struct {
	ID                  uuid.UUID
	Title               string
	Status              string
	DueDate             *time.Time
	ReferenceURL        *string
	AssigneeID          *uuid.UUID
	CreatedBy           uuid.UUID
	ApproverIDs         []uuid.UUID
	ApprovedApproverIDs []uuid.UUID
	ApprovalState       string
	Version             int32
	Year                int32
	Month               int32
}

type Announcement struct {
	ID               uuid.UUID
	Year             int32
	Month            int32
	Body             string
	Status           string
	PublishChannel   *string
	DiscordMessageID *string
	LastError        *string
}

type PublishRequest struct {
	AnnouncementID uuid.UUID
	Year           int32
	Month          int32
	Body           string
	PublishChannel string
}

type WorkflowPeriod struct {
	Year                 int32
	Month                int32
	MeetingStatus        string
	TaskCompletionStatus string
	AnnouncementStatus   string
	NextStageReady       bool
}

func ComputeApprovalState(required, approved int) string {
	switch {
	case required == 0:
		return ApprovalStateNotRequired
	case approved == 0:
		return ApprovalStatePending
	case approved < required:
		return ApprovalStatePartiallyApprove
	default:
		return ApprovalStateApproved
	}
}
