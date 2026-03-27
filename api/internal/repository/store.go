package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"firebase.google.com/go/v4/db"
	"github.com/google/uuid"

	"github.com/dokkiitech/ashiato/api/internal/domain"
)

// ErrNotFound is returned when a document is not found.
var ErrNotFound = errors.New("document not found")

// Store wraps a Realtime Database client and provides domain-specific data operations.
type Store struct {
	client *db.Client
}

func New(client *db.Client) *Store {
	return &Store{client: client}
}

// --- Organizations ---

type OrganizationDoc struct {
	Slug      string `json:"slug"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func (s *Store) EnsureOrganization(ctx context.Context, slug, name string) (uuid.UUID, OrganizationDoc, error) {
	ref := s.client.NewRef("organizations")
	var all map[string]OrganizationDoc
	if err := ref.Get(ctx, &all); err != nil {
		return uuid.Nil, OrganizationDoc{}, err
	}
	for idStr, org := range all {
		if org.Slug == slug {
			id, err := uuid.Parse(idStr)
			if err != nil {
				return uuid.Nil, OrganizationDoc{}, err
			}
			return id, org, nil
		}
	}

	id := uuid.New()
	now := time.Now().UTC().Format(time.RFC3339)
	org := OrganizationDoc{Slug: slug, Name: name, CreatedAt: now, UpdatedAt: now}
	if err := ref.Child(id.String()).Set(ctx, org); err != nil {
		return uuid.Nil, OrganizationDoc{}, err
	}
	return id, org, nil
}

func (s *Store) GetOrganizationByID(ctx context.Context, id uuid.UUID) (OrganizationDoc, error) {
	var org OrganizationDoc
	if err := s.client.NewRef("organizations").Child(id.String()).Get(ctx, &org); err != nil {
		return OrganizationDoc{}, err
	}
	if org.Slug == "" {
		return OrganizationDoc{}, ErrNotFound
	}
	return org, nil
}

// --- Users ---

type UserDoc struct {
	OIDCSubject string `json:"oidc_subject"`
	Email       string `json:"email"`
	Name        string `json:"name"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func (s *Store) UpsertUser(ctx context.Context, subject, email, name string) (uuid.UUID, UserDoc, error) {
	ref := s.client.NewRef("users")
	var all map[string]UserDoc
	if err := ref.Get(ctx, &all); err != nil && err.Error() != "unexpected end of JSON input" {
		return uuid.Nil, UserDoc{}, err
	}
	for idStr, user := range all {
		if user.OIDCSubject == subject {
			user.Email = email
			user.Name = name
			user.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			if err := ref.Child(idStr).Set(ctx, user); err != nil {
				return uuid.Nil, UserDoc{}, err
			}
			id, err := uuid.Parse(idStr)
			if err != nil {
				return uuid.Nil, UserDoc{}, err
			}
			return id, user, nil
		}
	}

	id := uuid.New()
	now := time.Now().UTC().Format(time.RFC3339)
	user := UserDoc{OIDCSubject: subject, Email: email, Name: name, CreatedAt: now, UpdatedAt: now}
	if err := ref.Child(id.String()).Set(ctx, user); err != nil {
		return uuid.Nil, UserDoc{}, err
	}
	return id, user, nil
}

// --- Organization Members ---

type MemberDoc struct {
	OrganizationID string `json:"organization_id"`
	UserID         string `json:"user_id"`
	Role           string `json:"role"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

func (s *Store) EnsureOrganizationMembership(ctx context.Context, orgID, userID uuid.UUID, role string) (MemberDoc, error) {
	ref := s.client.NewRef("organization_members")
	var all map[string]MemberDoc
	if err := ref.Get(ctx, &all); err != nil && err.Error() != "unexpected end of JSON input" {
		return MemberDoc{}, err
	}
	for _, member := range all {
		if member.OrganizationID == orgID.String() && member.UserID == userID.String() {
			return member, nil
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	member := MemberDoc{
		OrganizationID: orgID.String(),
		UserID:         userID.String(),
		Role:           role,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	id := uuid.New()
	if err := ref.Child(id.String()).Set(ctx, member); err != nil {
		return MemberDoc{}, err
	}
	return member, nil
}

func (s *Store) ListOrganizationMembers(ctx context.Context, orgID uuid.UUID) ([]domain.Member, error) {
	ref := s.client.NewRef("organization_members")
	var all map[string]MemberDoc
	if err := ref.Get(ctx, &all); err != nil {
		return nil, err
	}

	var members []domain.Member
	for _, m := range all {
		if m.OrganizationID != orgID.String() {
			continue
		}
		userID, _ := uuid.Parse(m.UserID)
		var user UserDoc
		if err := s.client.NewRef("users").Child(m.UserID).Get(ctx, &user); err != nil {
			continue
		}
		if user.Email == "" {
			continue
		}
		members = append(members, domain.Member{
			ID:    userID,
			Email: user.Email,
			Name:  user.Name,
			Role:  m.Role,
		})
	}
	return members, nil
}

func (s *Store) GetOrganizationMember(ctx context.Context, orgID, userID uuid.UUID) (MemberDoc, error) {
	ref := s.client.NewRef("organization_members")
	var all map[string]MemberDoc
	if err := ref.Get(ctx, &all); err != nil {
		return MemberDoc{}, err
	}
	for _, m := range all {
		if m.OrganizationID == orgID.String() && m.UserID == userID.String() {
			return m, nil
		}
	}
	return MemberDoc{}, ErrNotFound
}

// --- Meetings ---

type MeetingDoc struct {
	OrganizationID string  `json:"organization_id"`
	Year           int32   `json:"year"`
	Month          int32   `json:"month"`
	ScheduledAt    *string `json:"scheduled_at,omitempty"`
	MeetingURL     *string `json:"meeting_url,omitempty"`
	Notes          *string `json:"notes,omitempty"`
	Status         string  `json:"status"`
	UpdatedBy      string  `json:"updated_by"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

func meetingDocID(orgID uuid.UUID, year, month int32) string {
	return fmt.Sprintf("%s_%d_%02d", orgID.String(), year, month)
}

func (s *Store) GetMeetingByPeriod(ctx context.Context, orgID uuid.UUID, year, month int32) (uuid.UUID, MeetingDoc, error) {
	docID := meetingDocID(orgID, year, month)
	var m MeetingDoc
	if err := s.client.NewRef("meetings").Child(docID).Get(ctx, &m); err != nil {
		return uuid.Nil, MeetingDoc{}, err
	}
	if m.OrganizationID == "" {
		return uuid.Nil, MeetingDoc{}, ErrNotFound
	}
	id, _ := uuid.Parse(docID)
	return id, m, nil
}

func (s *Store) UpsertMeeting(ctx context.Context, orgID uuid.UUID, year, month int32, scheduledAt *time.Time, meetingURL, notes *string, meetingStatus, updatedBy string) (uuid.UUID, MeetingDoc, error) {
	docID := meetingDocID(orgID, year, month)
	now := time.Now().UTC().Format(time.RFC3339)
	ref := s.client.NewRef("meetings").Child(docID)

	var m MeetingDoc
	_ = ref.Get(ctx, &m)

	m.OrganizationID = orgID.String()
	m.Year = year
	m.Month = month
	if scheduledAt != nil {
		s := scheduledAt.Format(time.RFC3339)
		m.ScheduledAt = &s
	} else {
		m.ScheduledAt = nil
	}
	m.MeetingURL = meetingURL
	m.Notes = notes
	m.Status = meetingStatus
	m.UpdatedBy = updatedBy
	m.UpdatedAt = now
	if m.CreatedAt == "" {
		m.CreatedAt = now
	}

	if err := ref.Set(ctx, m); err != nil {
		return uuid.Nil, MeetingDoc{}, err
	}
	id, _ := uuid.Parse(docID)
	return id, m, nil
}

func (s *Store) ListMeetingPeriods(ctx context.Context, orgID uuid.UUID) ([]MeetingDoc, error) {
	ref := s.client.NewRef("meetings")
	var all map[string]MeetingDoc
	if err := ref.Get(ctx, &all); err != nil {
		return nil, err
	}

	var results []MeetingDoc
	for _, m := range all {
		if m.OrganizationID == orgID.String() {
			results = append(results, m)
		}
	}
	return results, nil
}

// --- Tasks ---

type TaskDoc struct {
	OrganizationID string  `json:"organization_id"`
	Year           int32   `json:"year"`
	Month          int32   `json:"month"`
	Title          string  `json:"title"`
	Status         string  `json:"status"`
	DueDate        *string `json:"due_date,omitempty"`
	ReferenceURL   *string `json:"reference_url,omitempty"`
	AssigneeID     *string `json:"assignee_id,omitempty"`
	CreatedBy      string  `json:"created_by"`
	Version        int32   `json:"version"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
	DeletedAt      *string `json:"deleted_at,omitempty"`
}

func (s *Store) CreateTask(ctx context.Context, task TaskDoc) (uuid.UUID, TaskDoc, error) {
	id := uuid.New()
	now := time.Now().UTC().Format(time.RFC3339)
	task.Version = 1
	task.CreatedAt = now
	task.UpdatedAt = now
	if err := s.client.NewRef("tasks").Child(id.String()).Set(ctx, task); err != nil {
		return uuid.Nil, TaskDoc{}, err
	}
	return id, task, nil
}

func (s *Store) GetTaskByID(ctx context.Context, orgID, taskID uuid.UUID) (TaskDoc, error) {
	var t TaskDoc
	if err := s.client.NewRef("tasks").Child(taskID.String()).Get(ctx, &t); err != nil {
		return TaskDoc{}, err
	}
	if t.OrganizationID == "" {
		return TaskDoc{}, ErrNotFound
	}
	if t.OrganizationID != orgID.String() || t.DeletedAt != nil {
		return TaskDoc{}, ErrNotFound
	}
	return t, nil
}

func (s *Store) ListTasksByPeriod(ctx context.Context, orgID uuid.UUID, year, month int32) ([]uuid.UUID, []TaskDoc, error) {
	ref := s.client.NewRef("tasks")
	var all map[string]TaskDoc
	if err := ref.Get(ctx, &all); err != nil {
		return nil, nil, err
	}

	var ids []uuid.UUID
	var tasks []TaskDoc
	for idStr, t := range all {
		if t.OrganizationID == orgID.String() && t.Year == year && t.Month == month && t.DeletedAt == nil {
			id, _ := uuid.Parse(idStr)
			ids = append(ids, id)
			tasks = append(tasks, t)
		}
	}
	return ids, tasks, nil
}

func (s *Store) UpdateTask(ctx context.Context, orgID, taskID uuid.UUID, title, taskStatus string, dueDate *time.Time, referenceURL *string, assigneeID *uuid.UUID, expectedVersion int32) (TaskDoc, error) {
	ref := s.client.NewRef("tasks").Child(taskID.String())
	var t TaskDoc
	if err := ref.Get(ctx, &t); err != nil {
		return TaskDoc{}, err
	}
	if t.OrganizationID == "" {
		return TaskDoc{}, ErrNotFound
	}
	if t.OrganizationID != orgID.String() || t.DeletedAt != nil {
		return TaskDoc{}, ErrNotFound
	}
	if t.Version != expectedVersion {
		return TaskDoc{}, domain.NewAppError(domain.ErrorCodeConflict, "task version conflict", nil)
	}

	t.Title = title
	t.Status = taskStatus
	if dueDate != nil {
		s := dueDate.Format(time.RFC3339)
		t.DueDate = &s
	} else {
		t.DueDate = nil
	}
	t.ReferenceURL = referenceURL
	if assigneeID != nil {
		s := assigneeID.String()
		t.AssigneeID = &s
	} else {
		t.AssigneeID = nil
	}
	t.Version++
	t.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	if err := ref.Set(ctx, t); err != nil {
		return TaskDoc{}, err
	}
	return t, nil
}

func (s *Store) DeleteTask(ctx context.Context, orgID, taskID uuid.UUID, expectedVersion int32) error {
	ref := s.client.NewRef("tasks").Child(taskID.String())
	var t TaskDoc
	if err := ref.Get(ctx, &t); err != nil {
		return err
	}
	if t.OrganizationID == "" {
		return ErrNotFound
	}
	if t.OrganizationID != orgID.String() || t.DeletedAt != nil {
		return ErrNotFound
	}
	if t.Version != expectedVersion {
		return domain.NewAppError(domain.ErrorCodeConflict, "task version conflict", nil)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	t.DeletedAt = &now
	t.UpdatedAt = now
	return ref.Set(ctx, t)
}

func (s *Store) ListTaskPeriodSummaries(ctx context.Context, orgID uuid.UUID) ([]TaskPeriodSummary, error) {
	ref := s.client.NewRef("tasks")
	var all map[string]TaskDoc
	if err := ref.Get(ctx, &all); err != nil {
		return nil, err
	}

	type periodKey struct {
		year  int32
		month int32
	}

	summaryMap := map[periodKey]*TaskPeriodSummary{}
	for idStr, t := range all {
		if t.OrganizationID != orgID.String() || t.DeletedAt != nil {
			continue
		}
		k := periodKey{year: t.Year, month: t.Month}
		summary, ok := summaryMap[k]
		if !ok {
			summary = &TaskPeriodSummary{Year: t.Year, Month: t.Month}
			summaryMap[k] = summary
		}
		summary.TaskCount++
		if t.Status == domain.TaskStatusDone {
			summary.DoneTaskCount++
		}

		taskID, _ := uuid.Parse(idStr)
		approvals, _ := s.ListTaskApprovals(ctx, taskID)
		for _, a := range approvals {
			summary.RequiredApprovals++
			if a.ApprovedAt != nil {
				summary.ApprovedApprovals++
			}
		}
	}

	results := make([]TaskPeriodSummary, 0, len(summaryMap))
	for _, summary := range summaryMap {
		results = append(results, *summary)
	}
	return results, nil
}

type TaskPeriodSummary struct {
	Year              int32
	Month             int32
	TaskCount         int64
	DoneTaskCount     int64
	RequiredApprovals int64
	ApprovedApprovals int64
}

// --- Task Approvals ---

type TaskApprovalDoc struct {
	TaskID         string  `json:"task_id"`
	ApproverUserID string  `json:"approver_user_id"`
	ApprovedAt     *string `json:"approved_at,omitempty"`
	CreatedAt      string  `json:"created_at"`
}

func (s *Store) ListTaskApprovals(ctx context.Context, taskID uuid.UUID) ([]TaskApprovalDoc, error) {
	ref := s.client.NewRef("task_approvals")
	var all map[string]TaskApprovalDoc
	if err := ref.Get(ctx, &all); err != nil {
		return nil, nil
	}

	var results []TaskApprovalDoc
	for _, a := range all {
		if a.TaskID == taskID.String() {
			results = append(results, a)
		}
	}
	return results, nil
}

func (s *Store) ApproveTask(ctx context.Context, taskID, approverUserID uuid.UUID) error {
	ref := s.client.NewRef("task_approvals")
	var all map[string]TaskApprovalDoc
	if err := ref.Get(ctx, &all); err != nil {
		return ErrNotFound
	}

	for idStr, a := range all {
		if a.TaskID == taskID.String() && a.ApproverUserID == approverUserID.String() {
			if a.ApprovedAt != nil {
				return domain.NewAppError(domain.ErrorCodeConflict, "task already approved", nil)
			}
			now := time.Now().UTC().Format(time.RFC3339)
			a.ApprovedAt = &now
			return ref.Child(idStr).Set(ctx, a)
		}
	}
	return ErrNotFound
}

func (s *Store) ReplaceTaskApprovals(ctx context.Context, taskID uuid.UUID, approvedMap map[uuid.UUID]*time.Time, approverIDs []uuid.UUID) error {
	ref := s.client.NewRef("task_approvals")
	var all map[string]TaskApprovalDoc
	_ = ref.Get(ctx, &all)

	// Delete existing approvals for this task
	for idStr, a := range all {
		if a.TaskID == taskID.String() {
			if err := ref.Child(idStr).Delete(ctx); err != nil {
				return err
			}
		}
	}

	// Create new approvals
	now := time.Now().UTC().Format(time.RFC3339)
	for _, approverID := range approverIDs {
		approval := TaskApprovalDoc{
			TaskID:         taskID.String(),
			ApproverUserID: approverID.String(),
			CreatedAt:      now,
		}
		if approvedMap != nil {
			if existing, ok := approvedMap[approverID]; ok && existing != nil {
				s := existing.Format(time.RFC3339)
				approval.ApprovedAt = &s
			}
		}
		id := uuid.New()
		if err := ref.Child(id.String()).Set(ctx, approval); err != nil {
			return err
		}
	}
	return nil
}

// --- Announcements ---

type AnnouncementDoc struct {
	OrganizationID   string  `json:"organization_id"`
	Year             int32   `json:"year"`
	Month            int32   `json:"month"`
	Body             string  `json:"body"`
	Status           string  `json:"status"`
	PublishChannel   *string `json:"publish_channel,omitempty"`
	DiscordMessageID *string `json:"discord_message_id,omitempty"`
	LastError        *string `json:"last_error,omitempty"`
	UpdatedBy        string  `json:"updated_by"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}

func announcementDocID(orgID uuid.UUID, year, month int32) string {
	return fmt.Sprintf("%s_%d_%02d", orgID.String(), year, month)
}

func (s *Store) GetAnnouncementByPeriod(ctx context.Context, orgID uuid.UUID, year, month int32) (uuid.UUID, AnnouncementDoc, error) {
	docID := announcementDocID(orgID, year, month)
	var a AnnouncementDoc
	if err := s.client.NewRef("announcements").Child(docID).Get(ctx, &a); err != nil {
		return uuid.Nil, AnnouncementDoc{}, err
	}
	if a.OrganizationID == "" {
		return uuid.Nil, AnnouncementDoc{}, ErrNotFound
	}
	id, _ := uuid.Parse(docID)
	return id, a, nil
}

func (s *Store) GetAnnouncementByID(ctx context.Context, id uuid.UUID) (AnnouncementDoc, error) {
	var a AnnouncementDoc
	if err := s.client.NewRef("announcements").Child(id.String()).Get(ctx, &a); err != nil {
		return AnnouncementDoc{}, err
	}
	if a.OrganizationID == "" {
		return AnnouncementDoc{}, ErrNotFound
	}
	return a, nil
}

func (s *Store) GetAnnouncementByIDForOrg(ctx context.Context, orgID, announcementID uuid.UUID) (uuid.UUID, AnnouncementDoc, error) {
	a, err := s.GetAnnouncementByID(ctx, announcementID)
	if err != nil {
		return uuid.Nil, AnnouncementDoc{}, err
	}
	if a.OrganizationID != orgID.String() {
		return uuid.Nil, AnnouncementDoc{}, ErrNotFound
	}
	return announcementID, a, nil
}

func (s *Store) UpsertAnnouncementDraft(ctx context.Context, orgID uuid.UUID, year, month int32, body string, publishChannel *string, updatedBy string) (uuid.UUID, AnnouncementDoc, error) {
	docID := announcementDocID(orgID, year, month)
	now := time.Now().UTC().Format(time.RFC3339)
	ref := s.client.NewRef("announcements").Child(docID)

	var a AnnouncementDoc
	_ = ref.Get(ctx, &a)

	a.OrganizationID = orgID.String()
	a.Year = year
	a.Month = month
	a.Body = body
	a.Status = domain.AnnouncementStatusDraft
	a.PublishChannel = publishChannel
	a.UpdatedBy = updatedBy
	a.UpdatedAt = now
	if a.CreatedAt == "" {
		a.CreatedAt = now
	}

	if err := ref.Set(ctx, a); err != nil {
		return uuid.Nil, AnnouncementDoc{}, err
	}
	id, _ := uuid.Parse(docID)
	return id, a, nil
}

func (s *Store) RequestAnnouncementPublish(ctx context.Context, orgID, announcementID uuid.UUID, publishChannel *string, updatedBy string) (uuid.UUID, AnnouncementDoc, error) {
	ref := s.client.NewRef("announcements").Child(announcementID.String())
	var a AnnouncementDoc
	if err := ref.Get(ctx, &a); err != nil {
		return uuid.Nil, AnnouncementDoc{}, err
	}
	if a.OrganizationID == "" {
		return uuid.Nil, AnnouncementDoc{}, ErrNotFound
	}
	if a.OrganizationID != orgID.String() {
		return uuid.Nil, AnnouncementDoc{}, ErrNotFound
	}

	a.Status = domain.AnnouncementStatusPublishRequested
	if publishChannel != nil {
		a.PublishChannel = publishChannel
	}
	a.UpdatedBy = updatedBy
	a.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	if err := ref.Set(ctx, a); err != nil {
		return uuid.Nil, AnnouncementDoc{}, err
	}
	return announcementID, a, nil
}

func (s *Store) CompleteAnnouncementPublish(ctx context.Context, id uuid.UUID, discordMessageID string) (uuid.UUID, AnnouncementDoc, error) {
	ref := s.client.NewRef("announcements").Child(id.String())
	var a AnnouncementDoc
	if err := ref.Get(ctx, &a); err != nil {
		return uuid.Nil, AnnouncementDoc{}, err
	}
	if a.OrganizationID == "" {
		return uuid.Nil, AnnouncementDoc{}, ErrNotFound
	}

	a.Status = domain.AnnouncementStatusPublished
	a.DiscordMessageID = &discordMessageID
	a.LastError = nil
	a.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	if err := ref.Set(ctx, a); err != nil {
		return uuid.Nil, AnnouncementDoc{}, err
	}
	return id, a, nil
}

func (s *Store) FailAnnouncementPublish(ctx context.Context, id uuid.UUID, lastError string) (uuid.UUID, AnnouncementDoc, error) {
	ref := s.client.NewRef("announcements").Child(id.String())
	var a AnnouncementDoc
	if err := ref.Get(ctx, &a); err != nil {
		return uuid.Nil, AnnouncementDoc{}, err
	}
	if a.OrganizationID == "" {
		return uuid.Nil, AnnouncementDoc{}, ErrNotFound
	}

	a.Status = domain.AnnouncementStatusPublishFailed
	a.LastError = &lastError
	a.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	if err := ref.Set(ctx, a); err != nil {
		return uuid.Nil, AnnouncementDoc{}, err
	}
	return id, a, nil
}

func (s *Store) ListAnnouncementPeriods(ctx context.Context, orgID uuid.UUID) ([]AnnouncementDoc, error) {
	ref := s.client.NewRef("announcements")
	var all map[string]AnnouncementDoc
	if err := ref.Get(ctx, &all); err != nil {
		return nil, err
	}

	var results []AnnouncementDoc
	for _, a := range all {
		if a.OrganizationID == orgID.String() {
			results = append(results, a)
		}
	}
	return results, nil
}

func (s *Store) ListPendingPublishRequests(ctx context.Context) ([]uuid.UUID, []AnnouncementDoc, error) {
	ref := s.client.NewRef("announcements")
	var all map[string]AnnouncementDoc
	if err := ref.Get(ctx, &all); err != nil {
		return nil, nil, err
	}

	var ids []uuid.UUID
	var results []AnnouncementDoc
	for idStr, a := range all {
		if a.Status == domain.AnnouncementStatusPublishRequested {
			id, _ := uuid.Parse(idStr)
			ids = append(ids, id)
			results = append(results, a)
		}
	}
	return ids, results, nil
}

// --- Audit Logs ---

type AuditLogDoc struct {
	OrganizationID string `json:"organization_id"`
	ActorUserID    *string `json:"actor_user_id,omitempty"`
	ActorType      string `json:"actor_type"`
	ActorLabel     string `json:"actor_label"`
	Action         string `json:"action"`
	ResourceType   string `json:"resource_type"`
	ResourceID     string `json:"resource_id"`
	BeforeState    any    `json:"before_state,omitempty"`
	AfterState     any    `json:"after_state,omitempty"`
	Result         string `json:"result"`
	IP             string `json:"ip"`
	CreatedAt      string `json:"created_at"`
}

func (s *Store) CreateAuditLog(ctx context.Context, log AuditLogDoc) error {
	log.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	id := uuid.New()
	return s.client.NewRef("audit_logs").Child(id.String()).Set(ctx, log)
}

// --- Helper: parse UUID from potentially invalid strings ---

func ParseUUIDsFromStrings(values []string) []uuid.UUID {
	result := make([]uuid.UUID, 0, len(values))
	for _, v := range values {
		id, err := uuid.Parse(strings.TrimSpace(v))
		if err == nil {
			result = append(result, id)
		}
	}
	return result
}
