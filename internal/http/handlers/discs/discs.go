package discs

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"

	"ss-api/internal/app"
)

type Handler struct {
	app    *app.App
	dbName string
	omit   map[string]struct{}
	order  []string
}

func New(appInstance *app.App) http.HandlerFunc {
	h := newHandler(
		appInstance,
		map[string]struct{}{
			"tag":             {},
			"mainSkill":       {},
			"secondarySkills": {},
			"supportNote":     {},
			"stats":           {},
			"dupe":            {},
			"upgrades":        {},
		},
	)

	return h.handleList
}

func NewDetail(appInstance *app.App) http.HandlerFunc {
	h := newHandler(appInstance, nil)
	return h.handleDetail
}

func newHandler(appInstance *app.App, omit map[string]struct{}) Handler {
	return Handler{
		app:    appInstance,
		dbName: appInstance.DatabaseName(),
		omit:   omit,
		order: []string{
			"id",
			"name",
			"star",
			"element",
			"tag",
			"mainSkill",
			"secondarySkills",
			"supportNote",
			"stats",
			"dupe",
			"upgrades",
		},
	}
}

func (h Handler) handleList(w http.ResponseWriter, r *http.Request) {
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

	collection := client.Database(h.dbName).Collection("discs")

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

	entries := make([]orderedDocument, 0)

	for cursor.Next(ctx) {
		doc := cursor.Current
		entriesValue := doc.Lookup("entries")
		if entriesValue.Type != bsontype.Array {
			continue
		}

		sanitized, err := h.sanitizeEntries(entriesValue)
		if err != nil {
			writeServerError(w, err)
			return
		}

		if len(sanitized) == 0 {
			continue
		}

		entries = append(entries, sanitized...)
	}

	if err := cursor.Err(); err != nil {
		writeServerError(w, err)
		return
	}

	if len(entries) == 0 {
		writeNotFound(w, "no disc data found")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(entries); err != nil {
		log.Printf("failed to write response: %v", err)
	}
}

func (h Handler) handleDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	identifier := strings.TrimSpace(r.PathValue("identifier"))
	if identifier == "" {
		http.Error(w, "missing disc identifier", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	client := h.app.MongoClient()
	if client == nil {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}

	collection := client.Database(h.dbName).Collection("discs")

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

	var result orderedDocument
	found := false

	for cursor.Next(ctx) {
		doc := cursor.Current
		entriesValue := doc.Lookup("entries")
		if entriesValue.Type != bsontype.Array {
			continue
		}

		entry, ok, err := h.findEntry(entriesValue, identifier)
		if err != nil {
			writeServerError(w, err)
			return
		}
		if ok {
			result = entry
			found = true
			break
		}
	}

	if err := cursor.Err(); err != nil {
		writeServerError(w, err)
		return
	}

	if !found {
		writeNotFound(w, "disc not found")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Printf("failed to write response: %v", err)
	}
}

func (h Handler) sanitizeEntries(raw bson.RawValue) ([]orderedDocument, error) {
	arrayRaw := raw.Array()
	values, err := arrayRaw.Values()
	if err != nil {
		return nil, err
	}

	results := make([]orderedDocument, 0, len(values))

	for _, value := range values {
		if value.Type != bsontype.EmbeddedDocument {
			continue
		}

		docRaw := value.Document()

		doc, err := h.convertDocument(docRaw)
		if err != nil {
			return nil, err
		}

		results = append(results, doc)
	}

	return results, nil
}

func (h Handler) findEntry(raw bson.RawValue, identifier string) (orderedDocument, bool, error) {
	arrayRaw := raw.Array()
	values, err := arrayRaw.Values()
	if err != nil {
		return orderedDocument{}, false, err
	}

	trimmed := strings.TrimSpace(identifier)

	numericValue, numericErr := strconv.ParseInt(trimmed, 10, 64)
	hasNumeric := numericErr == nil

	for _, value := range values {
		if value.Type != bsontype.EmbeddedDocument {
			continue
		}

		docRaw := value.Document()
		if entryMatches(docRaw, trimmed, hasNumeric, numericValue) {
			doc, err := h.convertDocument(docRaw)
			if err != nil {
				return orderedDocument{}, false, err
			}
			return doc, true, nil
		}
	}

	return orderedDocument{}, false, nil
}

func entryMatches(doc bson.Raw, identifier string, hasNumeric bool, numeric int64) bool {
	if hasNumeric {
		if idMatches(doc.Lookup("id"), numeric) {
			return true
		}
	}

	nameVal := doc.Lookup("name")
	if nameVal.Type == bsontype.String {
		return strings.EqualFold(strings.TrimSpace(nameVal.StringValue()), identifier)
	}

	return false
}

func idMatches(value bson.RawValue, target int64) bool {
	switch value.Type {
	case bsontype.Int32:
		return int64(value.Int32()) == target
	case bsontype.Int64:
		return value.Int64() == target
	case bsontype.Double:
		return int64(value.Double()) == target
	case bsontype.Decimal128:
		if dec, ok := value.Decimal128OK(); ok {
			if parsed, err := strconv.ParseInt(dec.String(), 10, 64); err == nil {
				return parsed == target
			}
		}
	case bsontype.String:
		if parsed, err := strconv.ParseInt(strings.TrimSpace(value.StringValue()), 10, 64); err == nil {
			return parsed == target
		}
	}
	return false
}

func (h Handler) convertDocument(raw bson.Raw) (orderedDocument, error) {
	elements, err := raw.Elements()
	if err != nil {
		return orderedDocument{}, err
	}

	pairs := make([]keyValue, 0, len(elements))

	for _, elem := range elements {
		key := elem.Key()
		if _, skip := h.omit[key]; skip {
			continue
		}

		value, err := h.convertRawValue(elem.Value())
		if err != nil {
			return orderedDocument{}, err
		}

		pairs = append(pairs, keyValue{key: key, value: value})
	}

	ordered := h.reorderPairs(pairs)

	return orderedDocument{pairs: ordered}, nil
}

func (h Handler) convertRawValue(rv bson.RawValue) (any, error) {
	switch rv.Type {
	case bsontype.EmbeddedDocument:
		docRaw := rv.Document()
		return h.convertDocument(docRaw)
	case bsontype.Array:
		arrayRaw := rv.Array()
		values, err := arrayRaw.Values()
		if err != nil {
			return nil, err
		}

		result := make([]any, 0, len(values))
		for _, value := range values {
			converted, err := h.convertRawValue(value)
			if err != nil {
				return nil, err
			}
			result = append(result, converted)
		}
		return result, nil
	default:
		var generic any
		if err := rv.Unmarshal(&generic); err != nil {
			return nil, err
		}
		return generic, nil
	}
}

func (h Handler) reorderPairs(pairs []keyValue) []keyValue {
	if len(pairs) == 0 {
		return pairs
	}

	ordered := make([]keyValue, 0, len(pairs))
	used := make([]bool, len(pairs))

	for _, key := range h.order {
		for i, kv := range pairs {
			if !used[i] && kv.key == key {
				ordered = append(ordered, kv)
				used[i] = true
				break
			}
		}
	}

	for i, kv := range pairs {
		if !used[i] {
			ordered = append(ordered, kv)
		}
	}

	return ordered
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

type orderedDocument struct {
	pairs []keyValue
}

type keyValue struct {
	key   string
	value any
}

func (d orderedDocument) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')

	for i, kv := range d.pairs {
		if i > 0 {
			buf.WriteByte(',')
		}

		keyBytes, err := json.Marshal(kv.key)
		if err != nil {
			return nil, err
		}
		buf.Write(keyBytes)
		buf.WriteByte(':')

		valueBytes, err := json.Marshal(kv.value)
		if err != nil {
			return nil, err
		}
		buf.Write(valueBytes)
	}

	buf.WriteByte('}')
	return buf.Bytes(), nil
}
