package usecase

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/dokkiitech/ashiato/api/internal/audit"
	"github.com/dokkiitech/ashiato/api/internal/config"
	"github.com/dokkiitech/ashiato/api/internal/discord"
	"github.com/dokkiitech/ashiato/api/internal/domain"
	appctx "github.com/dokkiitech/ashiato/api/internal/middleware"
	"github.com/dokkiitech/ashiato/api/internal/repository"
)

type Service struct {
	store       *repository.Store
	webhook     *discord.WebhookClient
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

func NewService(store *repository.Store, webhook *discord.WebhookClient, logger *slog.Logger, cfg config.Config) *Service {
	return &Service{
		store:       store,
		webhook:     webhook,
		logger:      logger,
		defaultName: cfg.DefaultOrgName,
		defaultSlug: cfg.DefaultOrgSlug,
		ownerEmails: cfg.OwnerEmails,
	}
}

func (s *Service) SyncPrincipal(ctx context.Context, principal domain.Principal) (domain.Actor, error) {
	orgID, org, err := s.store.EnsureOrganization(ctx, s.defaultSlug, s.defaultName)
	if err != nil {
		return domain.Actor{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to ensure organization", err)
	}
	_ = org

	userID, user, err := s.store.UpsertUser(ctx, principal.Subject, principal.Email, principal.Name)
	if err != nil {
		return domain.Actor{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to upsert user", err)
	}

	role := domain.RoleEditor
	if _, ok := s.ownerEmails[strings.ToLower(principal.Email)]; ok {
		role = domain.RoleOwner
	}

	member, err := s.store.EnsureOrganizationMembership(ctx, orgID, userID, role)
	if err != nil {
		return domain.Actor{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to ensure membership", err)
	}

	return domain.Actor{
		UserID:         userID,
		OrganizationID: orgID,
		Role:           member.Role,
		Subject:        user.OIDCSubject,
		Email:          user.Email,
		Name:           user.Name,
	}, nil
}

func (s *Service) GetMe(ctx context.Context, actor domain.Actor) (domain.Actor, domain.Organization, error) {
	org, err := s.store.GetOrganizationByID(ctx, actor.OrganizationID)
	if err != nil {
		return domain.Actor{}, domain.Organization{}, toAppError("organization not found", err)
	}

	return actor, domain.Organization{
		ID:   actor.OrganizationID,
		Slug: org.Slug,
		Name: org.Name,
	}, nil
}

func (s *Service) ListMembers(ctx context.Context, actor domain.Actor) ([]domain.Member, error) {
	members, err := s.store.ListOrganizationMembers(ctx, actor.OrganizationID)
	if err != nil {
		return nil, domain.NewAppError(domain.ErrorCodeInternal, "failed to list members", err)
	}
	return members, nil
}

func (s *Service) ListWorkflowPeriods(ctx context.Context, actor domain.Actor) ([]domain.WorkflowPeriod, error) {
	meetings, err := s.store.ListMeetingPeriods(ctx, actor.OrganizationID)
	if err != nil {
		return nil, domain.NewAppError(domain.ErrorCodeInternal, "failed to list meeting periods", err)
	}
	taskSummaries, err := s.store.ListTaskPeriodSummaries(ctx, actor.OrganizationID)
	if err != nil {
		return nil, domain.NewAppError(domain.ErrorCodeInternal, "failed to list task periods", err)
	}
	announcements, err := s.store.ListAnnouncementPeriods(ctx, actor.OrganizationID)
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

	for _, row := range taskSummaries {
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
	id, doc, err := s.store.GetMeetingByPeriod(ctx, actor.OrganizationID, year, month)
	if err != nil {
		return domain.Meeting{}, toAppError("meeting not found", err)
	}
	return meetingFromDoc(id, doc), nil
}

func (s *Service) PutMeeting(ctx context.Context, actor domain.Actor, input PutMeetingInput) (domain.Meeting, error) {
	if !canWrite(actor.Role) {
		return domain.Meeting{}, domain.NewAppError(domain.ErrorCodeForbidden, "write access denied", nil)
	}
	if err := validatePeriod(input.Year, input.Month); err != nil {
		return domain.Meeting{}, err
	}

	var before any
	existingID, existingDoc, err := s.store.GetMeetingByPeriod(ctx, actor.OrganizationID, input.Year, input.Month)
	if err == nil {
		before = meetingFromDoc(existingID, existingDoc)
	} else if !errors.Is(err, repository.ErrNotFound) {
		return domain.Meeting{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to load meeting", err)
	}

	id, doc, err := s.store.UpsertMeeting(ctx, actor.OrganizationID, input.Year, input.Month, input.ScheduledAt, input.MeetingURL, input.Notes, input.Status, actor.UserID.String())
	if err != nil {
		return domain.Meeting{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to upsert meeting", err)
	}

	result := meetingFromDoc(id, doc)
	if err := audit.WriteUser(ctx, s.store, actor, appctx.RequestIPFromContext(ctx), "meeting.upsert", "meeting", result.ID, before, result); err != nil {
		return domain.Meeting{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to write audit log", err)
	}
	return result, nil
}

func (s *Service) ListTasks(ctx context.Context, actor domain.Actor, year, month int32) ([]domain.Task, error) {
	if err := validatePeriod(year, month); err != nil {
		return nil, err
	}

	ids, docs, err := s.store.ListTasksByPeriod(ctx, actor.OrganizationID, year, month)
	if err != nil {
		return nil, domain.NewAppError(domain.ErrorCodeInternal, "failed to list tasks", err)
	}
	return s.attachApprovals(ctx, ids, docs)
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

	if err := s.ensureMembers(ctx, actor.OrganizationID, approverIDs); err != nil {
		return domain.Task{}, err
	}
	if err := s.ensureOptionalMember(ctx, actor.OrganizationID, input.AssigneeID); err != nil {
		return domain.Task{}, err
	}

	var assigneeIDStr *string
	if input.AssigneeID != nil {
		s := input.AssigneeID.String()
		assigneeIDStr = &s
	}

	taskDoc := repository.TaskDoc{
		OrganizationID: actor.OrganizationID.String(),
		Year:           input.Year,
		Month:          input.Month,
		Title:          input.Title,
		Status:         input.Status,
		DueDate:        timeToStringPtr(input.DueDate),
		ReferenceURL:   input.ReferenceURL,
		AssigneeID:     assigneeIDStr,
		CreatedBy:      actor.UserID.String(),
	}

	taskID, _, err := s.store.CreateTask(ctx, taskDoc)
	if err != nil {
		return domain.Task{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to create task", err)
	}

	if err := s.store.ReplaceTaskApprovals(ctx, taskID, nil, approverIDs); err != nil {
		return domain.Task{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to create task approvals", err)
	}

	task, err := s.loadTask(ctx, actor.OrganizationID, taskID)
	if err != nil {
		return domain.Task{}, err
	}

	if err := audit.WriteUser(ctx, s.store, actor, appctx.RequestIPFromContext(ctx), "task.create", "task", task.ID, nil, task); err != nil {
		return domain.Task{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to write audit log", err)
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

	before, err := s.loadTask(ctx, actor.OrganizationID, input.ID)
	if err != nil {
		return domain.Task{}, err
	}

	if err := s.ensureMembers(ctx, actor.OrganizationID, approverIDs); err != nil {
		return domain.Task{}, err
	}
	if err := s.ensureOptionalMember(ctx, actor.OrganizationID, input.AssigneeID); err != nil {
		return domain.Task{}, err
	}

	if _, err := s.store.UpdateTask(ctx, actor.OrganizationID, input.ID, input.Title, input.Status, input.DueDate, input.ReferenceURL, input.AssigneeID, input.Version); err != nil {
		return domain.Task{}, toAppError("failed to update task", err)
	}

	existingApprovals, err := s.store.ListTaskApprovals(ctx, input.ID)
	if err != nil {
		return domain.Task{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to load task approvals", err)
	}
	approvedMap := map[uuid.UUID]*time.Time{}
	for _, approval := range existingApprovals {
		approverID, _ := uuid.Parse(approval.ApproverUserID)
		if approval.ApprovedAt != nil {
			approvedMap[approverID] = stringPtrToTime(approval.ApprovedAt)
		}
	}

	if err := s.store.ReplaceTaskApprovals(ctx, input.ID, approvedMap, approverIDs); err != nil {
		return domain.Task{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to update task approvals", err)
	}

	after, err := s.loadTask(ctx, actor.OrganizationID, input.ID)
	if err != nil {
		return domain.Task{}, err
	}

	if err := audit.WriteUser(ctx, s.store, actor, appctx.RequestIPFromContext(ctx), "task.update", "task", after.ID, before, after); err != nil {
		return domain.Task{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to write audit log", err)
	}
	return after, nil
}

func (s *Service) DeleteTask(ctx context.Context, actor domain.Actor, id uuid.UUID, version int32) error {
	if !canWrite(actor.Role) {
		return domain.NewAppError(domain.ErrorCodeForbidden, "write access denied", nil)
	}

	before, err := s.loadTask(ctx, actor.OrganizationID, id)
	if err != nil {
		return err
	}

	if err := s.store.DeleteTask(ctx, actor.OrganizationID, id, version); err != nil {
		return toAppError("failed to delete task", err)
	}

	after := map[string]any{"id": id.String(), "deleted": true}
	if err := audit.WriteUser(ctx, s.store, actor, appctx.RequestIPFromContext(ctx), "task.delete", "task", id, before, after); err != nil {
		return domain.NewAppError(domain.ErrorCodeInternal, "failed to write audit log", err)
	}
	return nil
}

func (s *Service) ApproveTask(ctx context.Context, actor domain.Actor, id uuid.UUID) (domain.Task, error) {
	before, err := s.loadTask(ctx, actor.OrganizationID, id)
	if err != nil {
		return domain.Task{}, err
	}
	if before.Status != domain.TaskStatusDone {
		return domain.Task{}, domain.NewAppError(domain.ErrorCodeConflict, "task must be done before approval", nil)
	}

	approvalRows, err := s.store.ListTaskApprovals(ctx, id)
	if err != nil {
		return domain.Task{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to load task approvals", err)
	}

	found := false
	for _, approval := range approvalRows {
		approverID, _ := uuid.Parse(approval.ApproverUserID)
		if approverID != actor.UserID {
			continue
		}
		found = true
		if approval.ApprovedAt != nil {
			return domain.Task{}, domain.NewAppError(domain.ErrorCodeConflict, "task already approved by this approver", nil)
		}
		break
	}
	if !found {
		return domain.Task{}, domain.NewAppError(domain.ErrorCodeForbidden, "user is not an approver for this task", nil)
	}

	if err := s.store.ApproveTask(ctx, id, actor.UserID); err != nil {
		return domain.Task{}, toAppError("failed to approve task", err)
	}

	after, err := s.loadTask(ctx, actor.OrganizationID, id)
	if err != nil {
		return domain.Task{}, err
	}

	if err := audit.WriteUser(ctx, s.store, actor, appctx.RequestIPFromContext(ctx), "task.approve", "task", id, before, after); err != nil {
		return domain.Task{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to write audit log", err)
	}
	return after, nil
}

func (s *Service) GetAnnouncement(ctx context.Context, actor domain.Actor, year, month int32) (domain.Announcement, error) {
	id, doc, err := s.store.GetAnnouncementByPeriod(ctx, actor.OrganizationID, year, month)
	if err != nil {
		return domain.Announcement{}, toAppError("announcement not found", err)
	}
	return announcementFromDoc(id, doc), nil
}

func (s *Service) PutAnnouncement(ctx context.Context, actor domain.Actor, input PutAnnouncementInput) (domain.Announcement, error) {
	if !canWrite(actor.Role) {
		return domain.Announcement{}, domain.NewAppError(domain.ErrorCodeForbidden, "write access denied", nil)
	}
	if err := validatePeriod(input.Year, input.Month); err != nil {
		return domain.Announcement{}, err
	}

	var before any
	existingID, existingDoc, err := s.store.GetAnnouncementByPeriod(ctx, actor.OrganizationID, input.Year, input.Month)
	if err == nil {
		before = announcementFromDoc(existingID, existingDoc)
	} else if !errors.Is(err, repository.ErrNotFound) {
		return domain.Announcement{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to load announcement", err)
	}

	id, doc, err := s.store.UpsertAnnouncementDraft(ctx, actor.OrganizationID, input.Year, input.Month, input.Body, input.PublishChannel, actor.UserID.String())
	if err != nil {
		return domain.Announcement{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to upsert announcement", err)
	}

	result := announcementFromDoc(id, doc)
	if err := audit.WriteUser(ctx, s.store, actor, appctx.RequestIPFromContext(ctx), "announcement.upsert", "announcement", result.ID, before, result); err != nil {
		return domain.Announcement{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to write audit log", err)
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

	_, beforeDoc, err := s.store.GetAnnouncementByIDForOrg(ctx, actor.OrganizationID, input.ID)
	if err != nil {
		return domain.Announcement{}, toAppError("announcement not found", err)
	}
	before := announcementFromDoc(input.ID, beforeDoc)
	if before.Status == domain.AnnouncementStatusPublished {
		return domain.Announcement{}, domain.NewAppError(domain.ErrorCodeConflict, "announcement already published", nil)
	}

	// Send to Discord via Webhook.
	discordMessageID, err := s.webhook.Send(ctx, before.Body)
	if err != nil {
		s.logger.Error("discord webhook failed", "error", err, "announcement_id", input.ID)
		// Mark as failed.
		_, failDoc, failErr := s.store.FailAnnouncementPublish(ctx, input.ID, err.Error())
		if failErr != nil {
			return domain.Announcement{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to send to discord and failed to record error", err)
		}
		result := announcementFromDoc(input.ID, failDoc)
		_ = audit.WriteUser(ctx, s.store, actor, appctx.RequestIPFromContext(ctx), "announcement.publish_fail", "announcement", result.ID, before, result)
		return result, domain.NewAppError(domain.ErrorCodeInternal, "failed to send to discord: "+err.Error(), err)
	}

	// Mark as published with the Discord message ID.
	_, doc, err := s.store.CompleteAnnouncementPublish(ctx, input.ID, discordMessageID)
	if err != nil {
		return domain.Announcement{}, domain.NewAppError(domain.ErrorCodeInternal, "discord message sent but failed to update status", err)
	}

	result := announcementFromDoc(input.ID, doc)
	if err := audit.WriteUser(ctx, s.store, actor, appctx.RequestIPFromContext(ctx), "announcement.publish", "announcement", result.ID, before, result); err != nil {
		return domain.Announcement{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to write audit log", err)
	}
	return result, nil
}

// ListPublishRequests is kept for OpenAPI interface compatibility but returns empty.
func (s *Service) ListPublishRequests(_ context.Context) ([]domain.PublishRequest, error) {
	return []domain.PublishRequest{}, nil
}

// CompletePublishRequest is kept for OpenAPI interface compatibility but returns not found.
func (s *Service) CompletePublishRequest(_ context.Context, _ uuid.UUID, _ string) (domain.Announcement, error) {
	return domain.Announcement{}, domain.NewAppError(domain.ErrorCodeNotFound, "internal endpoints are deprecated; use webhook", nil)
}

// FailPublishRequest is kept for OpenAPI interface compatibility but returns not found.
func (s *Service) FailPublishRequest(_ context.Context, _ uuid.UUID, _ string) (domain.Announcement, error) {
	return domain.Announcement{}, domain.NewAppError(domain.ErrorCodeNotFound, "internal endpoints are deprecated; use webhook", nil)
}

// --- internal helpers ---

func (s *Service) loadTask(ctx context.Context, organizationID uuid.UUID, taskID uuid.UUID) (domain.Task, error) {
	doc, err := s.store.GetTaskByID(ctx, organizationID, taskID)
	if err != nil {
		return domain.Task{}, toAppError("task not found", err)
	}
	approvals, err := s.store.ListTaskApprovals(ctx, taskID)
	if err != nil {
		return domain.Task{}, domain.NewAppError(domain.ErrorCodeInternal, "failed to load task approvals", err)
	}
	return taskFromDoc(taskID, doc, approvals), nil
}

func (s *Service) attachApprovals(ctx context.Context, ids []uuid.UUID, docs []repository.TaskDoc) ([]domain.Task, error) {
	if len(docs) == 0 {
		return []domain.Task{}, nil
	}

	results := make([]domain.Task, 0, len(docs))
	for i, doc := range docs {
		approvals, err := s.store.ListTaskApprovals(ctx, ids[i])
		if err != nil {
			return nil, domain.NewAppError(domain.ErrorCodeInternal, "failed to list task approvals", err)
		}
		results = append(results, taskFromDoc(ids[i], doc, approvals))
	}
	return results, nil
}

func (s *Service) ensureMembers(ctx context.Context, organizationID uuid.UUID, userIDs []uuid.UUID) error {
	for _, userID := range userIDs {
		if _, err := s.store.GetOrganizationMember(ctx, organizationID, userID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return domain.NewAppError(domain.ErrorCodeValidation, "user "+userID.String()+" is not a member of the organization", err)
			}
			return domain.NewAppError(domain.ErrorCodeInternal, "failed to validate organization member", err)
		}
	}
	return nil
}

func (s *Service) ensureOptionalMember(ctx context.Context, organizationID uuid.UUID, userID *uuid.UUID) error {
	if userID == nil {
		return nil
	}
	return s.ensureMembers(ctx, organizationID, []uuid.UUID{*userID})
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
	if errors.Is(err, repository.ErrNotFound) {
		return domain.NewAppError(domain.ErrorCodeNotFound, message, err)
	}
	var appErr *domain.AppError
	if errors.As(err, &appErr) {
		return err
	}
	return domain.NewAppError(domain.ErrorCodeInternal, message, err)
}

func taskFromDoc(id uuid.UUID, doc repository.TaskDoc, approvals []repository.TaskApprovalDoc) domain.Task {
	approverIDs := make([]uuid.UUID, 0, len(approvals))
	approvedIDs := make([]uuid.UUID, 0, len(approvals))
	for _, approval := range approvals {
		approverID, _ := uuid.Parse(approval.ApproverUserID)
		approverIDs = append(approverIDs, approverID)
		if approval.ApprovedAt != nil {
			approvedIDs = append(approvedIDs, approverID)
		}
	}

	var assigneeID *uuid.UUID
	if doc.AssigneeID != nil {
		parsed, err := uuid.Parse(*doc.AssigneeID)
		if err == nil {
			assigneeID = &parsed
		}
	}
	createdBy, _ := uuid.Parse(doc.CreatedBy)

	return domain.Task{
		ID:                  id,
		Title:               doc.Title,
		Status:              doc.Status,
		DueDate:             stringPtrToTime(doc.DueDate),
		ReferenceURL:        doc.ReferenceURL,
		AssigneeID:          assigneeID,
		CreatedBy:           createdBy,
		ApproverIDs:         approverIDs,
		ApprovedApproverIDs: approvedIDs,
		ApprovalState:       domain.ComputeApprovalState(len(approverIDs), len(approvedIDs)),
		Version:             doc.Version,
		Year:                doc.Year,
		Month:               doc.Month,
	}
}

func meetingFromDoc(id uuid.UUID, doc repository.MeetingDoc) domain.Meeting {
	return domain.Meeting{
		ID:          id,
		Year:        doc.Year,
		Month:       doc.Month,
		ScheduledAt: stringPtrToTime(doc.ScheduledAt),
		MeetingURL:  doc.MeetingURL,
		Notes:       doc.Notes,
		Status:      doc.Status,
	}
}

func timeToStringPtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}

func stringPtrToTime(s *string) *time.Time {
	if s == nil {
		return nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return nil
	}
	return &t
}

func announcementFromDoc(id uuid.UUID, doc repository.AnnouncementDoc) domain.Announcement {
	return domain.Announcement{
		ID:               id,
		Year:             doc.Year,
		Month:            doc.Month,
		Body:             doc.Body,
		Status:           doc.Status,
		PublishChannel:   doc.PublishChannel,
		DiscordMessageID: doc.DiscordMessageID,
		LastError:        doc.LastError,
	}
}
