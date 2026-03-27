-- name: UpsertUser :one
INSERT INTO users (oidc_subject, email, name)
VALUES ($1, $2, $3)
ON CONFLICT (oidc_subject) DO UPDATE
SET email = EXCLUDED.email,
    name = EXCLUDED.name,
    updated_at = NOW()
RETURNING *;

-- name: EnsureOrganization :one
INSERT INTO organizations (slug, name)
VALUES ($1, $2)
ON CONFLICT (slug) DO UPDATE
SET name = EXCLUDED.name,
    updated_at = NOW()
RETURNING *;

-- name: EnsureOrganizationMembership :one
INSERT INTO organization_members (organization_id, user_id, role)
VALUES ($1, $2, $3)
ON CONFLICT (organization_id, user_id) DO UPDATE
SET role = EXCLUDED.role
RETURNING *;

-- name: GetOrganizationByID :one
SELECT * FROM organizations
WHERE id = $1;

-- name: ListOrganizationMembers :many
SELECT u.id, u.email, u.name, om.role
FROM organization_members om
JOIN users u ON u.id = om.user_id
WHERE om.organization_id = $1
ORDER BY u.name ASC;

-- name: GetOrganizationMember :one
SELECT om.*, u.oidc_subject, u.email, u.name
FROM organization_members om
JOIN users u ON u.id = om.user_id
WHERE om.organization_id = $1
  AND om.user_id = $2;

-- name: GetMeetingByPeriod :one
SELECT * FROM meetings
WHERE organization_id = $1
  AND year = $2
  AND month = $3;

-- name: UpsertMeeting :one
INSERT INTO meetings (
  organization_id,
  year,
  month,
  scheduled_at,
  meeting_url,
  notes,
  status,
  updated_by
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (organization_id, year, month) DO UPDATE
SET scheduled_at = EXCLUDED.scheduled_at,
    meeting_url = EXCLUDED.meeting_url,
    notes = EXCLUDED.notes,
    status = EXCLUDED.status,
    updated_by = EXCLUDED.updated_by,
    updated_at = NOW()
RETURNING *;

-- name: ListMeetingPeriods :many
SELECT year, month, status
FROM meetings
WHERE organization_id = $1
ORDER BY year DESC, month DESC;

-- name: ListTasksByPeriod :many
SELECT * FROM tasks
WHERE organization_id = $1
  AND year = $2
  AND month = $3
  AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: GetTaskByID :one
SELECT * FROM tasks
WHERE organization_id = $1
  AND id = $2
  AND deleted_at IS NULL;

-- name: CreateTask :one
INSERT INTO tasks (
  organization_id,
  year,
  month,
  title,
  status,
  due_date,
  reference_url,
  assignee_id,
  created_by
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: UpdateTask :one
UPDATE tasks
SET title = $3,
    status = $4,
    due_date = $5,
    reference_url = $6,
    assignee_id = $7,
    version = version + 1,
    updated_at = NOW()
WHERE organization_id = $1
  AND id = $2
  AND version = $8
  AND deleted_at IS NULL
RETURNING *;

-- name: DeleteTask :one
UPDATE tasks
SET deleted_at = NOW(),
    version = version + 1,
    updated_at = NOW()
WHERE organization_id = $1
  AND id = $2
  AND version = $3
  AND deleted_at IS NULL
RETURNING *;

-- name: DeleteTaskApprovalsByTaskID :exec
DELETE FROM task_approvals
WHERE task_id = $1;

-- name: CreateTaskApproval :exec
INSERT INTO task_approvals (task_id, approver_user_id)
VALUES ($1, $2);

-- name: CreateTaskApprovalWithState :exec
INSERT INTO task_approvals (task_id, approver_user_id, approved_at)
VALUES ($1, $2, $3);

-- name: ListTaskApprovalsByTaskID :many
SELECT * FROM task_approvals
WHERE task_id = $1
ORDER BY created_at ASC;

-- name: ListTaskApprovalsByTaskIDs :many
SELECT * FROM task_approvals
WHERE task_id = ANY($1::uuid[])
ORDER BY created_at ASC;

-- name: ApproveTask :one
UPDATE task_approvals
SET approved_at = NOW()
WHERE task_id = $1
  AND approver_user_id = $2
  AND approved_at IS NULL
RETURNING *;

-- name: ListTaskPeriodSummaries :many
SELECT
  t.year,
  t.month,
  COUNT(*)::bigint AS task_count,
  COUNT(*) FILTER (WHERE t.status = 'done')::bigint AS done_task_count,
  COALESCE(SUM(approval_stats.required_count), 0)::bigint AS required_approvals,
  COALESCE(SUM(approval_stats.approved_count), 0)::bigint AS approved_approvals
FROM tasks t
LEFT JOIN LATERAL (
  SELECT
    COUNT(*)::bigint AS required_count,
    COUNT(*) FILTER (WHERE approved_at IS NOT NULL)::bigint AS approved_count
  FROM task_approvals ta
  WHERE ta.task_id = t.id
) approval_stats ON TRUE
WHERE t.organization_id = $1
  AND t.deleted_at IS NULL
GROUP BY t.year, t.month
ORDER BY t.year DESC, t.month DESC;

-- name: GetAnnouncementByPeriod :one
SELECT * FROM announcements
WHERE organization_id = $1
  AND year = $2
  AND month = $3;

-- name: GetAnnouncementByID :one
SELECT * FROM announcements
WHERE id = $1;

-- name: GetAnnouncementByIDForOrg :one
SELECT * FROM announcements
WHERE organization_id = $1
  AND id = $2;

-- name: UpsertAnnouncementDraft :one
INSERT INTO announcements (
  organization_id,
  year,
  month,
  body,
  status,
  publish_channel,
  discord_message_id,
  last_error,
  updated_by
)
VALUES ($1, $2, $3, $4, 'draft', $5, NULL, NULL, $6)
ON CONFLICT (organization_id, year, month) DO UPDATE
SET body = EXCLUDED.body,
    publish_channel = COALESCE(EXCLUDED.publish_channel, announcements.publish_channel),
    status = 'draft',
    discord_message_id = NULL,
    last_error = NULL,
    updated_by = EXCLUDED.updated_by,
    updated_at = NOW()
RETURNING *;

-- name: RequestAnnouncementPublish :one
UPDATE announcements
SET status = 'publish_requested',
    publish_channel = $3,
    last_error = NULL,
    updated_by = $4,
    updated_at = NOW()
WHERE organization_id = $1
  AND id = $2
RETURNING *;

-- name: CompleteAnnouncementPublish :one
UPDATE announcements
SET status = 'published',
    discord_message_id = $2,
    last_error = NULL,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: FailAnnouncementPublish :one
UPDATE announcements
SET status = 'publish_failed',
    last_error = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: ListPendingAnnouncementPublishRequests :many
SELECT *
FROM announcements
WHERE status = 'publish_requested'
ORDER BY updated_at ASC;

-- name: ListAnnouncementPeriods :many
SELECT year, month, status
FROM announcements
WHERE organization_id = $1
ORDER BY year DESC, month DESC;

-- name: CreateAuditLog :exec
INSERT INTO audit_logs (
  organization_id,
  actor_user_id,
  actor_type,
  actor_label,
  action,
  resource_type,
  resource_id,
  before_state,
  after_state,
  result,
  ip
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);
