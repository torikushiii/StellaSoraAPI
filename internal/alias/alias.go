package alias

import (
	"path"
	"strings"
)

// BaseName normalizes a display name into an underscore-delimited alias.
func BaseName(name string) string {
	var builder strings.Builder
	lastUnderscore := false

	for _, r := range name {
		switch {
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastUnderscore = false
		case r >= 'A' && r <= 'Z':
			builder.WriteRune(r)
			lastUnderscore = false
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
			lastUnderscore = false
		case r == ' ' || r == '-' || r == '_' || r == '.':
			if builder.Len() > 0 && !lastUnderscore {
				builder.WriteByte('_')
				lastUnderscore = true
			}
		default:
			if builder.Len() > 0 && !lastUnderscore {
				builder.WriteByte('_')
				lastUnderscore = true
			}
		}
	}

	return strings.Trim(builder.String(), "_")
}

const assetBasePath = "/stella/assets/"

// IconAlias returns the canonical friendly filename for a character icon.
func IconAlias(name string) string {
	base := BaseName(name)
	if base == "" {
		return ""
	}
	return base + ".png"
}

// IconPath returns the HTTP path to the icon asset for a character.
func IconPath(name string) string {
	return PathFromAlias(IconAlias(name))
}

// PathFromAlias prefixes a friendly filename with the asset base path.
func PathFromAlias(alias string) string {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return ""
	}
	if strings.HasPrefix(alias, assetBasePath) {
		return alias
	}
	return assetBasePath + alias
}

// PathFromSource converts an internal asset path (e.g. "Icon/Outfit/outfit_1001_b")
// into a served asset path under /stella/assets/.
func PathFromSource(source string) string {
	source = strings.TrimSpace(source)
	if source == "" {
		return ""
	}

	base := path.Base(source)
	if base == "" || base == "." {
		return ""
	}

	if !strings.Contains(base, ".") {
		base = base + ".png"
	}

	return PathFromAlias(base)
}
