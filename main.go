package main

import (
	"net/http"
	"os"
	"time"

	"github.com/gnur/quice/config"
	"github.com/gnur/quice/memdb"
	"github.com/gorilla/mux"
	"github.com/minio/minio-go"
	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
}

func main() {
	host := os.Getenv("S3_HOST")
	accessKeyID := os.Getenv("S3_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("S3_SECRET_ACCESS_KEY")
	bucket := os.Getenv("S3_BUCKET")

	minioClient, err := minio.New(host, accessKeyID, secretAccessKey, true)
	if err != nil {
		log.WithField("err", err).Fatal("Could not create client")
		return
	}
	log.Debug("created client")

	config, err := config.Load(minioClient, bucket)
	if err != nil {
		log.WithField("err", err).Fatal("Could not load config")
		return
	}
	log.Info("loaded playlists")

	db := memdb.Init(minioClient, bucket, config, time.Minute, time.Minute*5)
	if os.Getenv("tests") == "tests" {
		_, key, pos := db.GetPlaylistPosition("erwin", "comedy")
		log.WithFields(log.Fields{
			"key": key,
			"pos": pos,
		}).Info("is next up for playing")
		_, key, pos = db.GetPlaylistPosition("erwin", "comedy")
		db.UpdateProgress("erwin is ok", "comedy", key, 15)
		_, key, pos = db.GetPlaylistPosition("erwin", "comedy")
		log.WithFields(log.Fields{
			"key": key,
			"pos": pos,
		}).Info("is next up for playing")
		return
	}

	r := mux.NewRouter()

	r.HandleFunc("/api/users", db.GetUsers())
	r.HandleFunc("/api/playlists/{user}/", db.GetPlaylists())
	r.HandleFunc("/api/current/{user}/{playlist}/", db.GetCurrentVideo()).Methods("GET")
	r.HandleFunc("/api/updatecurrent/", db.SetCurrentVideo()).Methods("POST")

	r.PathPrefix("/").Handler(http.FileServer(assetFS()))

	http.Handle("/", r)

	log.Fatal(http.ListenAndServe("localhost:8624", nil))

}
