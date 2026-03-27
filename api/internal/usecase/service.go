package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/dokkiitech/ashiato/api/internal/audit"
	"github.com/dokkiitech/ashiato/api/internal/config"
	"github.com/dokkiitech/ashiato/api/internal/db"
	"github.com/dokkiitech/ashiato/api/internal/domain"
	appctx "github.com/dokkiitech/ashiato/api/internal/middleware"
	"github.com/dokkiitech/ashiato/api/internal/repository"
)

type Service struct {
	store       *repository.Store
	logger      *slog.Logger
	defaultName string
	defaultSlug string
	ownerEmails map[string]struct{}
}

type PutMeetingInput struct {
	Year        int32
	Month       int32
	ScheduledAt *time.Time
	MeetingURL  *string
	Notes       *string
	Status      string
}

type CreateTaskInput struct {
	Title        string
	Status       string
	DueDate      *time.Time
	ReferenceURL *string
	AssigneeID   *uuid.UUID
	ApproverIDs  []uuid.UUID
	Year         int32
	Month        int32
}

type UpdateTaskInput struct {
	ID           uuid.UUID
	Title        string
	Status       string
	DueDate      *time.Time
	ReferenceURL *string
	AssigneeID   *uuid.UUID
	ApproverIDs  []uuid.UUID
	Version      int32
}

type PutAnnouncementInput struct {
	Year           int32
	Month          int32
	Body           string
	PublishChannel *string
}

type PublishAnnouncementInput struct {
	ID             uuid.UUID
	PublishChannel string
}

func NewService(store *repository.Store, logger *slog.Logger, cfg config.Config) *Service {
	return &Service{
		store:       store,
		logger:      logger,
		defaultName: cfg.DefaultOrgName,
		defaultSlug: cfg.DefaultOrgSlug,
		ownerEmails: cfg.OwnerEmails,
	}
}

func (s *Service) SyncPrincipal(ctx context.Context, principal domain.Principal) (domain.Actor, error) {
	q := s.store.Queries()

	org, err := q.EnsureOrganization(ctx, db.EnsureOrganizationParams{
		Slug: s.defaultSlug,
		Name: s.defaultName,
	})
	if err != nil {
		return domain.Actor{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to ensure organization", err)
	}

	user, err := q.UpsertUser(ctx, db.UpsertUserParams{
		OidcSubject: principal.Subject,
		Email:       principal.Email,
		Name:        principal.Name,
	})
	if err != nil {
		return domain.Actor{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to upsert user", err)
	}

	role := domain.RoleEditor
	if _, ok := s.ownerEmails[strings.ToLower(principal.Email)]; ok {
		role = domain.RoleOwner
	}

	member, err := q.EnsureOrganizationMembership(ctx, db.EnsureOrganizationMembershipParams{
		OrganizationID: org.ID,
		UserID:         user.ID,
		Role:           role,
	})
	if err != nil {
		return domain.Actor{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to ensure membership", err)
	}

	return domain.Actor{
		UserID:         uuidFromPG(user.ID),
		OrganizationID: uuidFromPG(org.ID),
		Role:           member.Role,
		Subject:        user.OidcSubject,
		Email:          user.Email,
		Name:           user.Name,
	}, nil
}

func (s *Service) GetMe(ctx context.Context, actor domain.Actor) (domain.Actor, domain.Organization, error) {
	org, err := s.store.Queries().GetOrganizationByID(ctx, pgUUID(actor.OrganizationID))
	if err != nil {
		return domain.Actor{}, domain.Organization{}, toAppError("organization not found", err)
	}

	return actor, domain.Organization{
		ID:   uuidFromPG(org.ID),
		Slug: org.Slug,
		Name: org.Name,
	}, nil
}

func (s *Service) ListMembers(ctx context.Context, actor domain.Actor) ([]domain.Member, error) {
	rows, err := s.store.Queries().ListOrganizationMembers(ctx, pgUUID(actor.OrganizationID))
	if err != nil {
		return nil, domain.NewAppError(domain.ErrorCodeInternal, "failed to list members", err)
	}

	members := make([]domain.Member, 0, len(rows))
	for _, row := range rows {
		members = append(members, domain.Member{
			ID:    uuidFromPG(row.ID),
			Email: row.Email,
			Name:  row.Name,
			Role:  row.Role,
		})
	}
	return members, nil
}

func (s *Service) ListWorkflowPeriods(ctx context.Context, actor domain.Actor) ([]domain.WorkflowPeriod, error) {
	q := s.store.Queries()
	meetings, err := q.ListMeetingPeriods(ctx, pgUUID(actor.OrganizationID))
	if err != nil {
		return nil, domain.NewAppError(domain.ErrorCodeInternal, "failed to list meeting periods", err)
	}
	tasks, err := q.ListTaskPeriodSummaries(ctx, pgUUID(actor.OrganizationID))
	if err != nil {
		return nil, domain.NewAppError(domain.ErrorCodeInternal, "failed to list task periods", err)
	}
	announcements, err := q.ListAnnouncementPeriods(ctx, pgUUID(actor.OrganizationID))
	if err != nil {
		return nil, domain.NewAppError(domain.ErrorCodeInternal, "failed to list announcement periods", err)
	}

	type key struct {
		year  int32
		month int32
	}

	periods := map[key]*domain.WorkflowPeriod{}
	getPeriod := func(year, month int32) *domain.WorkflowPeriod {
		k := key{year: year, month: month}
		if existing, ok := periods[k]; ok {
			return existing
		}
		periods[k] = &domain.WorkflowPeriod{
			Year:                 year,
			Month:                month,
			MeetingStatus:        "missing",
			TaskCompletionStatus: "empty",
			AnnouncementStatus:   "missing",
			NextStageReady:       false,
		}
		return periods[k]
	}

	for _, row := range meetings {
		period := getPeriod(row.Year, row.Month)
		period.MeetingStatus = row.Status
	}

	for _, row := range tasks {
		period := getPeriod(row.Year, row.Month)
		switch {
		case row.TaskCount == 0:
			period.TaskCompletionStatus = "empty"
		case row.DoneTaskCount < row.TaskCount:
			period.TaskCompletionStatus = "in_progress"
		case row.ApprovedApprovals < row.RequiredApprovals:
			period.TaskCompletionStatus = "awaiting_approval"
		default:
			period.TaskCompletionStatus = "ready"
		}
	}

	for _, row := range announcements {
		period := getPeriod(row.Year, row.Month)
		period.AnnouncementStatus = row.Status
	}

	results := make([]domain.WorkflowPeriod, 0, len(periods))
	for _, period := range periods {
		period.NextStageReady = period.MeetingStatus == domain.MeetingStatusCompleted || period.TaskCompletionStatus == "ready"
		results = append(results, *period)
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Year == results[j].Year {
			return results[i].Month > results[j].Month
		}
		return results[i].Year > results[j].Year
	})
	return results, nil
}

func (s *Service) GetMeeting(ctx context.Context, actor domain.Actor, year, month int32) (domain.Meeting, error) {
	row, err := s.store.Queries().GetMeetingByPeriod(ctx, db.GetMeetingByPeriodParams{
		OrganizationID: pgUUID(actor.OrganizationID),
		Year:           year,
		Month:          month,
	})
	if err != nil {
		return domain.Meeting{}, toAppError("meeting not found", err)
	}
	return meetingFromDB(row), nil
}

func (s *Service) PutMeeting(ctx context.Context, actor domain.Actor, input PutMeetingInput) (domain.Meeting, error) {
	if !canWrite(actor.Role) {
		return domain.Meeting{}, domain.NewAppError(domain.ErrorCodeForbidden, "write access denied", nil)
	}
	if err := validatePeriod(input.Year, input.Month); err != nil {
		return domain.Meeting{}, err
	}

	tx, q, err := s.store.Begin(ctx)
	if err != nil {
		return domain.Meeting{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to start transaction", err)
	}
	defer rollbackTx(ctx, tx)

	var before any
	existing, err := q.GetMeetingByPeriod(ctx, db.GetMeetingByPeriodParams{
		OrganizationID: pgUUID(actor.OrganizationID),
		Year:           input.Year,
		Month:          input.Month,
	})
	if err == nil {
		before = meetingFromDB(existing)
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return domain.Meeting{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to load meeting", err)
	}

	row, err := q.UpsertMeeting(ctx, db.UpsertMeetingParams{
		OrganizationID: pgUUID(actor.OrganizationID),
		Year:           input.Year,
		Month:          input.Month,
		ScheduledAt:    timestamptz(input.ScheduledAt),
		MeetingUrl:     text(input.MeetingURL),
		Notes:          text(input.Notes),
		Status:         input.Status,
		UpdatedBy:      pgUUID(actor.UserID),
	})
	if err != nil {
		return domain.Meeting{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to upsert meeting", err)
	}

	result := meetingFromDB(row)
	if err := audit.WriteUser(ctx, q, actor, appctx.RequestIPFromContext(ctx), "meeting.upsert", "meeting", result.ID, before, result); err != nil {
		return domain.Meeting{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to write audit log", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Meeting{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to commit meeting", err)
	}
	return result, nil
}

func (s *Service) ListTasks(ctx context.Context, actor domain.Actor, year, month int32) ([]domain.Task, error) {
	if err := validatePeriod(year, month); err != nil {
		return nil, err
	}

	q := s.store.Queries()
	rows, err := q.ListTasksByPeriod(ctx, db.ListTasksByPeriodParams{
		OrganizationID: pgUUID(actor.OrganizationID),
		Year:           year,
		Month:          month,
	})
	if err != nil {
		return nil, domain.NewAppError(domain.ErrorCodeInternal, "failed to list tasks", err)
	}
	return s.attachApprovals(ctx, q, rows)
}

func (s *Service) CreateTask(ctx context.Context, actor domain.Actor, input CreateTaskInput) (domain.Task, error) {
	if !canWrite(actor.Role) {
		return domain.Task{}, domain.NewAppError(domain.ErrorCodeForbidden, "write access denied", nil)
	}
	if err := validatePeriod(input.Year, input.Month); err != nil {
		return domain.Task{}, err
	}

	approverIDs, err := normalizeUniqueUUIDs(input.ApproverIDs)
	if err != nil {
		return domain.Task{}, err
	}

	tx, q, err := s.store.Begin(ctx)
	if err != nil {
		return domain.Task{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to start transaction", err)
	}
	defer rollbackTx(ctx, tx)

	if err := ensureMembers(ctx, q, actor.OrganizationID, approverIDs); err != nil {
		return domain.Task{}, err
	}
	if err := ensureOptionalMember(ctx, q, actor.OrganizationID, input.AssigneeID); err != nil {
		return domain.Task{}, err
	}

	row, err := q.CreateTask(ctx, db.CreateTaskParams{
		OrganizationID: pgUUID(actor.OrganizationID),
		Year:           input.Year,
		Month:          input.Month,
		Title:          input.Title,
		Status:         input.Status,
		DueDate:        date(input.DueDate),
		ReferenceUrl:   text(input.ReferenceURL),
		AssigneeID:     uuidPtr(input.AssigneeID),
		CreatedBy:      pgUUID(actor.UserID),
	})
	if err != nil {
		return domain.Task{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to create task", err)
	}

	if err := upsertTaskApprovals(ctx, q, uuidFromPG(row.ID), nil, approverIDs); err != nil {
		return domain.Task{}, err
	}

	task, err := s.loadTask(ctx, q, actor.OrganizationID, uuidFromPG(row.ID))
	if err != nil {
		return domain.Task{}, err
	}

	if err := audit.WriteUser(ctx, q, actor, appctx.RequestIPFromContext(ctx), "task.create", "task", task.ID, nil, task); err != nil {
		return domain.Task{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to write audit log", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Task{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to commit task", err)
	}
	return task, nil
}

func (s *Service) UpdateTask(ctx context.Context, actor domain.Actor, input UpdateTaskInput) (domain.Task, error) {
	if !canWrite(actor.Role) {
		return domain.Task{}, domain.NewAppError(domain.ErrorCodeForbidden, "write access denied", nil)
	}

	approverIDs, err := normalizeUniqueUUIDs(input.ApproverIDs)
	if err != nil {
		return domain.Task{}, err
	}

	tx, q, err := s.store.Begin(ctx)
	if err != nil {
		return domain.Task{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to start transaction", err)
	}
	defer rollbackTx(ctx, tx)

	before, err := s.loadTask(ctx, q, actor.OrganizationID, input.ID)
	if err != nil {
		return domain.Task{}, err
	}

	if err := ensureMembers(ctx, q, actor.OrganizationID, approverIDs); err != nil {
		return domain.Task{}, err
	}
	if err := ensureOptionalMember(ctx, q, actor.OrganizationID, input.AssigneeID); err != nil {
		return domain.Task{}, err
	}

	updatedRow, err := q.UpdateTask(ctx, db.UpdateTaskParams{
		OrganizationID: pgUUID(actor.OrganizationID),
		ID:             pgUUID(input.ID),
		Title:          input.Title,
		Status:         input.Status,
		DueDate:        date(input.DueDate),
		ReferenceUrl:   text(input.ReferenceURL),
		AssigneeID:     uuidPtr(input.AssigneeID),
		Version:        input.Version,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Task{}, domain.NewAppError(domain.ErrorCodeConflict, "task version conflict", err)
		}
		return domain.Task{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to update task", err)
	}

	existingApprovals, err := q.ListTaskApprovalsByTaskID(ctx, pgUUID(input.ID))
	if err != nil {
		return domain.Task{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to load task approvals", err)
	}
	approvedMap := map[uuid.UUID]*time.Time{}
	for _, approval := range existingApprovals {
		approverID := uuidFromPG(approval.ApproverUserID)
		if approval.ApprovedAt.Valid {
			approvedAt := approval.ApprovedAt.Time
			approvedMap[approverID] = &approvedAt
		}
	}

	if err := upsertTaskApprovals(ctx, q, uuidFromPG(updatedRow.ID), approvedMap, approverIDs); err != nil {
		return domain.Task{}, err
	}

	after, err := s.loadTask(ctx, q, actor.OrganizationID, input.ID)
	if err != nil {
		return domain.Task{}, err
	}

	if err := audit.WriteUser(ctx, q, actor, appctx.RequestIPFromContext(ctx), "task.update", "task", after.ID, before, after); err != nil {
		return domain.Task{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to write audit log", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Task{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to commit task", err)
	}
	return after, nil
}

func (s *Service) DeleteTask(ctx context.Context, actor domain.Actor, id uuid.UUID, version int32) error {
	if !canWrite(actor.Role) {
		return domain.NewAppError(domain.ErrorCodeForbidden, "write access denied", nil)
	}

	tx, q, err := s.store.Begin(ctx)
	if err != nil {
		return domain.NewAppError(domain.ErrorCodeInternal, "failed to start transaction", err)
	}
	defer rollbackTx(ctx, tx)

	before, err := s.loadTask(ctx, q, actor.OrganizationID, id)
	if err != nil {
		return err
	}

	if _, err := q.DeleteTask(ctx, db.DeleteTaskParams{
		OrganizationID: pgUUID(actor.OrganizationID),
		ID:             pgUUID(id),
		Version:        version,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.NewAppError(domain.ErrorCodeConflict, "task version conflict", err)
		}
		return domain.NewAppError(domain.ErrorCodeInternal, "failed to delete task", err)
	}

	after := map[string]any{"id": id.String(), "deleted": true}
	if err := audit.WriteUser(ctx, q, actor, appctx.RequestIPFromContext(ctx), "task.delete", "task", id, before, after); err != nil {
		return domain.NewAppError(domain.ErrorCodeInternal, "failed to write audit log", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.NewAppError(domain.ErrorCodeInternal, "failed to commit delete", err)
	}
	return nil
}

func (s *Service) ApproveTask(ctx context.Context, actor domain.Actor, id uuid.UUID) (domain.Task, error) {
	tx, q, err := s.store.Begin(ctx)
	if err != nil {
		return domain.Task{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to start transaction", err)
	}
	defer rollbackTx(ctx, tx)

	before, err := s.loadTask(ctx, q, actor.OrganizationID, id)
	if err != nil {
		return domain.Task{}, err
	}
	if before.Status != domain.TaskStatusDone {
		return domain.Task{}, domain.NewAppError(domain.ErrorCodeConflict, "task must be done before approval", nil)
	}

	approvalRows, err := q.ListTaskApprovalsByTaskID(ctx, pgUUID(id))
	if err != nil {
		return domain.Task{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to load task approvals", err)
	}

	found := false
	for _, approval := range approvalRows {
		if uuidFromPG(approval.ApproverUserID) != actor.UserID {
			continue
		}
		found = true
		if approval.ApprovedAt.Valid {
			return domain.Task{}, domain.NewAppError(domain.ErrorCodeConflict, "task already approved by this approver", nil)
		}
		break
	}
	if !found {
		return domain.Task{}, domain.NewAppError(domain.ErrorCodeForbidden, "user is not an approver for this task", nil)
	}

	if _, err := q.ApproveTask(ctx, db.ApproveTaskParams{
		TaskID:         pgUUID(id),
		ApproverUserID: pgUUID(actor.UserID),
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Task{}, domain.NewAppError(domain.ErrorCodeConflict, "task already approved", err)
		}
		return domain.Task{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to approve task", err)
	}

	after, err := s.loadTask(ctx, q, actor.OrganizationID, id)
	if err != nil {
		return domain.Task{}, err
	}

	if err := audit.WriteUser(ctx, q, actor, appctx.RequestIPFromContext(ctx), "task.approve", "task", id, before, after); err != nil {
		return domain.Task{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to write audit log", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Task{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to commit approval", err)
	}
	return after, nil
}

func (s *Service) GetAnnouncement(ctx context.Context, actor domain.Actor, year, month int32) (domain.Announcement, error) {
	row, err := s.store.Queries().GetAnnouncementByPeriod(ctx, db.GetAnnouncementByPeriodParams{
		OrganizationID: pgUUID(actor.OrganizationID),
		Year:           year,
		Month:          month,
	})
	if err != nil {
		return domain.Announcement{}, toAppError("announcement not found", err)
	}
	return announcementFromDB(row), nil
}

func (s *Service) PutAnnouncement(ctx context.Context, actor domain.Actor, input PutAnnouncementInput) (domain.Announcement, error) {
	if !canWrite(actor.Role) {
		return domain.Announcement{}, domain.NewAppError(domain.ErrorCodeForbidden, "write access denied", nil)
	}
	if err := validatePeriod(input.Year, input.Month); err != nil {
		return domain.Announcement{}, err
	}

	tx, q, err := s.store.Begin(ctx)
	if err != nil {
		return domain.Announcement{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to start transaction", err)
	}
	defer rollbackTx(ctx, tx)

	var before any
	existing, err := q.GetAnnouncementByPeriod(ctx, db.GetAnnouncementByPeriodParams{
		OrganizationID: pgUUID(actor.OrganizationID),
		Year:           input.Year,
		Month:          input.Month,
	})
	if err == nil {
		before = announcementFromDB(existing)
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return domain.Announcement{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to load announcement", err)
	}

	row, err := q.UpsertAnnouncementDraft(ctx, db.UpsertAnnouncementDraftParams{
		OrganizationID: pgUUID(actor.OrganizationID),
		Year:           input.Year,
		Month:          input.Month,
		Body:           input.Body,
		PublishChannel: text(input.PublishChannel),
		UpdatedBy:      pgUUID(actor.UserID),
	})
	if err != nil {
		return domain.Announcement{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to upsert announcement", err)
	}

	result := announcementFromDB(row)
	if err := audit.WriteUser(ctx, q, actor, appctx.RequestIPFromContext(ctx), "announcement.upsert", "announcement", result.ID, before, result); err != nil {
		return domain.Announcement{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to write audit log", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Announcement{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to commit announcement", err)
	}
	return result, nil
}

func (s *Service) PublishAnnouncement(ctx context.Context, actor domain.Actor, input PublishAnnouncementInput) (domain.Announcement, error) {
	if !canWrite(actor.Role) {
		return domain.Announcement{}, domain.NewAppError(domain.ErrorCodeForbidden, "write access denied", nil)
	}
	if strings.TrimSpace(input.PublishChannel) == "" {
		return domain.Announcement{}, domain.NewAppError(domain.ErrorCodeValidation, "publish_channel is required", nil)
	}

	tx, q, err := s.store.Begin(ctx)
	if err != nil {
		return domain.Announcement{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to start transaction", err)
	}
	defer rollbackTx(ctx, tx)

	beforeRow, err := q.GetAnnouncementByIDForOrg(ctx, db.GetAnnouncementByIDForOrgParams{
		OrganizationID: pgUUID(actor.OrganizationID),
		ID:             pgUUID(input.ID),
	})
	if err != nil {
		return domain.Announcement{}, toAppError("announcement not found", err)
	}
	before := announcementFromDB(beforeRow)
	if before.Status == domain.AnnouncementStatusPublishRequested {
		return domain.Announcement{}, domain.NewAppError(domain.ErrorCodeConflict, "announcement publish already requested", nil)
	}

	row, err := q.RequestAnnouncementPublish(ctx, db.RequestAnnouncementPublishParams{
		OrganizationID: pgUUID(actor.OrganizationID),
		ID:             pgUUID(input.ID),
		PublishChannel: text(&input.PublishChannel),
		UpdatedBy:      pgUUID(actor.UserID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Announcement{}, domain.NewAppError(domain.ErrorCodeNotFound, "announcement not found", err)
		}
		return domain.Announcement{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to request announcement publish", err)
	}

	result := announcementFromDB(row)
	if err := audit.WriteUser(ctx, q, actor, appctx.RequestIPFromContext(ctx), "announcement.publish_request", "announcement", result.ID, before, result); err != nil {
		return domain.Announcement{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to write audit log", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Announcement{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to commit publish request", err)
	}
	return result, nil
}

func (s *Service) ListPublishRequests(ctx context.Context) ([]domain.PublishRequest, error) {
	rows, err := s.store.Queries().ListPendingAnnouncementPublishRequests(ctx)
	if err != nil {
		return nil, domain.NewAppError(domain.ErrorCodeInternal, "failed to list publish requests", err)
	}

	requests := make([]domain.PublishRequest, 0, len(rows))
	for _, row := range rows {
		if !row.PublishChannel.Valid {
			continue
		}
		requests = append(requests, domain.PublishRequest{
			AnnouncementID: uuidFromPG(row.ID),
			Year:           row.Year,
			Month:          row.Month,
			Body:           row.Body,
			PublishChannel: row.PublishChannel.String,
		})
	}
	return requests, nil
}

func (s *Service) CompletePublishRequest(ctx context.Context, id uuid.UUID, discordMessageID string) (domain.Announcement, error) {
	tx, q, err := s.store.Begin(ctx)
	if err != nil {
		return domain.Announcement{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to start transaction", err)
	}
	defer rollbackTx(ctx, tx)

	beforeRow, err := q.GetAnnouncementByID(ctx, pgUUID(id))
	if err != nil {
		return domain.Announcement{}, toAppError("announcement not found", err)
	}
	before := announcementFromDB(beforeRow)

	row, err := q.CompleteAnnouncementPublish(ctx, db.CompleteAnnouncementPublishParams{
		ID:               pgUUID(id),
		DiscordMessageID: text(&discordMessageID),
	})
	if err != nil {
		return domain.Announcement{}, toAppError("announcement not found", err)
	}

	result := announcementFromDB(row)
	if err := audit.WriteService(ctx, q, uuidFromPG(row.OrganizationID), "discord-bot", appctx.RequestIPFromContext(ctx), "announcement.publish_complete", "announcement", result.ID, before, result); err != nil {
		return domain.Announcement{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to write audit log", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Announcement{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to commit publish complete", err)
	}
	return result, nil
}

func (s *Service) FailPublishRequest(ctx context.Context, id uuid.UUID, failure string) (domain.Announcement, error) {
	tx, q, err := s.store.Begin(ctx)
	if err != nil {
		return domain.Announcement{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to start transaction", err)
	}
	defer rollbackTx(ctx, tx)

	beforeRow, err := q.GetAnnouncementByID(ctx, pgUUID(id))
	if err != nil {
		return domain.Announcement{}, toAppError("announcement not found", err)
	}
	before := announcementFromDB(beforeRow)

	row, err := q.FailAnnouncementPublish(ctx, db.FailAnnouncementPublishParams{
		ID:        pgUUID(id),
		LastError: text(&failure),
	})
	if err != nil {
		return domain.Announcement{}, toAppError("announcement not found", err)
	}

	result := announcementFromDB(row)
	if err := audit.WriteService(ctx, q, uuidFromPG(row.OrganizationID), "discord-bot", appctx.RequestIPFromContext(ctx), "announcement.publish_fail", "announcement", result.ID, before, result); err != nil {
		return domain.Announcement{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to write audit log", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Announcement{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to commit publish failure", err)
	}
	return result, nil
}

func (s *Service) loadTask(ctx context.Context, q *db.Queries, organizationID uuid.UUID, taskID uuid.UUID) (domain.Task, error) {
	row, err := q.GetTaskByID(ctx, db.GetTaskByIDParams{
		OrganizationID: pgUUID(organizationID),
		ID:             pgUUID(taskID),
	})
	if err != nil {
		return domain.Task{}, toAppError("task not found", err)
	}
	approvals, err := q.ListTaskApprovalsByTaskID(ctx, pgUUID(taskID))
	if err != nil {
		return domain.Task{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to load task approvals", err)
	}
	return taskFromDB(row, approvals), nil
}

func (s *Service) attachApprovals(ctx context.Context, q *db.Queries, rows []db.Task) ([]domain.Task, error) {
	if len(rows) == 0 {
		return []domain.Task{}, nil
	}

	ids := make([]pgtype.UUID, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	approvals, err := q.ListTaskApprovalsByTaskIDs(ctx, ids)
	if err != nil {
		return nil, domain.NewAppError(domain.ErrorCodeInternal, "failed to list task approvals", err)
	}

	grouped := map[uuid.UUID][]db.TaskApproval{}
	for _, approval := range approvals {
		taskID := uuidFromPG(approval.TaskID)
		grouped[taskID] = append(grouped[taskID], approval)
	}

	results := make([]domain.Task, 0, len(rows))
	for _, row := range rows {
		results = append(results, taskFromDB(row, grouped[uuidFromPG(row.ID)]))
	}
	return results, nil
}

func ensureMembers(ctx context.Context, q *db.Queries, organizationID uuid.UUID, userIDs []uuid.UUID) error {
	for _, userID := range userIDs {
		if _, err := q.GetOrganizationMember(ctx, db.GetOrganizationMemberParams{
			OrganizationID: pgUUID(organizationID),
			UserID:         pgUUID(userID),
		}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.NewAppError(domain.ErrorCodeValidation, fmt.Sprintf("user %s is not a member of the organization", userID), err)
			}
			return domain.NewAppError(domain.ErrorCodeInternal, "failed to validate organization member", err)
		}
	}
	return nil
}

func ensureOptionalMember(ctx context.Context, q *db.Queries, organizationID uuid.UUID, userID *uuid.UUID) error {
	if userID == nil {
		return nil
	}
	return ensureMembers(ctx, q, organizationID, []uuid.UUID{*userID})
}

func upsertTaskApprovals(ctx context.Context, q *db.Queries, taskID uuid.UUID, approvedMap map[uuid.UUID]*time.Time, approverIDs []uuid.UUID) error {
	if err := q.DeleteTaskApprovalsByTaskID(ctx, pgUUID(taskID)); err != nil {
		return domain.NewAppError(domain.ErrorCodeInternal, "failed to replace task approvals", err)
	}
	for _, approverID := range approverIDs {
		approvedAt := timestamptz(nil)
		if approvedMap != nil {
			if existing, ok := approvedMap[approverID]; ok {
				approvedAt = timestamptz(existing)
			}
		}
		if err := q.CreateTaskApprovalWithState(ctx, db.CreateTaskApprovalWithStateParams{
			TaskID:         pgUUID(taskID),
			ApproverUserID: pgUUID(approverID),
			ApprovedAt:     approvedAt,
		}); err != nil {
			return domain.NewAppError(domain.ErrorCodeInternal, "failed to create task approval", err)
		}
	}
	return nil
}

func rollbackTx(ctx context.Context, tx pgx.Tx) {
	_ = tx.Rollback(ctx)
}

func canWrite(role string) bool {
	return role == domain.RoleOwner || role == domain.RoleEditor
}

func validatePeriod(year, month int32) error {
	if year < 2000 || year > 2100 {
		return domain.NewAppError(domain.ErrorCodeValidation, "year must be between 2000 and 2100", nil)
	}
	if month < 1 || month > 12 {
		return domain.NewAppError(domain.ErrorCodeValidation, "month must be between 1 and 12", nil)
	}
	return nil
}

func normalizeUniqueUUIDs(values []uuid.UUID) ([]uuid.UUID, error) {
	seen := map[uuid.UUID]struct{}{}
	result := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		if value == uuid.Nil {
			return nil, domain.NewAppError(domain.ErrorCodeValidation, "uuid must not be nil", nil)
		}
		if _, exists := seen[value]; exists {
			return nil, domain.NewAppError(domain.ErrorCodeValidation, "duplicate uuid is not allowed", nil)
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result, nil
}

func toAppError(message string, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.NewAppError(domain.ErrorCodeNotFound, message, err)
	}
	return domain.NewAppError(domain.ErrorCodeInternal, message, err)
}

func taskFromDB(row db.Task, approvals []db.TaskApproval) domain.Task {
	approverIDs := make([]uuid.UUID, 0, len(approvals))
	approvedIDs := make([]uuid.UUID, 0, len(approvals))
	for _, approval := range approvals {
		approverID := uuidFromPG(approval.ApproverUserID)
		approverIDs = append(approverIDs, approverID)
		if approval.ApprovedAt.Valid {
			approvedIDs = append(approvedIDs, approverID)
		}
	}

	return domain.Task{
		ID:                  uuidFromPG(row.ID),
		Title:               row.Title,
		Status:              row.Status,
		DueDate:             datePtr(row.DueDate),
		ReferenceURL:        textPtr(row.ReferenceUrl),
		AssigneeID:          uuidPtrFromPG(row.AssigneeID),
		CreatedBy:           uuidFromPG(row.CreatedBy),
		ApproverIDs:         approverIDs,
		ApprovedApproverIDs: approvedIDs,
		ApprovalState:       domain.ComputeApprovalState(len(approverIDs), len(approvedIDs)),
		Version:             row.Version,
		Year:                row.Year,
		Month:               row.Month,
	}
}

func meetingFromDB(row db.Meeting) domain.Meeting {
	return domain.Meeting{
		ID:          uuidFromPG(row.ID),
		Year:        row.Year,
		Month:       row.Month,
		ScheduledAt: timestamptzPtr(row.ScheduledAt),
		MeetingURL:  textPtr(row.MeetingUrl),
		Notes:       textPtr(row.Notes),
		Status:      row.Status,
	}
}

func announcementFromDB(row db.Announcement) domain.Announcement {
	return domain.Announcement{
		ID:               uuidFromPG(row.ID),
		Year:             row.Year,
		Month:            row.Month,
		Body:             row.Body,
		Status:           row.Status,
		PublishChannel:   textPtr(row.PublishChannel),
		DiscordMessageID: textPtr(row.DiscordMessageID),
		LastError:        textPtr(row.LastError),
	}
}

func pgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte(id), Valid: id != uuid.Nil}
}

func uuidPtr(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}
	return pgUUID(*id)
}

func uuidFromPG(value pgtype.UUID) uuid.UUID {
	if !value.Valid {
		return uuid.Nil
	}
	return uuid.UUID(value.Bytes)
}

func uuidPtrFromPG(value pgtype.UUID) *uuid.UUID {
	if !value.Valid {
		return nil
	}
	id := uuid.UUID(value.Bytes)
	return &id
}

func text(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *value, Valid: true}
}

func textPtr(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	result := value.String
	return &result
}

func date(value *time.Time) pgtype.Date {
	if value == nil {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: *value, Valid: true}
}

func datePtr(value pgtype.Date) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}

func timestamptz(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *value, Valid: true}
}

func timestamptzPtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}
