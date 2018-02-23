package main

import (
	"flag"
	"fmt"
	"github.com/caarlos0/env"
	"github.com/minio/minio-go"
	"github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	tb "gopkg.in/tucnak/telebot.v2"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

type config struct {
	TelegramToken  string `env:"TELEGRAM_TOKEN"`
	TelegramUserID string `env:"TELEGRAM_USERID"`
	MinioURL       string `env:"MINIO_URL"`
	MinioAccessKey string `env:"MINIO_ACCESS_KEY"`
	MinioSecretKey string `env:"MINIO_SECRET_KEY"`
}

var (
	versionflag bool
	version     string
)

func init() {
	flag.BoolVar(&versionflag, "v", false, "Print build id")
	flag.Parse()
}

func main() {
	if versionflag {
		fmt.Printf("build : %s\n", version)
		os.Exit(0)
	}

	c := config{}
	err := env.Parse(&c)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Fatal("Can't parse config...")
	}
	b, err := tb.NewBot(tb.Settings{
		Token:  c.TelegramToken,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Fatal("Can't start bot. Check your config for telegram.")
	}
	b.Handle(tb.OnPhoto, func(m *tb.Message) {
		userid, err := strconv.Atoi(c.TelegramUserID)
		if err != nil {
			log.WithFields(log.Fields{"err": err}).Fatal("TELEGRAM_USERID is not int")
		}
		if m.Sender.ID == userid {
			if m.Photo.File.InCloud() {
				fileurl, err := b.FileURLByID(m.Photo.File.FileID)
				if err != nil {
					log.WithFields(log.Fields{"iduser": m.Sender.ID}).Fatal("Can't find file from telegram")
					return
				}
				r, err := http.Get(fileurl)
				if err != nil {
					log.WithFields(log.Fields{"iduser": m.Sender.ID, "err": err}).Fatal("Can't download file from telegram")
					b.Send(m.Sender, "Internal error :(")
				}
				s3Client, err := minio.New(c.MinioURL, c.MinioAccessKey, c.MinioSecretKey, true)

				if err != nil {
					log.WithFields(log.Fields{"iduser": m.Sender.ID, "err": err}).Fatal("Can't connect to S3")
					b.Send(m.Sender, "Internal error :(")
				}
				// or error handling
				uuid, err := uuid.NewV4()
				if err != nil {
					log.WithFields(log.Fields{"iduser": m.Sender.ID, "err": err}).Fatal("Something went wrong when you generate the uuid")
					b.Send(m.Sender, "Internal error :(")
				}
				namefile := uuid.String() + ".jpg"
				n, err := s3Client.PutObject("telegram", namefile, r.Body, r.ContentLength, minio.PutObjectOptions{ContentType: "application/octet-stream"})
				if err != nil {
					log.WithFields(log.Fields{"iduser": m.Sender.ID, "err": err}).Fatal("It's not possible to upload file to S3")
					b.Send(m.Sender, "Internal error :(")
				}
				log.WithFields(log.Fields{"iduser": m.Sender.ID, "err": err, "s3": n}).Debug("Uploaded Successfully")
				reqParams := make(url.Values)

				presignedURL, err := s3Client.PresignedGetObject("telegram", namefile, time.Second*24*60*60, reqParams)
				if err != nil {
					log.WithFields(log.Fields{"iduser": m.Sender.ID, "err": err}).Fatal("Can't Presigned url")
				}
				log.WithFields(log.Fields{"iduser": m.Sender.ID, "url": presignedURL.String()}).Debug("Successfully generated presigned URL")
				b.Send(m.Sender, presignedURL.String())
			} else {
				log.WithFields(log.Fields{"iduser": m.Sender.ID}).Warn("Error telegram upload")
				b.Send(m.Sender, "Error telegram upload.")
			}

		} else {
			log.WithFields(log.Fields{"iduser": m.Sender.ID}).Warn("Unauthorized user")
			b.Send(m.Sender, "Unauthorized user")
		}
	})
	b.Start()
}
