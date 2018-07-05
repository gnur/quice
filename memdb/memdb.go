package memdb

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gnur/quice/config"
	"github.com/gorilla/mux"
	"github.com/minio/minio-go"
	log "github.com/sirupsen/logrus"
)

var (
	episodeRegexp      = regexp.MustCompile(`.*?[sS]?([0-9]{1,2})[xeXE]([0-9]{1,2}).*`)
	episodeShortRegexp = regexp.MustCompile(`.*[^2-9]([0-9]{1})([0-3][0-9])[^0-9p].*`)
)

// Memdb is the in-memory struct that can be saved to S3
type Memdb struct {
	lock    sync.Mutex
	store   *minio.Client
	bucket  string
	conf    *config.Cfg
	Users   map[string]*User
	changes bool
}

// User holds playlists for a user
type User struct {
	Playlists map[string]*Playlist
}

// Playlist is a collection of videos
type Playlist struct {
	Videos     map[string]*Video
	Prefixes   []string
	Sorttype   string
	sortedKeys []string
}

// Video refers to a video
type Video struct {
	Position  int64
	Filename  string
	Key       string
	Completed bool
}

// Init loads the data from S3 and syncs everything periodically
func Init(mc *minio.Client, bucket string, config *config.Cfg, saveInterval time.Duration, refreshInterval time.Duration) *Memdb {
	var m Memdb
	m.store = mc
	m.bucket = bucket
	m.conf = config
	err := m.Load()
	if err != nil {
		log.WithField("error", err).Fatal("could not load quice-db from S3")
	}
	m.refresh()
	go func() {
		for {
			time.Sleep(refreshInterval)
			m.refresh()
		}
	}()
	go func() {
		for {
			time.Sleep(saveInterval)
			m.Save()
		}
	}()
	return &m
}

// Save will store the current in-mem struct to S3
func (m *Memdb) Save() error {
	m.lock.Lock()
	defer m.lock.Unlock()
	if !m.changes {
		log.Debug("no changes were detected, not writing to S3")
		return nil
	}
	log.Info("changes were detected, writing to S3")
	buf := &bytes.Buffer{}
	if err := gob.NewEncoder(buf).Encode(m); err != nil {
		return err
	}
	_, err := m.store.PutObject(m.bucket, ".quice-db.dat", buf, -1, minio.PutObjectOptions{})
	m.changes = false
	return err
}

// Load will load the current S3 struct to memory
func (m *Memdb) Load() error {
	m.lock.Lock()
	defer m.lock.Unlock()

	object, err := m.store.GetObject(m.bucket, ".quice-db.dat", minio.GetObjectOptions{})
	if err != nil {
		return err
	}
	buf := new(bytes.Buffer)
	buf.ReadFrom(object)
	dec := gob.NewDecoder(buf)
	err = dec.Decode(&m)

	if m.Users == nil {
		log.Debug("Creating empty user list")
		v := make(map[string]*User)
		m.Users = v
	}

	for username, user := range m.conf.Users {
		log.WithFields(log.Fields{
			"username": username,
		}).Debug("found in config")
		if _, ok := m.Users[username]; !ok {
			log.WithFields(log.Fields{
				"username": username,
			}).Debug("user not yet present in db")
			var u User
			u.Playlists = make(map[string]*Playlist)
			m.Users[username] = &u
		}
		for playlistName, play := range user.Playlists {
			if p, ok := m.Users[username].Playlists[playlistName]; ok {
				log.WithFields(log.Fields{
					"username": username,
					"playlist": playlistName,
				}).Debug("playlist with this name already present for user")
				p.Prefixes = play.Prefixes
				continue
			}

			var p Playlist
			p.Prefixes = play.Prefixes
			p.Sorttype = play.Sorttype
			p.Videos = make(map[string]*Video)
			m.Users[username].Playlists[playlistName] = &p
			log.WithFields(log.Fields{
				"username": username,
				"playlist": playlistName,
			}).Debug("added playlist for user")
		}
	}
	return nil
}

// UpdateProgress saves the progress in memory
func (m *Memdb) UpdateProgress(u string, p string, key string, pos int64) {
	m.lock.Lock()
	defer m.lock.Unlock()
	l := log.WithFields(log.Fields{
		"user":     u,
		"playlist": p,
		"key":      key,
	})
	user, ok := m.Users[u]
	if !ok {
		l.Warning("user not present")
		return
	}
	playlist, ok := user.Playlists[p]
	if !ok {
		l.Warning("playlist not present")
		return
	}
	video, ok := playlist.Videos[key]
	if !ok {
		l.Warning("video not not found")
		return
	}
	m.changes = true
	video.Position = pos
}

// GetPlaylistPosition returns the video-url that should be played next for a user u and playlist
func (m *Memdb) GetPlaylistPosition(u, p string) (string, string, int64) {
	m.lock.Lock()
	defer m.lock.Unlock()
	l := log.WithFields(log.Fields{
		"user":     u,
		"playlist": p,
	})
	user, ok := m.Users[u]
	if !ok {
		l.Warning("user not present")
		return "", "", 0
	}
	playlist, ok := user.Playlists[p]
	if !ok {
		l.Warning("playlist not present")
		return "", "", 0
	}
	for _, vidID := range playlist.sortedKeys {
		video, ok := playlist.Videos[vidID]
		if !ok {
			//weird edge case, but it could happen probably
			continue
		}
		if video.Completed {
			continue
		}
		l.WithField("video", video.Filename).Debug("is up next")
		url, err := m.store.PresignedGetObject(m.bucket, video.Key, 12*time.Hour, nil)
		if err != nil {
			l.WithField("error", err).Warning("could not get presigned url")
			return "", "", 0
		}
		return url.String(), vidID, video.Position
	}
	return "", "", 0
}

// SetCompleted marks a video completed for a video
func (m *Memdb) SetCompleted(u, p, key string) {
	m.lock.Lock()
	defer m.lock.Unlock()
	l := log.WithFields(log.Fields{
		"user":     u,
		"playlist": p,
		"key":      key,
	})
	user, ok := m.Users[u]
	if !ok {
		l.Warning("user not present")
		return
	}
	playlist, ok := user.Playlists[p]
	if !ok {
		l.Warning("playlist not present")
		return
	}
	video, ok := playlist.Videos[key]
	if !ok {
		l.Warning("video not not found")
		return
	}
	video.Completed = true
	m.changes = true
	return
}

func (m *Memdb) KeyExists(k string) bool {
	_, err := m.store.StatObject(m.bucket, k, minio.StatObjectOptions{})
	if err != nil {
		log.WithField("err", err).Warning("statobject failed")
	}
	return err == nil
}

func (m *Memdb) refresh() {
	m.lock.Lock()
	defer m.lock.Unlock()
	log.Info("Refreshing bucket contents")

	for username, user := range m.Users {
		for playlistname, play := range user.Playlists {
			l := log.WithFields(log.Fields{
				"user":     username,
				"playlist": playlistname,
			})
			for k, v := range play.Videos {
				if !m.KeyExists(v.Key) {
					l.WithField("key", v.Key).Info("file was removed")
					delete(play.Videos, k)
					m.changes = true
				}
			}
			for _, prefix := range play.Prefixes {
				l.WithField("prefix", prefix).Debug("adding files from prefix")
				files := m.listFiles(prefix)
				for _, o := range files {
					if !strings.HasSuffix(o.Key, ".mp4") {
						// doesn't make sense to process files that cannot be viewed
						continue
					}
					keyParts := strings.Split(o.Key, "/")
					var v Video
					var videoID string
					v.Completed = false
					v.Position = 0
					v.Filename = keyParts[len(keyParts)-1]
					v.Key = o.Key
					if play.Sorttype == "date" {
						videoID = o.LastModified.UTC().Format(time.RFC3339) + "_" + o.Key
					} else if play.Sorttype == "episode" {
						id, err := parseEpisode(v.Filename)
						if err != nil {
							l.WithField("filename", v.Filename).Debug("could not extract episode")
							continue
						}
						videoID = id + "_" + o.Key
					} else if play.Sorttype == "filename" {
						videoID = v.Filename
					}

					if _, ok := play.Videos[videoID]; !ok {
						play.Videos[videoID] = &v
						m.changes = true
					}
				}
			}
			play.sortedKeys = []string{}
			for videoID := range play.Videos {
				play.sortedKeys = append(play.sortedKeys, videoID)
			}
			sort.Sort(sort.StringSlice(play.sortedKeys))
		}
	}

}

func (m *Memdb) listFiles(prefix string) []minio.ObjectInfo {
	var matches []minio.ObjectInfo
	// Create a done channel to control 'ListObjectsV2' go routine.
	doneCh := make(chan struct{})

	// Indicate to our routine to exit cleanly upon return.
	defer close(doneCh)

	isRecursive := true
	objectCh := m.store.ListObjectsV2(m.bucket, prefix, isRecursive, doneCh)
	for object := range objectCh {
		if object.Err != nil {
			log.WithField("error", object.Err).Error("listing files failed")
			break
		}
		matches = append(matches, object)
	}
	return matches

}
func parseEpisode(title string) (string, error) {
	var matches []string

	matches = episodeRegexp.FindStringSubmatch(title)
	if matches != nil && len(matches) == 3 {
		season, err := strconv.ParseInt(matches[1], 10, 64)
		if err != nil {
			return "", errors.New("could not parse season")
		}
		episode, err := strconv.ParseInt(matches[2], 10, 64)
		if err != nil {
			return "", errors.New("could not parse episode")
		}
		return fmt.Sprintf("s%02de%02d", season, episode), nil
	}
	matches = episodeShortRegexp.FindStringSubmatch(title)
	if matches != nil && len(matches) == 3 {
		season, err := strconv.ParseInt(matches[1], 10, 64)
		if err != nil {
			return "", errors.New("could not parse season")
		}
		episode, err := strconv.ParseInt(matches[2], 10, 64)
		if err != nil {
			return "", errors.New("could not parse episode")
		}
		return fmt.Sprintf("s%02de%02d", season, episode), nil
	}
	return "", errors.New("Could not extract episode id")
}

type userResp struct {
	Users []string `json:"users"`
}

func (m *Memdb) GetUsers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m.lock.Lock()
		var u userResp
		for username := range m.Users {
			u.Users = append(u.Users, username)
		}
		m.lock.Unlock()
		sort.Sort(sort.StringSlice(u.Users))
		json.NewEncoder(w).Encode(u)
		return
	}
}

type playlistResp struct {
	Playlists []playResp `json:"playlists"`
}

type playResp struct {
	Name  string `json:"name"`
	Total int    `json:"count"`
	New   int64  `json:"new"`
}

func (m *Memdb) GetPlaylists() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m.lock.Lock()
		defer m.lock.Unlock()
		var u playlistResp
		var p playResp
		vars := mux.Vars(r)
		if _, ok := m.Users[vars["user"]]; !ok {
			//TODO: add error
			return
		}
		for pl, list := range m.Users[vars["user"]].Playlists {
			p.Name = pl
			p.Total = len(list.Videos)
			p.New = 0
			for _, v := range list.Videos {
				if !v.Completed {
					p.New += 1
				}
			}
			u.Playlists = append(u.Playlists, p)
		}
		json.NewEncoder(w).Encode(u)
		return
	}
}

type currentResp struct {
	Url       string   `json:"url"`
	Key       string   `json:"key"`
	Pos       int64    `json:"pos"`
	AllVideos []string `json:"all"`
}

func (m *Memdb) GetCurrentVideo() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var u currentResp
		vars := mux.Vars(r)
		url, key, pos := m.GetPlaylistPosition(vars["user"], vars["playlist"])
		if url != "" {
			u.AllVideos = m.Users[vars["user"]].Playlists[vars["playlist"]].sortedKeys
		}
		u.Url = url
		u.Key = key
		u.Pos = pos
		json.NewEncoder(w).Encode(u)
	}
}

type currentVideo struct {
	User     string `json:"user"`
	Key      string `json:"key"`
	Playlist string `json:"playlist"`
	Position int64  `json:"position"`
}

func (m *Memdb) SetCurrentVideo() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var c currentVideo
		b, _ := ioutil.ReadAll(r.Body)
		json.Unmarshal(b, &c)
		m.UpdateProgress(c.User, c.Playlist, c.Key, c.Position)
		fmt.Fprintf(w, "%q", "ok")
		return
	}
}

func (m *Memdb) CompleteVideo() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var c currentVideo
		b, _ := ioutil.ReadAll(r.Body)
		json.Unmarshal(b, &c)
		m.SetCompleted(c.User, c.Playlist, c.Key)
		fmt.Fprintf(w, "%q", "ok")
		return
	}
}
