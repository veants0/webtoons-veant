package mail

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/d4c5d1e0/webtoons/internal/helpers"
)

var (
	emailEndpoint = "https://api.tidal.lol/api/v1/emails/%s"
	maxCount      = 20
)

var _ Mailer = (*Tidal)(nil)

type tidalApiResponse struct {
	Code   int           `json:"code"`
	Emails []tidalEmails `json:"emails"`
}

type tidalEmails struct {
	UniqueID string `json:"unique_id"`
	To       string `json:"to"`
	From     string `json:"from"`
	Subject  string `json:"subject"`
	Date     string `json:"date"`
	Body     struct {
		Text string `json:"text"`
		HTML string `json:"html"`
	} `json:"body"`
}

type Tidal struct {
	domain string
	client *http.Client
}

// NewTidalMailer return a Mailer using tidal.lol temp mail api
// please consider supporting the creator financially: https://t.me/modules
func NewTidalMailer(domain string) Mailer {
	t := &Tidal{
		domain: domain,
	}

	t.client = &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:    1024,
			MaxConnsPerHost: 100,
			IdleConnTimeout: 10 * time.Second,
		},
		Timeout: 10 * time.Second,
	}

	return t
}

func (t *Tidal) GetContent(address string) (string, error) {
	slave := func() (*tidalApiResponse, error) {
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf(emailEndpoint, address), nil)
		if err != nil {
			return nil, fmt.Errorf("new request: %w", err)
		}

		res, err := t.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("do: %w", err)
		}
		defer res.Body.Close()

		var response tidalApiResponse

		decoder := json.NewDecoder(res.Body)

		if err := decoder.Decode(&response); err != nil {
			return nil, fmt.Errorf("json: %w", err)
		}

		return &response, nil
	}

	counter := 0
	for {
		if counter >= maxCount {
			return "", ErrNotFound
		}

		time.Sleep(1 * time.Second)
		result, err := slave()
		if err != nil {
			return "", fmt.Errorf("tidal: get: slave: %w", err)
		}
		if result.Code != 200 {
			counter++
			continue
		}

		return result.Emails[len(result.Emails)-1].Body.Text, nil
	}
}

func (t *Tidal) RandomAddress() string {
	return fmt.Sprintf("%s@%s", helpers.RandString(8), t.domain)
}