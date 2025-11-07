package events

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

type eventDocument struct {
	Entries []eventEntry `bson:"entries"`
}

type eventEntry struct {
	ID          int           `bson:"id" json:"id"`
	Title       *string       `bson:"title" json:"title"`
	Description *string       `bson:"description" json:"description"`
	Start       *string       `bson:"startTime" json:"startTime"`
	End         *string       `bson:"endTime" json:"endTime"`
	ClaimEnd    *string       `bson:"claimEndTime" json:"claimEndTime"`
	Rewards     []eventReward `bson:"rewards" json:"rewards"`
	Shops       []eventShop   `bson:"shops" json:"shops"`
}

type eventReward struct {
	ID       int     `bson:"id" json:"id"`
	Name     *string `bson:"name" json:"name"`
	Category string  `bson:"category" json:"category"`
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

type eventItemSummary struct {
	ItemID      int     `bson:"itemId" json:"itemId"`
	Name        *string `bson:"name" json:"name"`
	Description *string `bson:"description" json:"description"`
	Flavor      *string `bson:"flavor" json:"flavor"`
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

		results = append(results, doc.Entries...)
	}

	if err := cursor.Err(); err != nil {
		writeServerError(w, err)
		return
	}

	if len(results) == 0 {
		writeNotFound(w, "no event data found")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(results); err != nil {
		log.Printf("failed to write response: %v", err)
	}
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
