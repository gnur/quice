package config

import (
	"bytes"
	"time"

	"github.com/burntsushi/toml"
	"github.com/minio/minio-go"
	log "github.com/sirupsen/logrus"
)

// Cfg holds the config for quice
type Cfg struct {
	Users map[string]User
}

// User is a user that has it's own playlists
type User struct {
	Playlists map[string]Playlist
}

// Playlist is the struct that defines a playlist
type Playlist struct {
	Prefixes []string
	Sorttype string
	MaxAge   duration
}

type duration struct {
	time.Duration
}

func (d *duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}

// Load loads the config from an S3 bucket and returns a pointer to the configuration
func Load(mc *minio.Client, bucket string) (*Cfg, error) {
	object, err := mc.GetObject(bucket, "quice.toml", minio.GetObjectOptions{})
	if err != nil {
		log.Fatal("Could not read config file")
		return nil, err
	}
	buf := new(bytes.Buffer)
	n, err := buf.ReadFrom(object)
	if err != nil {
		return nil, err
	}
	log.WithField("size", n).Debug("reading quice.toml")
	f := buf.Bytes()

	var v Cfg
	_, err = toml.Decode(string(f), &v)
	if err != nil {
		return nil, err
	}

	return &v, nil

}
