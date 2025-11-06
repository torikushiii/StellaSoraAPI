package banner

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"ss-api/internal/app"
)

type Handler struct {
	app    *app.App
	dbName string
}

type bannerDocument struct {
	Entries []bannerEntry `bson:"entries"`
}

type bannerEntry struct {
	ID         int             `bson:"id" json:"id"`
	Name       string          `bson:"name" json:"name"`
	BannerType *string         `bson:"bannerType" json:"bannerType"`
	Start      *string         `bson:"startTime" json:"startTime"`
	End        *string         `bson:"endTime" json:"endTime"`
	RateUp     bannerRateUpSet `bson:"rateUp" json:"rateUp"`
}

type bannerRateUpSet struct {
	FiveStar *bannerRateUpPool `bson:"fiveStar" json:"fiveStar"`
	FourStar *bannerRateUpPool `bson:"fourStar" json:"fourStar"`
}

type bannerRateUpPool struct {
	PackageID int                `bson:"packageId" json:"packageId"`
	Entries   []bannerFocusEntry `bson:"entries" json:"entries"`
}

type bannerFocusEntry struct {
	ID   int     `bson:"id" json:"id"`
	Name *string `bson:"name" json:"name"`
}

type groupedBanners struct {
	Current  []bannerEntry `json:"current"`
	Upcoming []bannerEntry `json:"upcoming"`
	Ended    []bannerEntry `json:"ended"`
}

func New(appInstance *app.App) http.HandlerFunc {
	h := Handler{
		app:    appInstance,
		dbName: appInstance.DatabaseName(),
	}

	return h.handle
}

func (h Handler) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	client := h.app.MongoClient()
	if client == nil {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}

	collection := client.Database(h.dbName).Collection("gacha")

	lang := strings.TrimSpace(r.URL.Query().Get("lang"))
	if lang == "" {
		lang = "EN"
	}
	lang = strings.ToUpper(lang)

	cursor, err := collection.Find(ctx, bson.D{{Key: "region", Value: lang}})
	if err != nil {
		writeServerError(w, err)
		return
	}
	defer cursor.Close(ctx)

	results := make([]bannerEntry, 0)

	for cursor.Next(ctx) {
		var doc bannerDocument
		if err := cursor.Decode(&doc); err != nil {
			writeServerError(w, err)
			return
		}

		results = append(results, doc.Entries...)
	}

	if err := cursor.Err(); err != nil {
		writeServerError(w, err)
		return
	}

	if len(results) == 0 {
		writeNotFound(w, "no banner data found")
		return
	}

	response := categorizeBanners(results, time.Now().UTC())

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("failed to write response: %v", err)
	}
}

func categorizeBanners(entries []bannerEntry, reference time.Time) groupedBanners {
	grouped := groupedBanners{}

	for _, entry := range entries {
		start := parseTimestamp(entry.Start)
		end := parseTimestamp(entry.End)

		switch {
		case end != nil && reference.After(*end):
			grouped.Ended = append(grouped.Ended, entry)
		case start != nil && reference.Before(*start):
			grouped.Upcoming = append(grouped.Upcoming, entry)
		default:
			grouped.Current = append(grouped.Current, entry)
		}
	}

	return grouped
}

func parseTimestamp(raw *string) *time.Time {
	if raw == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*raw)
	if trimmed == "" {
		return nil
	}

	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		log.Printf("banner: failed to parse time %q: %v", trimmed, err)
		return nil
	}

	return &parsed
}

func writeServerError(w http.ResponseWriter, err error) {
	log.Printf("internal server error: %v", err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}

func writeNotFound(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}
