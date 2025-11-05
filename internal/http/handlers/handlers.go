package handlers

import (
	"net/http"

	"ss-api/internal/app"
	"ss-api/internal/http/handlers/banner"
	"ss-api/internal/http/handlers/characters"
	"ss-api/internal/http/handlers/discs"
	"ss-api/internal/http/handlers/status"
)

type Set struct {
	Status          http.HandlerFunc
	Characters      http.HandlerFunc
	CharacterDetail http.HandlerFunc
	Discs           http.HandlerFunc
	DiscDetail      http.HandlerFunc
	Banner          http.HandlerFunc
}

func New(appInstance *app.App) Set {
	return Set{
		Status:          status.New(appInstance),
		Characters:      characters.New(appInstance),
		CharacterDetail: characters.NewDetail(appInstance),
		Discs:           discs.New(appInstance),
		DiscDetail:      discs.NewDetail(appInstance),
		Banner:          banner.New(appInstance),
	}
}
