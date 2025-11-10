package news

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"ss-api/internal/app"
)

const (
	newsListEndpoint   = "https://stellasora.global/api/resource/news"
	newsDetailEndpoint = "https://stellasora.global/api/resource/news/detail"
	thumbnailCacheTTL  = 10 * time.Minute
)

var (
	categoryTypeMap = map[string]string{
		"updates": "latest",
		"notices": "notice",
		"news":    "news",
		"events":  "activity",
	}
	imgSrcPattern = regexp.MustCompile(`(?i)<img[^>]+src=["']([^"']+)["']`)
)

type Handler struct {
	client  *http.Client
	cache   map[int]cacheEntry
	cacheMu sync.RWMutex
}

type cacheEntry struct {
	detail        newsDetail
	heroThumbnail string
	expires       time.Time
}

// New constructs a handler that proxies Stella Sora news lists, enriching thumbnails via the detail API.
func New(_ *app.App) http.HandlerFunc {
	h := &Handler{
		client: &http.Client{Timeout: 10 * time.Second},
		cache:  make(map[int]cacheEntry),
	}
	return h.handle
}

func (h *Handler) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	category := strings.ToLower(strings.TrimSpace(r.PathValue("category")))
	if category == "" {
		writeJSONError(w, http.StatusNotFound, "news category required")
		return
	}

	newsType, ok := categoryTypeMap[category]
	if !ok {
		writeJSONError(w, http.StatusNotFound, fmt.Sprintf("unknown news category %q", category))
		return
	}

	index, err := parsePositiveQueryInt("index", r.URL.Query().Get("index"), 1)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	size, err := parsePositiveQueryInt("size", r.URL.Query().Get("size"), 6)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	endpoint, err := buildNewsListURL(newsType, index, size)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to build upstream request")
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to create upstream request")
		return
	}
	req.Header.Set("Accept", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "upstream news service unreachable")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		writeJSONError(w, http.StatusBadGateway, "upstream news service returned an error")
		return
	}

	decoder := json.NewDecoder(resp.Body)
	decoder.UseNumber()

	var payload newsListResponse
	if err := decoder.Decode(&payload); err != nil {
		writeJSONError(w, http.StatusBadGateway, "failed to decode upstream response")
		return
	}

	filteredRows := filterRowsByType(payload.Data.Rows, newsType)
	payload.Data.Count = len(filteredRows)
	payload.Data.Rows = filteredRows

	if err := h.enrichThumbnails(r.Context(), payload.Data.Rows); err != nil {
		log.Printf("news: thumbnail enrichment failed: %v", err)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("failed to write news response: %v", err)
	}
}

func (h *Handler) enrichThumbnails(ctx context.Context, rows []map[string]interface{}) error {
	if len(rows) == 0 {
		return nil
	}

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(4)

	for i := range rows {
		row := rows[i]
		g.Go(func() error {
			idValue, ok := row["id"].(json.Number)
			if !ok {
				return nil
			}

			id, err := idValue.Int64()
			if err != nil {
				return nil
			}

			_, thumbnail, err := h.fetchNewsDetail(ctx, int(id))
			if err != nil || thumbnail == "" {
				return nil
			}

			row["thumbnail"] = thumbnail
			return nil
		})
	}

	return g.Wait()
}

func (h *Handler) fetchNewsDetail(ctx context.Context, id int) (newsDetail, string, error) {
	if detail, hero, ok := h.cachedNews(id); ok {
		return detail, hero, nil
	}

	childCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	endpoint, err := buildNewsDetailURL(id)
	if err != nil {
		return newsDetail{}, "", err
	}

	req, err := http.NewRequestWithContext(childCtx, http.MethodGet, endpoint, nil)
	if err != nil {
		return newsDetail{}, "", err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return newsDetail{}, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return newsDetail{}, "", fmt.Errorf("upstream status %d", resp.StatusCode)
	}

	var detailResp newsDetailResponse
	if err := json.NewDecoder(resp.Body).Decode(&detailResp); err != nil {
		return newsDetail{}, "", err
	}

	detail := detailResp.Data.News
	hero := detail.Thumbnail
	if img := extractFirstImage(detail.Content); img != "" {
		hero = img
	}

	h.storeNews(id, detail, hero)
	return detail, hero, nil
}

func extractFirstImage(content string) string {
	match := imgSrcPattern.FindStringSubmatch(content)
	if len(match) < 2 {
		return ""
	}

	return html.UnescapeString(match[1])
}

func parsePositiveQueryInt(name, value string, fallback int) (int, error) {
	if strings.TrimSpace(value) == "" {
		return fallback, nil
	}

	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}

	return n, nil
}

func buildNewsListURL(newsType string, index, size int) (string, error) {
	values := url.Values{}
	values.Set("type", newsType)
	values.Set("index", strconv.Itoa(index))
	values.Set("size", strconv.Itoa(size))

	endpoint, err := url.Parse(newsListEndpoint)
	if err != nil {
		return "", err
	}

	endpoint.RawQuery = values.Encode()
	return endpoint.String(), nil
}

func buildNewsDetailURL(id int) (string, error) {
	endpoint, err := url.Parse(newsDetailEndpoint)
	if err != nil {
		return "", err
	}

	values := url.Values{}
	values.Set("id", strconv.Itoa(id))
	endpoint.RawQuery = values.Encode()
	return endpoint.String(), nil
}

type newsListResponse struct {
	Code      int          `json:"code"`
	Message   string       `json:"message"`
	Data      newsListData `json:"data"`
	Timestamp json.Number  `json:"timestamp"`
}

type newsListData struct {
	Count int                      `json:"count"`
	Rows  []map[string]interface{} `json:"rows"`
}

type newsDetailResponse struct {
	Code    int        `json:"code"`
	Message string     `json:"message"`
	Data    detailData `json:"data"`
}

type detailData struct {
	News newsDetail `json:"news"`
}

type newsDetail struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Type        string `json:"type"`
	TypeLabel   string `json:"typeLabel"`
	PublishTime int64  `json:"publishTime"`
	Thumbnail   string `json:"thumbnail"`
	Content     string `json:"content"`
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func filterRowsByType(rows []map[string]interface{}, expectedType string) []map[string]interface{} {
	if expectedType == "" || strings.EqualFold(expectedType, "latest") {
		return rows
	}

	filtered := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		rowType, _ := row["type"].(string)
		if strings.EqualFold(rowType, expectedType) {
			filtered = append(filtered, row)
		}
	}

	return filtered
}

func (h *Handler) cachedNews(id int) (newsDetail, string, bool) {
	h.cacheMu.RLock()
	entry, ok := h.cache[id]
	h.cacheMu.RUnlock()
	if !ok {
		return newsDetail{}, "", false
	}

	if time.Now().After(entry.expires) {
		h.cacheMu.Lock()
		delete(h.cache, id)
		h.cacheMu.Unlock()
		return newsDetail{}, "", false
	}

	return entry.detail, entry.heroThumbnail, true
}

func (h *Handler) storeNews(id int, detail newsDetail, hero string) {
	if hero == "" {
		hero = detail.Thumbnail
	}

	h.cacheMu.Lock()
	h.cache[id] = cacheEntry{
		detail:        detail,
		heroThumbnail: hero,
		expires:       time.Now().Add(thumbnailCacheTTL),
	}
	h.cacheMu.Unlock()
}
