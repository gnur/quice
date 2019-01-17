package main

import (
	"net/http"
	"os"
	"time"

	"github.com/gnur/quice/config"
	"github.com/gnur/quice/memdb"
	"github.com/gnur/quice/static"
	"github.com/gorilla/mux"
	minio "github.com/minio/minio-go"
	log "github.com/sirupsen/logrus"
)

//go:generate fileb0x fileb0x.toml

func init() {
	log.SetOutput(os.Stdout)
	if os.Getenv("LOGLEVEL") == "DEBUG" {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
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

	db := memdb.Init(minioClient, bucket, config, time.Minute, time.Minute*10)

	r := mux.NewRouter()

	indexBytes, err := static.ReadFile("index.html")
	if err != nil {
		log.WithField("err", err).Fatal("could not read index.html")
	}

	r.HandleFunc("/api/users", db.GetUsers())
	r.HandleFunc("/api/playlists/{user}/", db.GetPlaylists())
	r.HandleFunc("/api/current/{user}/{playlist}/", db.GetCurrentVideo()).Methods("GET")
	r.HandleFunc("/api/updatecurrent/", db.SetCurrentVideo()).Methods("POST")
	r.HandleFunc("/api/setcompleted/", db.CompleteVideo()).Methods("POST")
	r.HandleFunc("/api/togglecompleted/", db.ToggleVideo()).Methods("POST")
	//r.HandleFunc("/api/refresh/", db.Refresh()).Methods("POST")
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write(indexBytes)
	}).Methods("GET")

	r.PathPrefix("/").Handler(static.Handler)

	http.Handle("/", r)

	if os.Getenv("BIND_ADDR") == "" {
		log.Fatal(http.ListenAndServe(":8624", nil))
	} else {
		log.Fatal(http.ListenAndServe(os.Getenv("BIND_ADDR"), nil))
	}
}
