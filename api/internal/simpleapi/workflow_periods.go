package simpleapi

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"firebase.google.com/go/v4/db"
	"github.com/labstack/echo/v4"
)

// WorkflowPeriodResponse represents a monthly workflow period's status summary.
type WorkflowPeriodResponse struct {
	Year                 int    `json:"year"`
	Month                int    `json:"month"`
	MeetingStatus        string `json:"meeting_status"`
	TaskCompletionStatus string `json:"task_completion_status"`
	AnnouncementStatus   string `json:"announcement_status"`
	NextStageReady       bool   `json:"next_stage_ready"`
}

// RegisterWorkflowPeriodRoutes registers the workflow-periods endpoint.
// RegisterWorkflowPeriodRoutes registers workflow period endpoints.
func RegisterWorkflowPeriodRoutes(g *echo.Group, client *db.Client) {
	g.GET("/workflow-periods", getWorkflowPeriodsHandler(client))
	g.POST("/workflow-periods/provision", provisionWorkflowPeriodHandler(client))
}

func getWorkflowPeriodsHandler(client *db.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		type periodKey struct {
			year  int
			month int
		}

		periods := map[periodKey]*WorkflowPeriodResponse{}
		getPeriod := func(year, month int) *WorkflowPeriodResponse {
			k := periodKey{year: year, month: month}
			if existing, ok := periods[k]; ok {
				return existing
			}
			periods[k] = &WorkflowPeriodResponse{
				Year:                 year,
				Month:                month,
				MeetingStatus:        "missing",
				TaskCompletionStatus: "empty",
				AnnouncementStatus:   "missing",
				NextStageReady:       false,
			}
			return periods[k]
		}

		// Scan meeting_settings for period-based docs.
		var meetingDocs map[string]MeetingResponse
		if err := client.NewRef(meetingCollection).Get(ctx, &meetingDocs); err == nil {
			for docID, doc := range meetingDocs {
				year, month := parsePeriodDocID(docID)
				if year == 0 {
					continue
				}
				p := getPeriod(year, month)
				if doc.MeetingAt != nil && *doc.MeetingAt != "" {
					p.MeetingStatus = "planned"
				}
			}
		}

		// Scan task collections for period-based collections.
		// Check reasonable range: -2 months to +3 months from now.
		now := time.Now()
		for offset := -2; offset <= 3; offset++ {
			t := now.AddDate(0, offset, 0)
			year := t.Year()
			month := int(t.Month())
			collection := tasksCollectionForPeriod(year, month)
			var tasks map[string]TaskResponse
			if err := client.NewRef(collection).Get(ctx, &tasks); err == nil && len(tasks) > 0 {
				p := getPeriod(year, month)
				total := 0
				done := 0
				for _, task := range tasks {
					total++
					if task.State == "done" || task.State == "approved" {
						done++
					}
				}
				switch {
				case total == 0:
					p.TaskCompletionStatus = "empty"
				case done < total:
					p.TaskCompletionStatus = "in_progress"
				default:
					p.TaskCompletionStatus = "ready"
				}
			}
		}

		// Scan publicity_template for period-based docs.
		var templateDocs map[string]TemplateResponse
		if err := client.NewRef(templateCollection).Get(ctx, &templateDocs); err == nil {
			for docID, doc := range templateDocs {
				year, month := parsePeriodDocID(docID)
				if year == 0 {
					continue
				}
				p := getPeriod(year, month)
				if doc.Text != "" {
					p.AnnouncementStatus = "draft"
				}
			}
		}

		// Scan publicity_channels for period-based collections.
		for offset := -2; offset <= 3; offset++ {
			t := now.AddDate(0, offset, 0)
			year := t.Year()
			month := int(t.Month())
			collection := channelsCollectionForPeriod(year, month)
			var channels map[string]ChannelResponse
			if err := client.NewRef(collection).Get(ctx, &channels); err == nil && len(channels) > 0 {
				p := getPeriod(year, month)
				allDone := true
				for _, ch := range channels {
					if ch.State != "done" {
						allDone = false
						break
					}
				}
				if allDone {
					p.AnnouncementStatus = "published"
				} else if p.AnnouncementStatus == "missing" {
					p.AnnouncementStatus = "draft"
				}
			}
		}

		// Ensure current month and next 2 months always appear.
		for offset := 0; offset <= 2; offset++ {
			t := now.AddDate(0, offset, 0)
			getPeriod(t.Year(), int(t.Month()))
		}

		// Build result, set next_stage_ready.
		results := make([]WorkflowPeriodResponse, 0, len(periods))
		for _, p := range periods {
			p.NextStageReady = p.MeetingStatus != "missing" || p.TaskCompletionStatus == "ready"
			results = append(results, *p)
		}

		// Sort descending (newest first).
		sort.Slice(results, func(i, j int) bool {
			if results[i].Year == results[j].Year {
				return results[i].Month > results[j].Month
			}
			return results[i].Year > results[j].Year
		})

		return c.JSON(http.StatusOK, map[string]interface{}{"periods": results})
	}
}

// provisionWorkflowPeriodHandler creates initial meeting and channel data for a given year/month.
func provisionWorkflowPeriodHandler(client *db.Client) echo.HandlerFunc {
	type request struct {
		Year  int `json:"year"`
		Month int `json:"month"`
	}

	return func(c echo.Context) error {
		var req request
		if err := c.Bind(&req); err != nil {
			return validationError(c, "body", "invalid JSON")
		}
		if req.Year < 2000 || req.Year > 2100 {
			return validationError(c, "year", "must be between 2000 and 2100")
		}
		if req.Month < 1 || req.Month > 12 {
			return validationError(c, "month", "must be between 1 and 12")
		}

		ctx := c.Request().Context()
		now := time.Now().Format(time.RFC3339)

		// Meeting が未作成なら初期化
		meetDocID := meetingDocID(req.Year, req.Month)
		meetRef := client.NewRef(meetingCollection).Child(meetDocID)
		var existingMeet MeetingResponse
		if err := meetRef.Get(ctx, &existingMeet); err != nil || existingMeet.UpdatedAt == "" {
			initial := MeetingResponse{MeetURL: "", UpdatedAt: now}
			_ = meetRef.Set(ctx, initial)
		}

		// Publicity channels が未作成なら初期化
		chCollection := channelsCollectionForPeriod(req.Year, req.Month)
		var existingCh map[string]ChannelResponse
		if err := client.NewRef(chCollection).Get(ctx, &existingCh); err != nil || len(existingCh) == 0 {
			for _, ch := range defaultChannels() {
				_ = client.NewRef(chCollection).Child(ch.ID).Set(ctx, ch)
			}
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"status":  "ok",
			"message": fmt.Sprintf("provisioned %d/%02d", req.Year, req.Month),
		})
	}
}

// parsePeriodDocID parses "YYYY_MM" format from a doc ID.
func parsePeriodDocID(docID string) (int, int) {
	if docID == "default" {
		return 0, 0
	}
	parts := strings.SplitN(docID, "_", 2)
	if len(parts) != 2 {
		return 0, 0
	}
	var year, month int
	if _, err := fmt.Sscanf(parts[0], "%d", &year); err != nil {
		return 0, 0
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &month); err != nil {
		return 0, 0
	}
	if year < 2000 || year > 2100 || month < 1 || month > 12 {
		return 0, 0
	}
	return year, month
}
