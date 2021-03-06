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
	minio "github.com/minio/minio-go"
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
	} else {
		//delete users that are in memdb but no longer in config file
		for u := range m.Users {
			if _, ok := m.conf.Users[u]; !ok {
				log.WithFields(log.Fields{
					"username": u,
				}).Debug("user no longer present in config file, deleting")
				delete(m.Users, u)
			}
		}
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
		//delete playlists that are in memdb but no longer in config file
		for p := range m.Users[username].Playlists {
			if _, ok := m.conf.Users[username].Playlists[p]; !ok {
				log.WithFields(log.Fields{
					"username": username,
					"playlist": p,
				}).Debug("playlist no longer present in config file, deleting")
				delete(m.Users[username].Playlists, p)
			}
		}
		for playlistName, play := range user.Playlists {
			if p, ok := m.Users[username].Playlists[playlistName]; ok {
				log.WithFields(log.Fields{
					"username": username,
					"playlist": playlistName,
				}).Debug("playlist with this name already present for user")
				p.Prefixes = play.Prefixes
				if play.MaxAge.Duration > time.Hour {
					p.MaxAge = play.MaxAge.Duration
				} else {
					p.MaxAge = time.Hour * 24 * 365 * 50 // 50 years ought to be enough for everybody
				}
				continue
			}

			var p Playlist
			p.Prefixes = play.Prefixes
			p.Sorttype = play.Sorttype
			p.Videos = make(map[string]*Video)
			if play.MaxAge.Duration > time.Hour {
				p.MaxAge = play.MaxAge.Duration
			} else {
				p.MaxAge = time.Hour * 24 * 365 * 50 // 50 years ought to be enough for everybody
			}
			m.Users[username].Playlists[playlistName] = &p
			log.WithFields(log.Fields{
				"username": username,
				"maxage":   p.MaxAge.String(),
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

// KeyExists returns true or false depending on the existence of an object in an S3 bucket
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
					v.Changed = o.LastModified
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
			for videoID, vid := range play.Videos {
				play.sortedKeys = append(play.sortedKeys, videoID)
				cutOffPoint := time.Now().Add(-play.MaxAge)
				if vid.Changed.Before(cutOffPoint) && !vid.Completed {
					vid.Completed = true
					m.changes = true
					log.WithFields(log.Fields{
						"cutoff":  cutOffPoint,
						"video":   vid.Filename,
						"changed": vid.Changed,
					}).Info("Marking videos as watched because it's old")
				} else {
					log.WithFields(log.Fields{
						"cutoff":  cutOffPoint,
						"video":   vid.Filename,
						"changed": vid.Changed,
					}).Debug("Video can stay, is not too old")
				}
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

// GetUsers returns all users as JSON
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

// GetPlaylists returns all playlists for an user as JSON
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
					p.New++
				}
			}
			u.Playlists = append(u.Playlists, p)
		}
		sort.Slice(u.Playlists, func(i, j int) bool {
			return u.Playlists[i].Name < u.Playlists[j].Name
		})
		json.NewEncoder(w).Encode(u)
		return
	}
}

type currentResp struct {
	URL        string            `json:"url"`
	Key        string            `json:"key"`
	Pos        int64             `json:"pos"`
	SortedKeys []string          `json:"sortedKeys"`
	Videos     map[string]*Video `json:"videos"`
	Completed  bool              `json:"completed"`
}

// GetCurrentVideo returns the first unwatched video from a playlist
func (m *Memdb) GetCurrentVideo() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var u currentResp
		vars := mux.Vars(r)
		url, key, pos := m.GetPlaylistPosition(vars["user"], vars["playlist"])
		if url == "" {
			u.Completed = true
		}
		u.SortedKeys = m.Users[vars["user"]].Playlists[vars["playlist"]].sortedKeys
		u.URL = url
		u.Key = key
		u.Pos = pos
		u.Videos = m.Users[vars["user"]].Playlists[vars["playlist"]].Videos
		json.NewEncoder(w).Encode(u)
	}
}

type currentVideo struct {
	User     string `json:"user"`
	Key      string `json:"key"`
	Playlist string `json:"playlist"`
	Position int64  `json:"position"`
}

// SetCurrentVideo handles the HTTP request to update the memDB with current position of a video
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

// CompleteVideo handles the HTTP request that marks a video as completed
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

// ToggleVideo handles the HTTP request that toggles a videos completed status
func (m *Memdb) ToggleVideo() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var c currentVideo
		b, _ := ioutil.ReadAll(r.Body)
		json.Unmarshal(b, &c)
		m.ToggleCompleted(c.User, c.Playlist, c.Key)
		fmt.Fprintf(w, "%q", "ok")
		return
	}
}

// ToggleCompleted toggles a videos completed status
func (m *Memdb) ToggleCompleted(u, p, key string) {
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
	video.Completed = !video.Completed
	m.changes = true
	return
}
