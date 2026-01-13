package alias

import (
	"context"
	"log"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	characterEnglishNames map[int64]string
	characterNamesMu      sync.RWMutex
)

func InitCharacterNamesFromDB(client *mongo.Client, dbName string) {
	if client == nil {
		log.Println("alias: mongo client is nil, skipping character name initialization")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	collection := client.Database(dbName).Collection("characters")

	cursor, err := collection.Find(ctx, bson.D{{Key: "region", Value: "EN"}})
	if err != nil {
		log.Printf("alias: failed to query EN characters: %v", err)
		return
	}
	defer cursor.Close(ctx)

	names := make(map[int64]string)

	for cursor.Next(ctx) {
		var doc struct {
			Entries []struct {
				ID   int64  `bson:"id"`
				Name string `bson:"name"`
			} `bson:"entries"`
		}

		if err := cursor.Decode(&doc); err != nil {
			log.Printf("alias: failed to decode character document: %v", err)
			continue
		}

		for _, entry := range doc.Entries {
			if entry.ID > 0 && entry.Name != "" && entry.Name != "???" {
				names[entry.ID] = entry.Name
			}
		}
	}

	if err := cursor.Err(); err != nil {
		log.Printf("alias: cursor error: %v", err)
		return
	}

	characterNamesMu.Lock()
	characterEnglishNames = names
	characterNamesMu.Unlock()

	log.Printf("alias: loaded %d character English names from database", len(names))
}

// EnglishNameFromID returns the English name for a character given their ID.
// Returns empty string if the character is not found.
func EnglishNameFromID(id int64) string {
	characterNamesMu.RLock()
	defer characterNamesMu.RUnlock()

	if characterEnglishNames == nil {
		return ""
	}
	if name, ok := characterEnglishNames[id]; ok {
		return name
	}
	return ""
}

// IconPathFromID returns the HTTP path to an icon asset using the character ID.
// It looks up the English name for the ID and uses that to generate the path.
func IconPathFromID(id int64) string {
	name := EnglishNameFromID(id)
	if name == "" {
		return ""
	}
	return IconPath(name)
}
