package characters

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"

	"ss-api/internal/alias"
	"ss-api/internal/app"
)

const characterCacheTTL = 30 * time.Minute

type Handler struct {
	app             *app.App
	dbName          string
	omit            map[string]struct{}
	order           []string
	icon            bool
	flattenTextures bool
	listCache       *responseCache
	detailCache     *responseCache
}

func New(appInstance *app.App) http.HandlerFunc {
	h := newHandler(
		appInstance,
		map[string]struct{}{
			"stats":           {},
			"normalAttack":    {},
			"skill":           {},
			"supportSkill":    {},
			"ultimate":        {},
			"potentials":      {},
			"talents":         {},
			"upgrades":        {},
			"skillUpgrades":   {},
			"dateEvents":      {},
			"giftPreferences": {},
			"textures":        {},
		},
		true,
		false,
	)

	return h.handleList
}

func NewDetail(appInstance *app.App) http.HandlerFunc {
	h := newHandler(appInstance, nil, false, true)
	return h.handleDetail
}

func newHandler(appInstance *app.App, omit map[string]struct{}, injectIcon bool, flattenTextures bool) Handler {
	return Handler{
		app:             appInstance,
		dbName:          appInstance.DatabaseName(),
		omit:            omit,
		icon:            injectIcon,
		flattenTextures: flattenTextures,
		order: []string{
			"id",
			"name",
			"icon",
			"portrait",
			"background",
			"variants",
			"description",
			"voiceActor",
			"birthday",
			"grade",
			"element",
			"position",
			"attackType",
			"style",
			"faction",
			"tags",
			"dateEvents",
			"giftPreferences",
			"normalAttack",
			"skill",
			"supportSkill",
			"ultimate",
			"potentials",
			"talents",
			"stats",
			"upgrades",
			"skillUpgrades",
		},
		listCache:   newResponseCache(characterCacheTTL),
		detailCache: newResponseCache(characterCacheTTL),
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

	collection := client.Database(h.dbName).Collection("characters")

	lang := strings.TrimSpace(r.URL.Query().Get("lang"))
	if lang == "" {
		lang = "EN"
	}

	lang = strings.ToUpper(lang)

	if payload, ok := h.listCache.get(lang); ok {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if _, err := w.Write(payload); err != nil {
			log.Printf("failed to write response: %v", err)
		}
		return
	}

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
		writeNotFound(w, "no character data found")
		return
	}

	responseBytes, err := json.Marshal(entries)
	if err != nil {
		writeServerError(w, err)
		return
	}

	h.listCache.set(lang, responseBytes)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if _, err := w.Write(responseBytes); err != nil {
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
		http.Error(w, "missing character identifier", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	client := h.app.MongoClient()
	if client == nil {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}

	collection := client.Database(h.dbName).Collection("characters")

	lang := strings.TrimSpace(r.URL.Query().Get("lang"))
	if lang == "" {
		lang = "EN"
	}
	lang = strings.ToUpper(lang)

	cacheKey := detailCacheKey(lang, identifier)
	if payload, ok := h.detailCache.get(cacheKey); ok {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if _, err := w.Write(payload); err != nil {
			log.Printf("failed to write response: %v", err)
		}
		return
	}

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
		writeNotFound(w, "character not found")
		return
	}

	responseBytes, err := json.Marshal(result)
	if err != nil {
		writeServerError(w, err)
		return
	}

	h.detailCache.set(cacheKey, responseBytes)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if _, err := w.Write(responseBytes); err != nil {
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

func numericFromRawValue(value bson.RawValue) (int64, bool) {
	switch value.Type {
	case bsontype.Int32:
		return int64(value.Int32()), true
	case bsontype.Int64:
		return value.Int64(), true
	case bsontype.Double:
		return int64(value.Double()), true
	case bsontype.Decimal128:
		if dec, ok := value.Decimal128OK(); ok {
			if parsed, err := strconv.ParseInt(dec.String(), 10, 64); err == nil {
				return parsed, true
			}
		}
	case bsontype.String:
		if parsed, err := strconv.ParseInt(strings.TrimSpace(value.StringValue()), 10, 64); err == nil {
			return parsed, true
		}
	}
	return 0, false
}

func (h Handler) convertDocument(raw bson.Raw) (orderedDocument, error) {
	elements, err := raw.Elements()
	if err != nil {
		return orderedDocument{}, err
	}

	pairs := make([]keyValue, 0, len(elements))
	var nameValue string
	var texturePairs []keyValue
	var idValue int64

	for _, elem := range elements {
		key := elem.Key()
		if _, skip := h.omit[key]; skip {
			continue
		}

		rawValue := elem.Value()
		value, err := h.convertRawValue(rawValue)
		if err != nil {
			return orderedDocument{}, err
		}

		switch key {
		case "name":
			if str, ok := value.(string); ok {
				nameValue = str
			}
		case "id":
			if parsed, ok := numericFromRawValue(rawValue); ok {
				idValue = parsed
			}
		case "voiceActors":
			if doc, ok := value.(orderedDocument); ok {
				if actors := voiceActorMap(doc); len(actors) > 0 {
					pairs = append(pairs, keyValue{key: "voiceActor", value: actors})
				}
			}
			continue
		}

		if h.flattenTextures && key == "textures" {
			if doc, ok := value.(orderedDocument); ok {
				texturePairs = buildFriendlyTexturePairs(doc)
			}
			continue
		}

		pairs = append(pairs, keyValue{key: key, value: value})
	}

	if len(texturePairs) > 0 {
		pairs = append(pairs, texturePairs...)
	}

	ordered := h.reorderPairs(pairs)

	if h.icon {
		iconPath := alias.IconPath(nameValue)
		var portraitPath string
		if idValue > 0 {
			portraitPath = alias.HeadPortraitPath(idValue)
		}
		ordered = insertCharacterTextureFields(ordered, iconPath, portraitPath)
	}

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

func insertCharacterTextureFields(pairs []keyValue, icon, portrait string) []keyValue {
	hasIcon := icon != ""
	hasPortrait := portrait != ""
	if !hasIcon && !hasPortrait {
		return pairs
	}

	insertPos := 0
	for i, kv := range pairs {
		if kv.key == "name" {
			insertPos = i + 1
			break
		}
	}

	capacity := len(pairs)
	if hasIcon {
		capacity++
	}
	if hasPortrait {
		capacity++
	}

	result := make([]keyValue, 0, capacity)
	result = append(result, pairs[:insertPos]...)
	if hasIcon {
		result = append(result, keyValue{key: "icon", value: icon})
	}
	if hasPortrait {
		result = append(result, keyValue{key: "portrait", value: portrait})
	}
	result = append(result, pairs[insertPos:]...)

	return result
}

func buildFriendlyTexturePairs(doc orderedDocument) []keyValue {
	friendlyDoc, ok := lookupOrderedDocument(doc, "friendly")
	if !ok || len(friendlyDoc.pairs) == 0 {
		return nil
	}

	result := make([]keyValue, 0, 4)

	if icon, ok := lookupString(friendlyDoc, "icon"); ok {
		if path := alias.PathFromAlias(icon); path != "" {
			result = append(result, keyValue{key: "icon", value: path})
		}
	}

	if portrait, ok := lookupString(friendlyDoc, "portrait"); ok {
		if path := alias.PathFromAlias(portrait); path != "" {
			result = append(result, keyValue{key: "portrait", value: path})
		}
	}

	if background, ok := lookupString(friendlyDoc, "background"); ok {
		if path := alias.PathFromAlias(background); path != "" {
			result = append(result, keyValue{key: "background", value: path})
		}
	}

	if variantsDoc, ok := lookupOrderedDocument(friendlyDoc, "variants"); ok {
		result = append(result, keyValue{key: "variants", value: pathifyOrderedDocument(variantsDoc)})
	}

	return result
}

func pathifyOrderedDocument(doc orderedDocument) orderedDocument {
	if len(doc.pairs) == 0 {
		return doc
	}

	pairs := copyKeyValues(doc.pairs)
	for i, kv := range pairs {
		switch v := kv.value.(type) {
		case string:
			pairs[i].value = alias.PathFromAlias(v)
		case orderedDocument:
			pairs[i].value = pathifyOrderedDocument(v)
		}
	}

	return orderedDocument{pairs: pairs}
}

func lookupOrderedDocument(doc orderedDocument, key string) (orderedDocument, bool) {
	for _, kv := range doc.pairs {
		if kv.key == key {
			if subDoc, ok := kv.value.(orderedDocument); ok {
				return subDoc, true
			}
			break
		}
	}
	return orderedDocument{}, false
}

func lookupString(doc orderedDocument, key string) (string, bool) {
	for _, kv := range doc.pairs {
		if kv.key == key {
			if str, ok := kv.value.(string); ok && str != "" {
				return str, true
			}
			break
		}
	}
	return "", false
}

func copyKeyValues(src []keyValue) []keyValue {
	dst := make([]keyValue, len(src))
	copy(dst, src)
	return dst
}

func voiceActorMap(doc orderedDocument) map[string]string {
	if len(doc.pairs) == 0 {
		return nil
	}

	result := make(map[string]string, len(doc.pairs))
	for _, kv := range doc.pairs {
		if name, ok := kv.value.(string); ok && name != "" {
			result[kv.key] = name
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
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

func detailCacheKey(lang, identifier string) string {
	normalizedLang := strings.ToUpper(strings.TrimSpace(lang))
	if normalizedLang == "" {
		normalizedLang = "EN"
	}
	return normalizedLang + "|" + normalizeIdentifierKey(identifier)
}

func normalizeIdentifierKey(identifier string) string {
	trimmed := strings.TrimSpace(identifier)
	if trimmed == "" {
		return ""
	}
	if numericValue, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		return "#" + strconv.FormatInt(numericValue, 10)
	}
	return strings.ToLower(trimmed)
}

type responseCache struct {
	ttl     time.Duration
	mu      sync.RWMutex
	entries map[string]cachedResponse
}

type cachedResponse struct {
	data    []byte
	expires time.Time
}

func newResponseCache(ttl time.Duration) *responseCache {
	return &responseCache{
		ttl:     ttl,
		entries: make(map[string]cachedResponse),
	}
}

func (c *responseCache) get(key string) ([]byte, bool) {
	if c == nil || key == "" {
		return nil, false
	}

	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}

	if time.Now().After(entry.expires) {
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		return nil, false
	}

	return entry.data, true
}

func (c *responseCache) set(key string, data []byte) {
	if c == nil || key == "" || len(data) == 0 {
		return
	}

	payload := make([]byte, len(data))
	copy(payload, data)

	c.mu.Lock()
	c.entries[key] = cachedResponse{
		data:    payload,
		expires: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()
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
