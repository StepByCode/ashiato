package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/dokkiitech/ashiato/api/internal/domain"
)

// ErrNotFound is returned when a document is not found.
var ErrNotFound = errors.New("document not found")

// Store wraps a Firestore client and provides domain-specific data operations.
type Store struct {
	client *firestore.Client
}

func New(client *firestore.Client) *Store {
	return &Store{client: client}
}

func (s *Store) Close() error {
	return s.client.Close()
}

// --- Organizations ---

type OrganizationDoc struct {
	Slug      string    `firestore:"slug"`
	Name      string    `firestore:"name"`
	CreatedAt time.Time `firestore:"created_at"`
	UpdatedAt time.Time `firestore:"updated_at"`
}

func (s *Store) EnsureOrganization(ctx context.Context, slug, name string) (uuid.UUID, OrganizationDoc, error) {
	col := s.client.Collection("organizations")
	iter := col.Where("slug", "==", slug).Limit(1).Documents(ctx)
	doc, err := iter.Next()
	if err == nil {
		var org OrganizationDoc
		if err := doc.DataTo(&org); err != nil {
			return uuid.Nil, OrganizationDoc{}, err
		}
		id, err := uuid.Parse(doc.Ref.ID)
		if err != nil {
			return uuid.Nil, OrganizationDoc{}, err
		}
		return id, org, nil
	}
	if !errors.Is(err, iterator.Done) {
		return uuid.Nil, OrganizationDoc{}, err
	}

	id := uuid.New()
	now := time.Now().UTC()
	org := OrganizationDoc{Slug: slug, Name: name, CreatedAt: now, UpdatedAt: now}
	if _, err := col.Doc(id.String()).Set(ctx, org); err != nil {
		return uuid.Nil, OrganizationDoc{}, err
	}
	return id, org, nil
}

func (s *Store) GetOrganizationByID(ctx context.Context, id uuid.UUID) (OrganizationDoc, error) {
	doc, err := s.client.Collection("organizations").Doc(id.String()).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return OrganizationDoc{}, ErrNotFound
		}
		return OrganizationDoc{}, err
	}
	var org OrganizationDoc
	if err := doc.DataTo(&org); err != nil {
		return OrganizationDoc{}, err
	}
	return org, nil
}

// --- Users ---

type UserDoc struct {
	OIDCSubject string    `firestore:"oidc_subject"`
	Email       string    `firestore:"email"`
	Name        string    `firestore:"name"`
	CreatedAt   time.Time `firestore:"created_at"`
	UpdatedAt   time.Time `firestore:"updated_at"`
}

func (s *Store) UpsertUser(ctx context.Context, subject, email, name string) (uuid.UUID, UserDoc, error) {
	col := s.client.Collection("users")
	iter := col.Where("oidc_subject", "==", subject).Limit(1).Documents(ctx)
	doc, err := iter.Next()
	if err == nil {
		var user UserDoc
		if err := doc.DataTo(&user); err != nil {
			return uuid.Nil, UserDoc{}, err
		}
		user.Email = email
		user.Name = name
		user.UpdatedAt = time.Now().UTC()
		if _, err := doc.Ref.Set(ctx, user); err != nil {
			return uuid.Nil, UserDoc{}, err
		}
		id, err := uuid.Parse(doc.Ref.ID)
		if err != nil {
			return uuid.Nil, UserDoc{}, err
		}
		return id, user, nil
	}
	if !errors.Is(err, iterator.Done) {
		return uuid.Nil, UserDoc{}, err
	}

	id := uuid.New()
	now := time.Now().UTC()
	user := UserDoc{OIDCSubject: subject, Email: email, Name: name, CreatedAt: now, UpdatedAt: now}
	if _, err := col.Doc(id.String()).Set(ctx, user); err != nil {
		return uuid.Nil, UserDoc{}, err
	}
	return id, user, nil
}

// --- Organization Members ---

type MemberDoc struct {
	OrganizationID string    `firestore:"organization_id"`
	UserID         string    `firestore:"user_id"`
	Role           string    `firestore:"role"`
	CreatedAt      time.Time `firestore:"created_at"`
	UpdatedAt      time.Time `firestore:"updated_at"`
}

func (s *Store) EnsureOrganizationMembership(ctx context.Context, orgID, userID uuid.UUID, role string) (MemberDoc, error) {
	col := s.client.Collection("organization_members")
	iter := col.Where("organization_id", "==", orgID.String()).Where("user_id", "==", userID.String()).Limit(1).Documents(ctx)
	doc, err := iter.Next()
	if err == nil {
		var member MemberDoc
		if err := doc.DataTo(&member); err != nil {
			return MemberDoc{}, err
		}
		return member, nil
	}
	if !errors.Is(err, iterator.Done) {
		return MemberDoc{}, err
	}

	now := time.Now().UTC()
	member := MemberDoc{
		OrganizationID: orgID.String(),
		UserID:         userID.String(),
		Role:           role,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	id := uuid.New()
	if _, err := col.Doc(id.String()).Set(ctx, member); err != nil {
		return MemberDoc{}, err
	}
	return member, nil
}

func (s *Store) ListOrganizationMembers(ctx context.Context, orgID uuid.UUID) ([]domain.Member, error) {
	memberIter := s.client.Collection("organization_members").Where("organization_id", "==", orgID.String()).Documents(ctx)
	defer memberIter.Stop()

	var members []domain.Member
	for {
		doc, err := memberIter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var m MemberDoc
		if err := doc.DataTo(&m); err != nil {
			return nil, err
		}
		userID, _ := uuid.Parse(m.UserID)
		userDoc, err := s.client.Collection("users").Doc(m.UserID).Get(ctx)
		if err != nil {
			continue
		}
		var user UserDoc
		if err := userDoc.DataTo(&user); err != nil {
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
	iter := s.client.Collection("organization_members").Where("organization_id", "==", orgID.String()).Where("user_id", "==", userID.String()).Limit(1).Documents(ctx)
	doc, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return MemberDoc{}, ErrNotFound
	}
	if err != nil {
		return MemberDoc{}, err
	}
	var m MemberDoc
	if err := doc.DataTo(&m); err != nil {
		return MemberDoc{}, err
	}
	return m, nil
}

// --- Meetings ---

type MeetingDoc struct {
	OrganizationID string     `firestore:"organization_id"`
	Year           int32      `firestore:"year"`
	Month          int32      `firestore:"month"`
	ScheduledAt    *time.Time `firestore:"scheduled_at,omitempty"`
	MeetingURL     *string    `firestore:"meeting_url,omitempty"`
	Notes          *string    `firestore:"notes,omitempty"`
	Status         string     `firestore:"status"`
	UpdatedBy      string     `firestore:"updated_by"`
	CreatedAt      time.Time  `firestore:"created_at"`
	UpdatedAt      time.Time  `firestore:"updated_at"`
}

func meetingDocID(orgID uuid.UUID, year, month int32) string {
	return fmt.Sprintf("%s_%d_%02d", orgID.String(), year, month)
}

func (s *Store) GetMeetingByPeriod(ctx context.Context, orgID uuid.UUID, year, month int32) (uuid.UUID, MeetingDoc, error) {
	docID := meetingDocID(orgID, year, month)
	doc, err := s.client.Collection("meetings").Doc(docID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return uuid.Nil, MeetingDoc{}, ErrNotFound
		}
		return uuid.Nil, MeetingDoc{}, err
	}
	var m MeetingDoc
	if err := doc.DataTo(&m); err != nil {
		return uuid.Nil, MeetingDoc{}, err
	}
	id, _ := uuid.Parse(doc.Ref.ID)
	return id, m, nil
}

func (s *Store) UpsertMeeting(ctx context.Context, orgID uuid.UUID, year, month int32, scheduledAt *time.Time, meetingURL, notes *string, meetingStatus, updatedBy string) (uuid.UUID, MeetingDoc, error) {
	docID := meetingDocID(orgID, year, month)
	now := time.Now().UTC()
	ref := s.client.Collection("meetings").Doc(docID)

	doc, err := ref.Get(ctx)
	var m MeetingDoc
	if err == nil {
		if err := doc.DataTo(&m); err != nil {
			return uuid.Nil, MeetingDoc{}, err
		}
	}

	m.OrganizationID = orgID.String()
	m.Year = year
	m.Month = month
	m.ScheduledAt = scheduledAt
	m.MeetingURL = meetingURL
	m.Notes = notes
	m.Status = meetingStatus
	m.UpdatedBy = updatedBy
	m.UpdatedAt = now
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}

	if _, err := ref.Set(ctx, m); err != nil {
		return uuid.Nil, MeetingDoc{}, err
	}
	id, _ := uuid.Parse(docID)
	return id, m, nil
}

func (s *Store) ListMeetingPeriods(ctx context.Context, orgID uuid.UUID) ([]MeetingDoc, error) {
	iter := s.client.Collection("meetings").Where("organization_id", "==", orgID.String()).Documents(ctx)
	defer iter.Stop()

	var results []MeetingDoc
	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var m MeetingDoc
		if err := doc.DataTo(&m); err != nil {
			continue
		}
		results = append(results, m)
	}
	return results, nil
}

// --- Tasks ---

type TaskDoc struct {
	OrganizationID string     `firestore:"organization_id"`
	Year           int32      `firestore:"year"`
	Month          int32      `firestore:"month"`
	Title          string     `firestore:"title"`
	Status         string     `firestore:"status"`
	DueDate        *time.Time `firestore:"due_date,omitempty"`
	ReferenceURL   *string    `firestore:"reference_url,omitempty"`
	AssigneeID     *string    `firestore:"assignee_id,omitempty"`
	CreatedBy      string     `firestore:"created_by"`
	Version        int32      `firestore:"version"`
	CreatedAt      time.Time  `firestore:"created_at"`
	UpdatedAt      time.Time  `firestore:"updated_at"`
	DeletedAt      *time.Time `firestore:"deleted_at,omitempty"`
}

func (s *Store) CreateTask(ctx context.Context, task TaskDoc) (uuid.UUID, TaskDoc, error) {
	id := uuid.New()
	now := time.Now().UTC()
	task.Version = 1
	task.CreatedAt = now
	task.UpdatedAt = now
	if _, err := s.client.Collection("tasks").Doc(id.String()).Set(ctx, task); err != nil {
		return uuid.Nil, TaskDoc{}, err
	}
	return id, task, nil
}

func (s *Store) GetTaskByID(ctx context.Context, orgID, taskID uuid.UUID) (TaskDoc, error) {
	doc, err := s.client.Collection("tasks").Doc(taskID.String()).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return TaskDoc{}, ErrNotFound
		}
		return TaskDoc{}, err
	}
	var t TaskDoc
	if err := doc.DataTo(&t); err != nil {
		return TaskDoc{}, err
	}
	if t.OrganizationID != orgID.String() || t.DeletedAt != nil {
		return TaskDoc{}, ErrNotFound
	}
	return t, nil
}

func (s *Store) ListTasksByPeriod(ctx context.Context, orgID uuid.UUID, year, month int32) ([]uuid.UUID, []TaskDoc, error) {
	iter := s.client.Collection("tasks").
		Where("organization_id", "==", orgID.String()).
		Where("year", "==", year).
		Where("month", "==", month).
		Documents(ctx)
	defer iter.Stop()

	var ids []uuid.UUID
	var tasks []TaskDoc
	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, nil, err
		}
		var t TaskDoc
		if err := doc.DataTo(&t); err != nil {
			continue
		}
		if t.DeletedAt != nil {
			continue
		}
		id, _ := uuid.Parse(doc.Ref.ID)
		ids = append(ids, id)
		tasks = append(tasks, t)
	}
	return ids, tasks, nil
}

func (s *Store) UpdateTask(ctx context.Context, orgID, taskID uuid.UUID, title, taskStatus string, dueDate *time.Time, referenceURL *string, assigneeID *uuid.UUID, expectedVersion int32) (TaskDoc, error) {
	ref := s.client.Collection("tasks").Doc(taskID.String())
	doc, err := ref.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return TaskDoc{}, ErrNotFound
		}
		return TaskDoc{}, err
	}
	var t TaskDoc
	if err := doc.DataTo(&t); err != nil {
		return TaskDoc{}, err
	}
	if t.OrganizationID != orgID.String() || t.DeletedAt != nil {
		return TaskDoc{}, ErrNotFound
	}
	if t.Version != expectedVersion {
		return TaskDoc{}, domain.NewAppError(domain.ErrorCodeConflict, "task version conflict", nil)
	}

	t.Title = title
	t.Status = taskStatus
	t.DueDate = dueDate
	t.ReferenceURL = referenceURL
	if assigneeID != nil {
		s := assigneeID.String()
		t.AssigneeID = &s
	} else {
		t.AssigneeID = nil
	}
	t.Version++
	t.UpdatedAt = time.Now().UTC()

	if _, err := ref.Set(ctx, t); err != nil {
		return TaskDoc{}, err
	}
	return t, nil
}

func (s *Store) DeleteTask(ctx context.Context, orgID, taskID uuid.UUID, expectedVersion int32) error {
	ref := s.client.Collection("tasks").Doc(taskID.String())
	doc, err := ref.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return ErrNotFound
		}
		return err
	}
	var t TaskDoc
	if err := doc.DataTo(&t); err != nil {
		return err
	}
	if t.OrganizationID != orgID.String() || t.DeletedAt != nil {
		return ErrNotFound
	}
	if t.Version != expectedVersion {
		return domain.NewAppError(domain.ErrorCodeConflict, "task version conflict", nil)
	}

	now := time.Now().UTC()
	t.DeletedAt = &now
	t.UpdatedAt = now
	_, err = ref.Set(ctx, t)
	return err
}

func (s *Store) ListTaskPeriodSummaries(ctx context.Context, orgID uuid.UUID) ([]TaskPeriodSummary, error) {
	iter := s.client.Collection("tasks").Where("organization_id", "==", orgID.String()).Documents(ctx)
	defer iter.Stop()

	type periodKey struct {
		year  int32
		month int32
	}

	summaryMap := map[periodKey]*TaskPeriodSummary{}
	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var t TaskDoc
		if err := doc.DataTo(&t); err != nil {
			continue
		}
		if t.DeletedAt != nil {
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

		taskID, _ := uuid.Parse(doc.Ref.ID)
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
	Year               int32
	Month              int32
	TaskCount          int64
	DoneTaskCount      int64
	RequiredApprovals  int64
	ApprovedApprovals  int64
}

// --- Task Approvals ---

type TaskApprovalDoc struct {
	TaskID         string     `firestore:"task_id"`
	ApproverUserID string     `firestore:"approver_user_id"`
	ApprovedAt     *time.Time `firestore:"approved_at,omitempty"`
	CreatedAt      time.Time  `firestore:"created_at"`
}

func (s *Store) ListTaskApprovals(ctx context.Context, taskID uuid.UUID) ([]TaskApprovalDoc, error) {
	iter := s.client.Collection("task_approvals").Where("task_id", "==", taskID.String()).Documents(ctx)
	defer iter.Stop()

	var results []TaskApprovalDoc
	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var a TaskApprovalDoc
		if err := doc.DataTo(&a); err != nil {
			continue
		}
		results = append(results, a)
	}
	return results, nil
}

func (s *Store) ApproveTask(ctx context.Context, taskID, approverUserID uuid.UUID) error {
	iter := s.client.Collection("task_approvals").
		Where("task_id", "==", taskID.String()).
		Where("approver_user_id", "==", approverUserID.String()).
		Limit(1).Documents(ctx)

	doc, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	var a TaskApprovalDoc
	if err := doc.DataTo(&a); err != nil {
		return err
	}
	if a.ApprovedAt != nil {
		return domain.NewAppError(domain.ErrorCodeConflict, "task already approved", nil)
	}

	now := time.Now().UTC()
	a.ApprovedAt = &now
	_, err = doc.Ref.Set(ctx, a)
	return err
}

func (s *Store) ReplaceTaskApprovals(ctx context.Context, taskID uuid.UUID, approvedMap map[uuid.UUID]*time.Time, approverIDs []uuid.UUID) error {
	// Delete existing approvals
	iter := s.client.Collection("task_approvals").Where("task_id", "==", taskID.String()).Documents(ctx)
	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		if _, err := doc.Ref.Delete(ctx); err != nil {
			return err
		}
	}

	// Create new approvals
	now := time.Now().UTC()
	for _, approverID := range approverIDs {
		approval := TaskApprovalDoc{
			TaskID:         taskID.String(),
			ApproverUserID: approverID.String(),
			CreatedAt:      now,
		}
		if approvedMap != nil {
			if existing, ok := approvedMap[approverID]; ok {
				approval.ApprovedAt = existing
			}
		}
		id := uuid.New()
		if _, err := s.client.Collection("task_approvals").Doc(id.String()).Set(ctx, approval); err != nil {
			return err
		}
	}
	return nil
}

// --- Announcements ---

type AnnouncementDoc struct {
	OrganizationID   string     `firestore:"organization_id"`
	Year             int32      `firestore:"year"`
	Month            int32      `firestore:"month"`
	Body             string     `firestore:"body"`
	Status           string     `firestore:"status"`
	PublishChannel   *string    `firestore:"publish_channel,omitempty"`
	DiscordMessageID *string    `firestore:"discord_message_id,omitempty"`
	LastError        *string    `firestore:"last_error,omitempty"`
	UpdatedBy        string     `firestore:"updated_by"`
	CreatedAt        time.Time  `firestore:"created_at"`
	UpdatedAt        time.Time  `firestore:"updated_at"`
}

func announcementDocID(orgID uuid.UUID, year, month int32) string {
	return fmt.Sprintf("%s_%d_%02d", orgID.String(), year, month)
}

func (s *Store) GetAnnouncementByPeriod(ctx context.Context, orgID uuid.UUID, year, month int32) (uuid.UUID, AnnouncementDoc, error) {
	docID := announcementDocID(orgID, year, month)
	doc, err := s.client.Collection("announcements").Doc(docID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return uuid.Nil, AnnouncementDoc{}, ErrNotFound
		}
		return uuid.Nil, AnnouncementDoc{}, err
	}
	var a AnnouncementDoc
	if err := doc.DataTo(&a); err != nil {
		return uuid.Nil, AnnouncementDoc{}, err
	}
	id, _ := uuid.Parse(docID)
	return id, a, nil
}

func (s *Store) GetAnnouncementByID(ctx context.Context, id uuid.UUID) (AnnouncementDoc, error) {
	doc, err := s.client.Collection("announcements").Doc(id.String()).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return AnnouncementDoc{}, ErrNotFound
		}
		return AnnouncementDoc{}, err
	}
	var a AnnouncementDoc
	if err := doc.DataTo(&a); err != nil {
		return AnnouncementDoc{}, err
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
	now := time.Now().UTC()
	ref := s.client.Collection("announcements").Doc(docID)

	doc, err := ref.Get(ctx)
	var a AnnouncementDoc
	if err == nil {
		if err := doc.DataTo(&a); err != nil {
			return uuid.Nil, AnnouncementDoc{}, err
		}
	}

	a.OrganizationID = orgID.String()
	a.Year = year
	a.Month = month
	a.Body = body
	a.Status = domain.AnnouncementStatusDraft
	a.PublishChannel = publishChannel
	a.UpdatedBy = updatedBy
	a.UpdatedAt = now
	if a.CreatedAt.IsZero() {
		a.CreatedAt = now
	}

	if _, err := ref.Set(ctx, a); err != nil {
		return uuid.Nil, AnnouncementDoc{}, err
	}
	id, _ := uuid.Parse(docID)
	return id, a, nil
}

func (s *Store) RequestAnnouncementPublish(ctx context.Context, orgID, announcementID uuid.UUID, publishChannel *string, updatedBy string) (uuid.UUID, AnnouncementDoc, error) {
	ref := s.client.Collection("announcements").Doc(announcementID.String())
	doc, err := ref.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return uuid.Nil, AnnouncementDoc{}, ErrNotFound
		}
		return uuid.Nil, AnnouncementDoc{}, err
	}
	var a AnnouncementDoc
	if err := doc.DataTo(&a); err != nil {
		return uuid.Nil, AnnouncementDoc{}, err
	}
	if a.OrganizationID != orgID.String() {
		return uuid.Nil, AnnouncementDoc{}, ErrNotFound
	}

	a.Status = domain.AnnouncementStatusPublishRequested
	if publishChannel != nil {
		a.PublishChannel = publishChannel
	}
	a.UpdatedBy = updatedBy
	a.UpdatedAt = time.Now().UTC()

	if _, err := ref.Set(ctx, a); err != nil {
		return uuid.Nil, AnnouncementDoc{}, err
	}
	return announcementID, a, nil
}

func (s *Store) CompleteAnnouncementPublish(ctx context.Context, id uuid.UUID, discordMessageID string) (uuid.UUID, AnnouncementDoc, error) {
	ref := s.client.Collection("announcements").Doc(id.String())
	doc, err := ref.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return uuid.Nil, AnnouncementDoc{}, ErrNotFound
		}
		return uuid.Nil, AnnouncementDoc{}, err
	}
	var a AnnouncementDoc
	if err := doc.DataTo(&a); err != nil {
		return uuid.Nil, AnnouncementDoc{}, err
	}

	a.Status = domain.AnnouncementStatusPublished
	a.DiscordMessageID = &discordMessageID
	a.LastError = nil
	a.UpdatedAt = time.Now().UTC()

	if _, err := ref.Set(ctx, a); err != nil {
		return uuid.Nil, AnnouncementDoc{}, err
	}
	return id, a, nil
}

func (s *Store) FailAnnouncementPublish(ctx context.Context, id uuid.UUID, lastError string) (uuid.UUID, AnnouncementDoc, error) {
	ref := s.client.Collection("announcements").Doc(id.String())
	doc, err := ref.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return uuid.Nil, AnnouncementDoc{}, ErrNotFound
		}
		return uuid.Nil, AnnouncementDoc{}, err
	}
	var a AnnouncementDoc
	if err := doc.DataTo(&a); err != nil {
		return uuid.Nil, AnnouncementDoc{}, err
	}

	a.Status = domain.AnnouncementStatusPublishFailed
	a.LastError = &lastError
	a.UpdatedAt = time.Now().UTC()

	if _, err := ref.Set(ctx, a); err != nil {
		return uuid.Nil, AnnouncementDoc{}, err
	}
	return id, a, nil
}

func (s *Store) ListAnnouncementPeriods(ctx context.Context, orgID uuid.UUID) ([]AnnouncementDoc, error) {
	iter := s.client.Collection("announcements").Where("organization_id", "==", orgID.String()).Documents(ctx)
	defer iter.Stop()

	var results []AnnouncementDoc
	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var a AnnouncementDoc
		if err := doc.DataTo(&a); err != nil {
			continue
		}
		results = append(results, a)
	}
	return results, nil
}

func (s *Store) ListPendingPublishRequests(ctx context.Context) ([]uuid.UUID, []AnnouncementDoc, error) {
	iter := s.client.Collection("announcements").Where("status", "==", domain.AnnouncementStatusPublishRequested).Documents(ctx)
	defer iter.Stop()

	var ids []uuid.UUID
	var results []AnnouncementDoc
	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, nil, err
		}
		var a AnnouncementDoc
		if err := doc.DataTo(&a); err != nil {
			continue
		}
		id, _ := uuid.Parse(doc.Ref.ID)
		ids = append(ids, id)
		results = append(results, a)
	}
	return ids, results, nil
}

// --- Audit Logs ---

type AuditLogDoc struct {
	OrganizationID string  `firestore:"organization_id"`
	ActorUserID    *string `firestore:"actor_user_id,omitempty"`
	ActorType      string  `firestore:"actor_type"`
	ActorLabel     string  `firestore:"actor_label"`
	Action         string  `firestore:"action"`
	ResourceType   string  `firestore:"resource_type"`
	ResourceID     string  `firestore:"resource_id"`
	BeforeState    any     `firestore:"before_state,omitempty"`
	AfterState     any     `firestore:"after_state,omitempty"`
	Result         string  `firestore:"result"`
	IP             string  `firestore:"ip"`
	CreatedAt      time.Time `firestore:"created_at"`
}

func (s *Store) CreateAuditLog(ctx context.Context, log AuditLogDoc) error {
	log.CreatedAt = time.Now().UTC()
	id := uuid.New()
	_, err := s.client.Collection("audit_logs").Doc(id.String()).Set(ctx, log)
	return err
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
