package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"ss-api/internal/alias"
	"ss-api/internal/app"
)

const (
	defaultAssetsDir = "assets"
	cacheWarmupLang  = "EN"
)

var (
	errAssetNotFound = errors.New("asset not found")
)

type assetHandler struct {
	assetsDir string
	resolver  *assetResolver
	logger    *log.Logger
}

func newAssetHandler(appInstance *app.App, logger *log.Logger) *assetHandler {
	dir := defaultAssetsDir
	if abs, err := filepath.Abs(defaultAssetsDir); err == nil {
		dir = abs
	}

	return &assetHandler{
		assetsDir: dir,
		resolver: &assetResolver{
			app:       appInstance,
			dbName:    appInstance.DatabaseName(),
			assetsDir: dir,
		},
		logger: logger,
	}
}

func (h *assetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	requested := strings.TrimSpace(r.PathValue("path"))
	if requested == "" {
		http.NotFound(w, r)
		return
	}

	normalized := normalizeRequestPath(requested)
	if normalized == "" {
		http.NotFound(w, r)
		return
	}

	if h.tryServePhysical(w, r, normalized) {
		return
	}

	target, err := h.resolver.Resolve(r.Context(), normalized)
	if err != nil {
		if errors.Is(err, errAssetNotFound) {
			http.NotFound(w, r)
			return
		}

		h.logger.Printf("asset resolve error: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	http.ServeFile(w, r, target)
}

func (h *assetHandler) tryServePhysical(w http.ResponseWriter, r *http.Request, name string) bool {
	candidates := candidateFilenames(name)
	for _, candidate := range candidates {
		fullPath := filepath.Join(h.assetsDir, candidate)
		if !strings.HasPrefix(fullPath, h.assetsDir) {
			continue
		}

		info, err := os.Stat(fullPath)
		if err != nil || info.IsDir() {
			continue
		}

		http.ServeFile(w, r, fullPath)
		return true
	}
	return false
}

func normalizeRequestPath(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "/")
	if value == "" {
		return ""
	}
	return value
}

func candidateFilenames(value string) []string {
	base := path.Base(value)
	if base == "." {
		return nil
	}
	base = strings.ReplaceAll(base, "..", "")

	if filepath.Ext(base) != "" {
		return []string{base}
	}

	return []string{base + ".png", base}
}

type assetResolver struct {
	app       *app.App
	dbName    string
	assetsDir string

	mu        sync.RWMutex
	cache     map[string]string
	lastBuilt time.Time
}

func (r *assetResolver) Resolve(ctx context.Context, alias string) (string, error) {
	key := normalizeAlias(alias)
	if key == "" {
		return "", errAssetNotFound
	}

	if path := r.lookup(key); path != "" {
		return path, nil
	}

	if err := r.ensureCache(ctx); err != nil {
		return "", err
	}

	if path := r.lookup(key); path != "" {
		return path, nil
	}

	return "", errAssetNotFound
}

func (r *assetResolver) lookup(key string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.cache == nil {
		return ""
	}
	return r.cache[key]
}

func (r *assetResolver) ensureCache(ctx context.Context) error {
	r.mu.RLock()
	if r.cache != nil {
		r.mu.RUnlock()
		return nil
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.cache != nil {
		return nil
	}

	entries, err := r.fetchCharacterTextures(ctx)
	if err != nil {
		return err
	}

	cache := make(map[string]string, len(entries)*8)
	for _, entry := range entries {
		base := alias.BaseName(entry.Name)
		if base == "" {
			continue
		}

		r.register(cache, base, "", entry.Textures.Icon)
		r.register(cache, base, "portrait", entry.Textures.Portrait)
		r.register(cache, base, "background", entry.Textures.Background)

		for variantKey, variantValue := range entry.Textures.Variants {
			suffix := friendlyVariantSuffix(variantKey)
			variantPath, _ := variantValue.(string)
			r.register(cache, base, suffix, variantPath)
		}
	}

	r.cache = cache
	r.lastBuilt = time.Now()
	return nil
}

func (r *assetResolver) fetchCharacterTextures(ctx context.Context) ([]characterTextureEntry, error) {
	client := r.app.MongoClient()
	if client == nil {
		return nil, errors.New("mongo client not initialised")
	}

	collection := client.Database(r.dbName).Collection("characters")
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cursor, err := collection.Find(dbCtx, bson.D{{Key: "region", Value: cacheWarmupLang}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(dbCtx)

	var results []characterTextureEntry

	for cursor.Next(dbCtx) {
		var doc struct {
			Entries []characterTextureEntry `bson:"entries"`
		}
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		results = append(results, doc.Entries...)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

func (r *assetResolver) register(cache map[string]string, baseName, suffix, source string) {
	baseName = strings.TrimSpace(baseName)
	if baseName == "" {
		return
	}

	target, ok := r.resolvePhysicalPath(source)
	if !ok {
		return
	}

	friendly := baseName
	if suffix != "" {
		friendly = fmt.Sprintf("%s_%s", baseName, suffix)
	}
	friendly = strings.Trim(friendly, "_")
	if friendly == "" {
		return
	}
	friendly = friendly + ".png"

	key := strings.ToLower(friendly)
	if _, exists := cache[key]; exists {
		return
	}

	cache[key] = target
}

func (r *assetResolver) resolvePhysicalPath(source string) (string, bool) {
	source = strings.TrimSpace(source)
	if source == "" {
		return "", false
	}

	base := path.Base(source)
	if base == "." {
		return "", false
	}

	if filepath.Ext(base) == "" {
		base += ".png"
	}

	full := filepath.Join(r.assetsDir, base)
	if _, err := os.Stat(full); err != nil {
		return "", false
	}

	return full, true
}

type characterTextureEntry struct {
	Name     string        `bson:"name"`
	Textures textureSource `bson:"textures"`
}

type textureSource struct {
	Icon       string         `bson:"icon"`
	Portrait   string         `bson:"portrait"`
	Background string         `bson:"background"`
	Variants   map[string]any `bson:"variants"`
}

func normalizeAlias(value string) string {
	value = path.Base(value)
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	if filepath.Ext(value) == "" {
		value += ".png"
	}

	return strings.ToLower(value)
}

func friendlyVariantSuffix(variant string) string {
	switch strings.ToLower(strings.TrimSpace(variant)) {
	case "", "xxl":
		return ""
	case "sk":
		return "portrait"
	default:
		return strings.ToLower(variant)
	}
}
