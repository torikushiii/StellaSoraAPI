package news

import (
	"context"
	"encoding/json"
	"errors"
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

	"github.com/jasonlvhit/gocron"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/sync/errgroup"

	"ss-api/internal/app"
)

const (
	newsListPath      = "/api/resource/news"
	newsDetailPath    = "/api/resource/news/detail"
	thumbnailCacheTTL = 10 * time.Minute
	newsCollectionName = "news_articles"
	newsSyncPageSize   = 30
)

var (
	categoryTypeMap = map[string]string{
		"updates": "latest",
		"notices": "notice",
		"news":    "news",
		"events":  "activity",
	}
	regionBaseURLs = map[string]string{
		"global": "https://stellasora.global",
		"jp":     "https://stellasora.jp",
		"tw":     "https://stellasora.stargazer-games.com",
		"cn":     "https://stellasora.yostar.cn",
	}
	langToRegion = map[string]string{
		"en":    "global",
		"us":    "global",
		"jp":    "jp",
		"ja":    "jp",
		"tw":    "tw",
		"zh-tw": "tw",
		"cn":    "cn",
		"zh-cn": "cn",
		"zh":    "cn",
	}
	imgSrcPattern = regexp.MustCompile(`(?i)<img[^>]+src=["']([^"']+)["']`)
)

type Handler struct {
	app        *app.App
	dbName     string
	client     *http.Client
	cache      map[string]cacheEntry
	cacheMu    sync.RWMutex
	syncOnce   sync.Once
	scheduler  *gocron.Scheduler
	stopSignal chan bool
}

type cacheEntry struct {
	detail        newsDetail
	heroThumbnail string
	expires       time.Time
}

// New constructs the news handler and starts the periodic cache synchronizer.
func New(appInstance *app.App) http.HandlerFunc {
	h := &Handler{
		app:    appInstance,
		dbName: appInstance.DatabaseName(),
		client: &http.Client{Timeout: 10 * time.Second},
		cache:  make(map[string]cacheEntry),
	}
	h.startSyncLoop()
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

	lang := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("lang")))
	if lang == "" {
		lang = "en"
	}

	region, ok := langToRegion[lang]
	if !ok {
		// Fallback: check if the user provided the raw region key directly or an unknown lang
		if _, valid := regionBaseURLs[lang]; valid {
			region = lang
		} else {
			writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("unsupported language/region %q", lang))
			return
		}
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

	collectionDoc, err := h.ensureCategoryDocument(r.Context(), category, region, newsType)
	if err != nil {
		log.Printf("news: failed to load cached data for %s (%s): %v", category, region, err)
		writeJSONError(w, http.StatusServiceUnavailable, "news cache unavailable")
		return
	}

	rows := paginateRows(collectionDoc.Rows, index, size)

	payload := newsListResponse{
		Code:    0,
		Message: "ok",
		Data: newsListData{
			Count: len(rows),
			Rows:  rows,
		},
		Timestamp: json.Number(strconv.FormatInt(time.Now().UnixMilli(), 10)),
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("failed to write news response: %v", err)
	}
}

type newsCategoryDocument struct {
	Category  string    `bson:"category"`
	Rows      []bson.M  `bson:"rows"`
	UpdatedAt time.Time `bson:"updatedAt"`
}

func paginateRows(rows []bson.M, index, size int) []map[string]interface{} {
	if len(rows) == 0 {
		return []map[string]interface{}{}
	}

	if index <= 0 {
		index = 1
	}
	if size <= 0 {
		size = len(rows)
	}

	start := (index - 1) * size
	if start >= len(rows) {
		return []map[string]interface{}{}
	}

	end := start + size
	if end > len(rows) {
		end = len(rows)
	}

	paged := make([]map[string]interface{}, end-start)
	for i := start; i < end; i++ {
		paged[i-start] = rows[i]
	}

	return paged
}

func (h *Handler) ensureCategoryDocument(ctx context.Context, category, region, newsType string) (newsCategoryDocument, error) {
	dbCategory := fmt.Sprintf("%s:%s", region, category)
	doc, err := h.loadCategoryDocument(ctx, dbCategory)
	if err == nil {
		return doc, nil
	}

	if errors.Is(err, mongo.ErrNoDocuments) {
		if refreshErr := h.refreshCategory(ctx, category, region, newsType); refreshErr != nil {
			return newsCategoryDocument{}, refreshErr
		}
		return h.loadCategoryDocument(ctx, dbCategory)
	}

	return newsCategoryDocument{}, err
}

func (h *Handler) loadCategoryDocument(ctx context.Context, dbCategory string) (newsCategoryDocument, error) {
	collection := h.newsCollection()
	if collection == nil {
		return newsCategoryDocument{}, errors.New("mongo client not initialised")
	}

	childCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var doc newsCategoryDocument
	err := collection.FindOne(childCtx, bson.M{"category": dbCategory}).Decode(&doc)
	return doc, err
}

func (h *Handler) refreshCategory(ctx context.Context, category, region, newsType string) error {
	rows, err := h.fetchCategoryRows(ctx, region, newsType)
	if err != nil {
		return err
	}

	normalized := normalizeRows(rows)
	collection := h.newsCollection()
	if collection == nil {
		return errors.New("mongo client not initialised")
	}

	childCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	dbCategory := fmt.Sprintf("%s:%s", region, category)
	update := bson.M{
		"$set": bson.M{
			"category":  dbCategory,
			"rows":      normalized,
			"updatedAt": time.Now().UTC(),
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err = collection.UpdateOne(childCtx, bson.M{"category": dbCategory}, update, opts)
	return err
}

func (h *Handler) fetchCategoryRows(ctx context.Context, region, newsType string) ([]map[string]interface{}, error) {
	allRows := make([]map[string]interface{}, 0)
	page := 1

	for {
		rows, upstreamCount, err := h.fetchNewsPage(ctx, region, newsType, page, newsSyncPageSize)
		if err != nil {
			return nil, err
		}

		allRows = append(allRows, rows...)
		if upstreamCount < newsSyncPageSize {
			break
		}

		page++
	}

	return allRows, nil
}

func (h *Handler) fetchNewsPage(ctx context.Context, region, newsType string, index, size int) ([]map[string]interface{}, int, error) {
	endpoint, err := buildNewsListURL(region, newsType, index, size)
	if err != nil {
		return nil, 0, err
	}

	childCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(childCtx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("upstream status %d", resp.StatusCode)
	}

	decoder := json.NewDecoder(resp.Body)
	decoder.UseNumber()

	var payload newsListResponse
	if err := decoder.Decode(&payload); err != nil {
		return nil, 0, err
	}

	upstreamCount := len(payload.Data.Rows)
	filteredRows := filterRowsByType(payload.Data.Rows, newsType)

	if err := h.enrichThumbnails(ctx, region, filteredRows); err != nil {
		log.Printf("news: thumbnail enrichment failed: %v", err)
	}

	return filteredRows, upstreamCount, nil
}

func normalizeRows(rows []map[string]interface{}) []bson.M {
	normalized := make([]bson.M, len(rows))
	for i, row := range rows {
		normalized[i] = convertMap(row)
	}
	return normalized
}

func convertMap(row map[string]interface{}) bson.M {
	normalized := make(bson.M, len(row))
	for k, v := range row {
		normalized[k] = convertJSONValue(v)
	}
	return normalized
}

func convertJSONValue(v interface{}) interface{} {
	switch val := v.(type) {
	case json.Number:
		if i, err := val.Int64(); err == nil {
			return i
		}
		if f, err := val.Float64(); err == nil {
			return f
		}
		return val.String()
	case map[string]interface{}:
		mapped := make(map[string]interface{}, len(val))
		for k, nested := range val {
			mapped[k] = convertJSONValue(nested)
		}
		return mapped
	case []interface{}:
		arr := make([]interface{}, len(val))
		for i, nested := range val {
			arr[i] = convertJSONValue(nested)
		}
		return arr
	default:
		return val
	}
}

func (h *Handler) newsCollection() *mongo.Collection {
	if h.app == nil {
		return nil
	}

	client := h.app.MongoClient()
	if client == nil {
		return nil
	}

	return client.Database(h.dbName).Collection(newsCollectionName)
}

func (h *Handler) startSyncLoop() {
	if h.app == nil {
		return
	}

	h.syncOnce.Do(func() {
		scheduler := gocron.NewScheduler()
		scheduler.ChangeLoc(time.UTC)

		start := nextHalfHour(time.Now().UTC())
		job := scheduler.Every(30).Minutes().From(&start)
		if err := job.Do(func() {
			if err := h.refreshAll(context.Background()); err != nil {
				log.Printf("news: scheduled sync failed: %v", err)
			}
		}); err != nil {
			log.Printf("news: failed to schedule sync job: %v", err)
			return
		}

		h.scheduler = scheduler
		h.stopSignal = h.scheduler.Start()
	})
}

func (h *Handler) refreshAll(ctx context.Context) error {
	var errs []error

	for region := range regionBaseURLs {
		for category, newsType := range categoryTypeMap {
			if err := h.refreshCategory(ctx, category, region, newsType); err != nil {
				errs = append(errs, fmt.Errorf("%s (%s): %w", category, region, err))
			}
		}
	}

	return errors.Join(errs...)
}

func nextHalfHour(now time.Time) time.Time {
	start := now.Truncate(30 * time.Minute)
	if !start.After(now) {
		start = start.Add(30 * time.Minute)
	}
	return start
}

func (h *Handler) enrichThumbnails(ctx context.Context, region string, rows []map[string]interface{}) error {
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

			_, thumbnail, err := h.fetchNewsDetail(ctx, region, int(id))
			if err != nil || thumbnail == "" {
				return nil
			}

			row["thumbnail"] = thumbnail
			return nil
		})
	}

	return g.Wait()
}

func (h *Handler) fetchNewsDetail(ctx context.Context, region string, id int) (newsDetail, string, error) {
	if detail, hero, ok := h.cachedNews(region, id); ok {
		return detail, hero, nil
	}

	childCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	endpoint, err := buildNewsDetailURL(region, id)
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

	h.storeNews(region, id, detail, hero)
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

func buildNewsListURL(region, newsType string, index, size int) (string, error) {
	base, ok := regionBaseURLs[region]
	if !ok {
		return "", fmt.Errorf("unknown region %q", region)
	}

	values := url.Values{}
	values.Set("type", newsType)
	values.Set("index", strconv.Itoa(index))
	values.Set("size", strconv.Itoa(size))

	endpoint, err := url.Parse(base + newsListPath)
	if err != nil {
		return "", err
	}

	endpoint.RawQuery = values.Encode()
	return endpoint.String(), nil
}

func buildNewsDetailURL(region string, id int) (string, error) {
	base, ok := regionBaseURLs[region]
	if !ok {
		return "", fmt.Errorf("unknown region %q", region)
	}

	endpoint, err := url.Parse(base + newsDetailPath)
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

func (h *Handler) cachedNews(region string, id int) (newsDetail, string, bool) {
	key := fmt.Sprintf("%s:%d", region, id)
	h.cacheMu.RLock()
	entry, ok := h.cache[key]
	h.cacheMu.RUnlock()
	if !ok {
		return newsDetail{}, "", false
	}

	if time.Now().After(entry.expires) {
		h.cacheMu.Lock()
		delete(h.cache, key)
		h.cacheMu.Unlock()
		return newsDetail{}, "", false
	}

	return entry.detail, entry.heroThumbnail, true
}

func (h *Handler) storeNews(region string, id int, detail newsDetail, hero string) {
	if hero == "" {
		hero = detail.Thumbnail
	}

	key := fmt.Sprintf("%s:%d", region, id)
	h.cacheMu.Lock()
	h.cache[key] = cacheEntry{
		detail:        detail,
		heroThumbnail: hero,
		expires:       time.Now().Add(thumbnailCacheTTL),
	}
	h.cacheMu.Unlock()
}
