// Package aurion provides HTTP client functionality for communicating with the Aurion system.
// It handles authentication, session management, and calendar data retrieval.
package aurion

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/FRFlo/isen-ical-go/internal/models"
)

const (
	defaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:135.0) Gecko/20100101 Firefox/135.0"
	contentType      = "application/x-www-form-urlencoded"
)

// Client handles HTTP communication with Aurion.
// It maintains session state including cookies, ViewState, and various form IDs
// required for navigating Aurion's JSF-based interface.
type Client struct {
	baseURL        string
	httpClient     *http.Client
	viewState      string // JSF ViewState token for form submissions
	idInit         string // Initialization ID for session state
	menuID         string // Sidebar menu ID for "Mon Planning"
	formIDPlanning string // Form ID for the planning page
}

// NewClient creates a new Aurion client
func NewClient(baseURL string) *Client {
	jar, _ := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: nil,
	})

	return &Client{
		baseURL: strings.TrimSpace(baseURL),
		httpClient: &http.Client{
			Jar: jar,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// SetHTTPClient allows setting a custom HTTP client (useful for testing)
func (c *Client) SetHTTPClient(client *http.Client) {
	c.httpClient = client
}

// Login authenticates with Aurion using email and password
func (c *Client) Login(email, password string) error {
	data := url.Values{}
	data.Set("username", email)
	data.Set("password", password)
	data.Set("j_idt28", "")

	req, err := http.NewRequest("POST", c.baseURL+"/login", strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}

	req.Header.Set("Content-Type", contentType)
	req.Header.Set("User-Agent", defaultUserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("failed to close response body: %v\n", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusFound {
		return fmt.Errorf("login failed, HTTP code %d", resp.StatusCode)
	}

	return nil
}

// InitializeSession fetches the home page and extracts ViewState and idInit
func (c *Client) InitializeSession() error {
	body, err := c.get("/", "")
	if err != nil {
		return fmt.Errorf("failed to initialize session: %w", err)
	}

	c.viewState, err = parseViewState(body)
	if err != nil {
		return fmt.Errorf("failed to parse ViewState: %w", err)
	}

	c.idInit, err = parseIdInit(body)
	if err != nil {
		return fmt.Errorf("failed to parse idInit: %w", err)
	}

	return nil
}

// NavigateToPlanning navigates to the planning page and extracts necessary IDs
func (c *Client) NavigateToPlanning() error {
	sidebarBody, err := c.get("/faces/MainMenuPage.xhtml", "https://aurion.junia.com/")
	if err != nil {
		return fmt.Errorf("failed to get main menu: %w", err)
	}

	c.menuID, err = parseSidebarMenuId(sidebarBody)
	if err != nil {
		return fmt.Errorf("failed to parse sidebar menu ID: %w", err)
	}

	data := url.Values{}
	data.Set("form", "form")
	data.Set("form:largeurDivCenter", "885")
	data.Set("form:idInit", c.idInit)
	data.Set("form:sauvegarde", "")
	data.Set("form:j_idt773_focus", "")
	data.Set("form:j_idt773_input", "44323")
	data.Set("javax.faces.ViewState", c.viewState)
	data.Set("form:sidebar", "form:sidebar")
	data.Set("form:sidebar_menuid", c.menuID)

	_, err = c.post("/faces/MainMenuPage.xhtml", data.Encode(), "")
	if err != nil {
		return fmt.Errorf("failed to post main menu: %w", err)
	}

	planningBody, err := c.get("/faces/Planning.xhtml", "https://aurion.junia.com/faces/MainMenuPage.xhtml")
	if err != nil {
		return fmt.Errorf("failed to get planning page: %w", err)
	}

	c.viewState, err = parseViewState(planningBody)
	if err != nil {
		return fmt.Errorf("failed to parse ViewState from planning: %w", err)
	}

	c.formIDPlanning, err = parseFormIdPlanning(planningBody)
	if err != nil {
		return fmt.Errorf("failed to parse form ID planning: %w", err)
	}

	return nil
}

// FetchPlanningData retrieves calendar events for the given date range
func (c *Client) FetchPlanningData(start, end int64) ([]models.AurionEvent, error) {
	now := time.Unix(start/1000, 0)
	today := now.Format("02/01/2006")
	week := fmt.Sprintf("%02d", getWeekNumber(now))
	year := strconv.Itoa(now.Year())

	data := url.Values{}
	data.Set("javax.faces.partial.ajax", "true")
	data.Set("javax.faces.source", c.formIDPlanning)
	data.Set("javax.faces.partial.execute", c.formIDPlanning)
	data.Set("javax.faces.partial.render", c.formIDPlanning)
	data.Set(c.formIDPlanning, c.formIDPlanning)
	data.Set(c.formIDPlanning+"_start", strconv.FormatInt(start, 10))
	data.Set(c.formIDPlanning+"_end", strconv.FormatInt(end, 10))
	data.Set("form", "form")
	data.Set("form:largeurDivCenter", "")
	data.Set("form:idInit", c.idInit)
	data.Set("form:date_input", today)
	data.Set("form:week", week+"-"+year)
	data.Set(c.formIDPlanning+"_view", "agendaWeek")
	data.Set("form:offsetFuseauNavigateur", "-7200000")
	data.Set("form:onglets_activeIndex", "0")
	data.Set("form:onglets_scrollState", "0")
	data.Set("form:j_idt244_focus", "")
	data.Set("form:j_idt244_input", "44323")
	data.Set("javax.faces.ViewState", c.viewState)

	body, err := c.post("/faces/Planning.xhtml", data.Encode(), "")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch planning data: %w", err)
	}

	events, err := parsePlanningData(body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse planning data: %w", err)
	}

	return events, nil
}

// GetPlanning is a high-level method that performs the full flow to get events
func (c *Client) GetPlanning(email, password string, startTimestamp, endTimestamp *int64) ([]models.AurionEvent, error) {
	var start, end int64

	if startTimestamp != nil {
		start = *startTimestamp
	} else {
		start = time.Now().Add(-7*24*time.Hour).Unix() * 1000
	}

	if endTimestamp != nil {
		end = *endTimestamp
	} else {
		end = start + 60*24*60*60*1000
	}

	if err := c.Login(email, password); err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}

	if err := c.InitializeSession(); err != nil {
		return nil, fmt.Errorf("session initialization failed: %w", err)
	}

	if err := c.NavigateToPlanning(); err != nil {
		return nil, fmt.Errorf("navigation to planning failed: %w", err)
	}

	events, err := c.FetchPlanningData(start, end)
	if err != nil {
		return nil, fmt.Errorf("fetch planning data failed: %w", err)
	}

	return events, nil
}

// get performs a GET request
func (c *Client) get(path, referer string) (string, error) {
	u := c.baseURL + path

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", defaultUserAgent)
	if referer != "" {
		req.Header.Set("Referer", referer)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("failed to close response body: %v\n", err)
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// post performs a POST request
func (c *Client) post(path, body, referer string) (string, error) {
	u := c.baseURL + path

	req, err := http.NewRequest("POST", u, strings.NewReader(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", contentType)
	req.Header.Set("User-Agent", defaultUserAgent)
	if referer != "" {
		req.Header.Set("Referer", referer)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("failed to close response body: %v\n", err)
		}
	}(resp.Body)

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(respBody), nil
}

// parseViewState extracts the ViewState value from HTML
func parseViewState(html string) (string, error) {
	re := regexp.MustCompile(`<input type="hidden" name="javax\.faces\.ViewState" id="j_id1:javax\.faces\.ViewState:0" value="([^"]+)" autocomplete="off" />`)
	matches := re.FindStringSubmatch(html)
	if len(matches) < 2 {
		return "", fmt.Errorf("ViewState not found")
	}
	return matches[1], nil
}

// parseIdInit extracts the form:idInit value from HTML
func parseIdInit(html string) (string, error) {
	prefix := `name="form:idInit" value="`
	startIdx := strings.Index(html, prefix)
	if startIdx == -1 {
		return "", fmt.Errorf("idInit not found")
	}
	startIdx += len(prefix)
	endIdx := strings.Index(html[startIdx:], `"`)
	if endIdx == -1 {
		return "", fmt.Errorf("idInit value not terminated")
	}
	return html[startIdx : startIdx+endIdx], nil
}

// parseSidebarMenuId finds the menu ID for 'Mon Planning'
func parseSidebarMenuId(html string) (string, error) {
	re := regexp.MustCompile(`onclick="[^"]*?PrimeFaces\.addSubmitParam\('form',\{'form:sidebar':'form:sidebar','form:sidebar_menuid':'(\d+)'}[^"]*?"[^>]*?>[^<]*<span class="ui-menuitem-icon ui-icon fa fa-calendar-alt"></span><span class="ui-menuitem-text">Mon Planning</span>`)
	matches := re.FindStringSubmatch(html)
	if len(matches) < 2 {
		return "", fmt.Errorf("sidebar menu id for Mon Planning not found")
	}
	return matches[1], nil
}

// parseFormIdPlanning extracts the Schedule widget ID from PrimeFaces.cw call
func parseFormIdPlanning(html string) (string, error) {
	re := regexp.MustCompile(`PrimeFaces\.cw\("Schedule","schedule",\{id:"([^"]+)"`)
	matches := re.FindStringSubmatch(html)
	if len(matches) < 2 {
		return "", fmt.Errorf("formIdPlanning not found")
	}
	return matches[1], nil
}

// parsePlanningData parses JSON events from the response
func parsePlanningData(response string) ([]models.AurionEvent, error) {
	re := regexp.MustCompile(`\[\{"id"(.*?)\]\]`)
	matches := re.FindStringSubmatch(response)
	if len(matches) < 1 {
		return nil, fmt.Errorf("planning data not found in response")
	}

	data := matches[0]
	if len(data) < 3 {
		return nil, fmt.Errorf("planning data not found in response")
	}
	data = data[:len(data)-3]

	var events []models.AurionEvent
	if err := json.Unmarshal([]byte(data), &events); err != nil {
		return nil, fmt.Errorf("failed to parse events JSON: %w", err)
	}

	return events, nil
}

// getWeekNumber calculates the ISO week number for a date
func getWeekNumber(t time.Time) int {
	_, week := t.ISOWeek()
	return week
}
