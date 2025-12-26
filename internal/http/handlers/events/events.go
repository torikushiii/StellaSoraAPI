package events

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"ss-api/internal/alias"
	"ss-api/internal/app"
)

type Handler struct {
	app    *app.App
	dbName string
}

type eventDocument struct {
	Entries []eventEntry `bson:"entries"`
}

type eventEntry struct {
	ID          int              `bson:"id" json:"id"`
	Title       *string          `bson:"title" json:"title"`
	Description *string          `bson:"description" json:"description"`
	Start       *string          `bson:"startTime" json:"startTime"`
	End         *string          `bson:"endTime" json:"endTime"`
	ClaimEnd    *string          `bson:"claimEndTime" json:"claimEndTime"`
	Textures    *eventAssets     `bson:"textures" json:"textures,omitempty"`
	Rewards     []eventReward    `bson:"rewards" json:"rewards"`
	Milestones  []eventMilestone `bson:"milestones" json:"milestones"`
	Shops       []eventShop      `bson:"shops" json:"shops"`
}

type eventReward struct {
	ID       int     `bson:"id" json:"id"`
	Name     *string `bson:"name" json:"name"`
	Category string  `bson:"category" json:"category"`
	Quantity *int    `bson:"quantity,omitempty" json:"quantity,omitempty"`
}

type eventMilestone struct {
	Description *string       `bson:"description" json:"description"`
	Rewards     []eventReward `bson:"rewards" json:"rewards"`
}

type eventShop struct {
	ID       int               `bson:"id" json:"id"`
	Name     *string           `bson:"name" json:"name"`
	Currency *eventItemSummary `bson:"currency" json:"currency"`
	Goods    []eventShopGood   `bson:"goods" json:"goods"`
}

type eventShopGood struct {
	ID          int               `bson:"id" json:"id"`
	Order       *int              `bson:"order" json:"order"`
	Name        *string           `bson:"name" json:"name"`
	Description *string           `bson:"description" json:"description"`
	ItemID      *int              `bson:"itemId" json:"itemId"`
	Quantity    *int              `bson:"quantity" json:"quantity"`
	Price       *int              `bson:"price" json:"price"`
	Currency    *eventItemSummary `bson:"currency" json:"currency"`
	Limit       *int              `bson:"limit" json:"limit"`
}

type eventAssets struct {
	Banner        *string       `bson:"banner" json:"banner"`
	TabBackground *string       `bson:"tabBackground" json:"tabBackground"`
	friendly      friendlyAtlas `bson:"friendly" json:"-"`
}

type friendlyAtlas struct {
	Banner        string `bson:"banner" json:"banner"`
	TabBackground string `bson:"tabBackground" json:"tabBackground"`
}

func (a *eventAssets) normalize() *eventAssets {
	if a == nil {
		return nil
	}

	normalized := *a
	normalized.Banner = resolveAssetPath(normalized.friendly.Banner, normalized.Banner)
	normalized.TabBackground = resolveAssetPath(normalized.friendly.TabBackground, normalized.TabBackground)

	return &normalized
}

func resolveAssetPath(friendlyAlias string, raw *string) *string {
	var path string

	switch {
	case friendlyAlias != "":
		path = alias.PathFromAlias(friendlyAlias)
	case raw != nil:
		path = alias.PathFromSource(*raw)
	default:
		return nil
	}

	if path == "" {
		return nil
	}

	val := path
	return &val
}

type eventItemSummary struct {
	ItemID      int     `bson:"itemId" json:"itemId"`
	Name        *string `bson:"name" json:"name"`
	Description *string `bson:"description" json:"description"`
	Flavor      *string `bson:"flavor" json:"flavor"`
}

type groupedEvents struct {
	Current  []eventEntry `json:"current"`
	Upcoming []eventEntry `json:"upcoming"`
	Ended    []eventEntry `json:"ended"`
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

	collection := client.Database(h.dbName).Collection("events")

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

	results := make([]eventEntry, 0)

	for cursor.Next(ctx) {
		var doc eventDocument
		if err := cursor.Decode(&doc); err != nil {
			writeServerError(w, err)
			return
		}

		if len(doc.Entries) == 0 {
			continue
		}

		for _, entry := range doc.Entries {
			entry.Textures = entry.Textures.normalize()
			results = append(results, entry)
		}
	}

	if err := cursor.Err(); err != nil {
		writeServerError(w, err)
		return
	}

	if len(results) == 0 {
		writeNotFound(w, "no event data found")
		return
	}

	grouped := categorizeEvents(results, time.Now().UTC())

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(grouped); err != nil {
		log.Printf("failed to write response: %v", err)
	}
}

func categorizeEvents(entries []eventEntry, reference time.Time) groupedEvents {
	grouped := groupedEvents{
		Current:  make([]eventEntry, 0),
		Upcoming: make([]eventEntry, 0),
		Ended:    make([]eventEntry, 0),
	}

	for _, entry := range entries {
		start := parseTimestamp(entry.Start)
		end := parseTimestamp(entry.End)

		switch {
		case end != nil && !reference.Before(*end):
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
		log.Printf("events: failed to parse time %q: %v", trimmed, err)
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
