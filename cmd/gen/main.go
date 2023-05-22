package main

import (
	"io"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/veants0/webtoons-veant"
	"github.com/veants0/webtoons-veant/internal/helpers"
	"github.com/veants0/webtoons-veant/mail"
)

const (
	ThreadNumber = 25
	// Change the proxy
	Proxy = "http://127.0.0.1:8888"
)

var promos = func() *os.File {
	file, err := os.OpenFile("promos.txt", os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	return file
}()

func main() {
	defer promos.Close()
	mailers := []mail.Mailer{
		mail.NewTidalMailer("ecriminal.online"),
	}

	for i := 0; i < ThreadNumber; i++ {
		time.Sleep(25 * time.Millisecond)

		go func() {
			for {
				mailer := mailers[rand.Intn(len(mailers))]
				creator, err := webtoons.NewCreator(Proxy, mailer)
				if err != nil {
					log.Printf("[x] ERROR %v\n", err)
					continue
				}

				err = creator.Create(mailer.RandomAddress(), helpers.RandString(8))
				if err != nil {
					log.Printf("[x] ERROR %v\n", err)
					continue
				}

				code, err := creator.RedeemCode()
				if err != nil {
					log.Printf("[x] ERROR %v\n", err)
					continue
				}

				log.Printf("[*] Got code (%s)\n", code)

				io.WriteString(promos, code+"\n")
			}
		}()
	}

	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, os.Interrupt)
	<-s

}
