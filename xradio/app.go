package main

// Minimal application.
type App struct {
	Channels []AppChannel `json:"channels"`
}

// Application channel.
type AppChannel struct {
	Id   uint64 `json:"id"`
	Name string `json:"name"`
	Slug string `json:"key"`
}
