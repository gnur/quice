package main

import "time"

// Playlist is a collection of videos
type Playlist struct {
	ID         string
	Videos     map[string]*Video
	Prefixes   []string
	Sorttype   string
	sortedKeys []string
	MaxAge     time.Duration
}

// Video refers to a video
type Video struct {
	Position  int64     `json:"position"`
	Filename  string    `json:"filename"`
	Key       string    `json:"key"`
	Completed bool      `json:"completed"`
	Changed   time.Time `json:"changed"`
}

type database interface {
	GetPlayLists() ([]string, error)
	GetPlaylist(string) ([]Playlist, error)

	SetVideoPosition(string, string, int64) error
	GetVideoPosition(string, string) (int64, error)
}
