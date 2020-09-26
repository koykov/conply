package main

type App struct {
	Channels []AppChannel `json:"channels"`
}

type AppChannel struct {
	Id   uint64 `json:"id"`
	Name string `json:"name"`
	Slug string `json:"key"`
}
