package aurion

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/FRFlo/isen-ical-go/internal/models"
)

func TestParseViewState(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
		wantErr  bool
	}{
		{
			name:     "valid ViewState",
			html:     `<input type="hidden" name="javax.faces.ViewState" id="j_id1:javax.faces.ViewState:0" value="abc123xyz" autocomplete="off" />`,
			expected: "abc123xyz",
			wantErr:  false,
		},
		{
			name:     "ViewState with special characters",
			html:     `<input type="hidden" name="javax.faces.ViewState" id="j_id1:javax.faces.ViewState:0" value="-1234567890!@#$%^&*()" autocomplete="off" />`,
			expected: "-1234567890!@#$%^&*()",
			wantErr:  false,
		},
		{
			name:     "missing ViewState",
			html:     `<input type="hidden" name="other" value="test" />`,
			expected: "",
			wantErr:  true,
		},
		{
			name:     "empty HTML",
			html:     "",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseViewState(tt.html)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseViewState() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("parseViewState() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseIdInit(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
		wantErr  bool
	}{
		{
			name:     "valid idInit",
			html:     `<input name="form:idInit" value="init123" />`,
			expected: "init123",
			wantErr:  false,
		},
		{
			name:     "idInit with complex value",
			html:     `<input name="form:idInit" value="abc-def_123.xyz" />`,
			expected: "abc-def_123.xyz",
			wantErr:  false,
		},
		{
			name:     "missing idInit",
			html:     `<input name="form:other" value="test" />`,
			expected: "",
			wantErr:  true,
		},
		{
			name:     "empty HTML",
			html:     "",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseIdInit(tt.html)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseIdInit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("parseIdInit() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseSidebarMenuId(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
		wantErr  bool
	}{
		{
			name: "valid Mon Planning menu",
			html: `onclick="PrimeFaces.addSubmitParam('form',{'form:sidebar':'form:sidebar','form:sidebar_menuid':'12345'}"` +
				`><span class="ui-menuitem-icon ui-icon fa fa-calendar-alt"></span><span class="ui-menuitem-text">Mon Planning</span>`,
			expected: "12345",
			wantErr:  false,
		},
		{
			name: "different menu ID",
			html: `onclick="PrimeFaces.addSubmitParam('form',{'form:sidebar':'form:sidebar','form:sidebar_menuid':'99999'}"` +
				`><span class="ui-menuitem-icon ui-icon fa fa-calendar-alt"></span><span class="ui-menuitem-text">Mon Planning</span>`,
			expected: "99999",
			wantErr:  false,
		},
		{
			name:     "missing Mon Planning",
			html:     `<span class="ui-menuitem-text">Other Menu</span>`,
			expected: "",
			wantErr:  true,
		},
		{
			name:     "empty HTML",
			html:     "",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSidebarMenuId(tt.html)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSidebarMenuId() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("parseSidebarMenuId() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseFormIdPlanning(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
		wantErr  bool
	}{
		{
			name:     "valid Schedule widget ID",
			html:     `PrimeFaces.cw("Schedule","schedule",{id:"form:j_idt123"`,
			expected: "form:j_idt123",
			wantErr:  false,
		},
		{
			name:     "different widget ID",
			html:     `PrimeFaces.cw("Schedule","schedule",{id:"form:scheduleWidget"`,
			expected: "form:scheduleWidget",
			wantErr:  false,
		},
		{
			name:     "missing Schedule widget",
			html:     `PrimeFaces.cw("Other","other",{id:"form:other"`,
			expected: "",
			wantErr:  true,
		},
		{
			name:     "empty HTML",
			html:     "",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseFormIdPlanning(tt.html)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseFormIdPlanning() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("parseFormIdPlanning() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParsePlanningData(t *testing.T) {
	tests := []struct {
		name     string
		response string
		expected []models.AurionEvent
		wantErr  bool
	}{
		{
			name:     "valid events JSON",
			response: `[{"id":"1","title":"Math Class","start":"2024-01-15T08:00:00","end":"2024-01-15T10:00:00","allDay":false,"editable":false,"className":"course"}]x]]`,
			expected: []models.AurionEvent{
				{
					ID:        "1",
					Title:     "Math Class",
					Start:     "2024-01-15T08:00:00",
					End:       "2024-01-15T10:00:00",
					AllDay:    false,
					Editable:  false,
					ClassName: "course",
				},
			},
			wantErr: false,
		},
		{
			name:     "multiple events",
			response: `[{"id":"1","title":"Math","start":"2024-01-15T08:00:00","end":"2024-01-15T10:00:00","allDay":false,"editable":false,"className":"course"},{"id":"2","title":"Physics","start":"2024-01-15T10:00:00","end":"2024-01-15T12:00:00","allDay":false,"editable":false,"className":"course"}]x]]`,
			expected: []models.AurionEvent{
				{
					ID:        "1",
					Title:     "Math",
					Start:     "2024-01-15T08:00:00",
					End:       "2024-01-15T10:00:00",
					AllDay:    false,
					Editable:  false,
					ClassName: "course",
				},
				{
					ID:        "2",
					Title:     "Physics",
					Start:     "2024-01-15T10:00:00",
					End:       "2024-01-15T12:00:00",
					AllDay:    false,
					Editable:  false,
					ClassName: "course",
				},
			},
			wantErr: false,
		},
		{
			name:     "allDay event",
			response: `[{"id":"3","title":"Holiday","start":"2024-01-15","end":"2024-01-16","allDay":true,"editable":false,"className":"holiday"}]x]]`,
			expected: []models.AurionEvent{
				{
					ID:        "3",
					Title:     "Holiday",
					Start:     "2024-01-15",
					End:       "2024-01-16",
					AllDay:    true,
					Editable:  false,
					ClassName: "holiday",
				},
			},
			wantErr: false,
		},
		{
			name:     "missing events data",
			response: `some random text without events`,
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "empty response",
			response: "",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "invalid JSON",
			response: `[{"id":"1","title":invalid}]]`,
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePlanningData(tt.response)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePlanningData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.expected == nil {
				if got != nil {
					t.Errorf("parsePlanningData() = %v, want nil", got)
				}
				return
			}

			if len(got) != len(tt.expected) {
				t.Errorf("parsePlanningData() returned %d events, want %d", len(got), len(tt.expected))
				return
			}

			for i := range got {
				if got[i].ID != tt.expected[i].ID ||
					got[i].Title != tt.expected[i].Title ||
					got[i].Start != tt.expected[i].Start ||
					got[i].End != tt.expected[i].End ||
					got[i].AllDay != tt.expected[i].AllDay ||
					got[i].Editable != tt.expected[i].Editable ||
					got[i].ClassName != tt.expected[i].ClassName {
					t.Errorf("parsePlanningData() event %d = %+v, want %+v", i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestGetWeekNumber(t *testing.T) {
	tests := []struct {
		name     string
		dateStr  string
		expected int
	}{
		{
			name:     "first week of 2024",
			dateStr:  "2024-01-01",
			expected: 1,
		},
		{
			name:     "mid-year 2024",
			dateStr:  "2024-06-15",
			expected: 24,
		},
		{
			name:     "end of year 2024",
			dateStr:  "2024-12-31",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			date, _ := parseDate(tt.dateStr)
			got := getWeekNumber(date)
			if got != tt.expected {
				t.Errorf("getWeekNumber() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func parseDate(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}

func TestAurionEventJSON(t *testing.T) {
	event := models.AurionEvent{
		ID:        "1",
		Title:     "Test Event",
		Start:     "2024-01-15T08:00:00",
		End:       "2024-01-15T10:00:00",
		AllDay:    false,
		Editable:  true,
		ClassName: "test-class",
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	var parsed models.AurionEvent
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if parsed.ID != event.ID ||
		parsed.Title != event.Title ||
		parsed.Start != event.Start ||
		parsed.End != event.End ||
		parsed.AllDay != event.AllDay ||
		parsed.Editable != event.Editable ||
		parsed.ClassName != event.ClassName {
		t.Errorf("Parsed event doesn't match original: got %+v, want %+v", parsed, event)
	}
}
