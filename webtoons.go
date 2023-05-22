package webtoons

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"mvdan.cc/xurls/v2"

	"github.com/veants0/webtoons-veant/internal/helpers"
	"github.com/veants0/webtoons-veant/mail"
	"github.com/tidwall/gjson"
)

var (
	getKeysEndpoint  = "https://global.apis.naver.com/lineWebtoon/webtoon/getRsaKey.json?language=en&locale=en&platform=APP_IPHONE&serviceZone=GLOBAL"
	registerEndpoint = "https://global.apis.naver.com/lineWebtoon/webtoon/joinById.json"
	loginEndpoint    = "https://global.apis.naver.com/lineWebtoon/webtoon/loginById.json"
	readEndpoint     = "https://global.apis.naver.com/lineWebtoon/webtoon/eventReadLog.json?episodeNo=%d&language=en&locale=en&platform=APP_IPHONE&serviceZone=GLOBAL&titleNo=5291&v=2&webtoonType=WEBTOON"
	codeEndpoint     = "https://m.webtoons.com/app/promotion/saveCompleteInfo?promotionName=en_discord_phase1_202305&memo=%s"

	promoFind = regexp.MustCompile(`https://promos\.discord\.gg/[A-Za-z0-9]{0,24}`)
)

// Creator is a new instance used to create an account
type Creator struct {
	keyRing *KeyRing
	client  *http.Client

	mailer mail.Mailer
	info   accountInfo
}

type accountInfo struct {
	username string
	email    string
	password string
	token    string
}

func NewCreator(proxy string, mailer mail.Mailer) (*Creator, error) {

	jar, _ := cookiejar.New(nil)

	client := &http.Client{
		Transport: &http.Transport{ForceAttemptHTTP2: true},
		Timeout:   10 * time.Second,
		Jar:       jar,
	}

	return &Creator{
		keyRing: nil,
		client:  client,
		mailer:  mailer,
	}, nil
}

func (c *Creator) Create(mail, username string) error {
	c.info = accountInfo{
		username: username,
		email:    mail,
		password: helpers.RandString(8) + "1*",
	}

	if err := c.getKeys(); err != nil {
		return fmt.Errorf("create: %w", err)
	}

	if err := c.registerAccount(); err != nil {
		return fmt.Errorf("create: %w", err)
	}

	if err := c.verifyEmail(); err != nil {
		return fmt.Errorf("create: %w", err)
	}

	log.Printf("[*] Registered (%s)\n", c.info.email)

	if err := c.doLogin(); err != nil {
		return fmt.Errorf("create: %w", err)
	}

	return nil

}

var find = xurls.Strict()

func (c *Creator) verifyEmail() error {
	content, err := c.mailer.GetContent(c.info.email)
	if err != nil {
		return fmt.Errorf("verify: %w", err)
	}

	matches := find.FindAllString(content, -1)
	if len(matches) < 1 {
		return fmt.Errorf("verify: wrong matches len: (%v)", matches)
	}

	link := matches[len(matches)-1]

	req, err := http.NewRequest(http.MethodGet, link, nil)
	if err != nil {
		return fmt.Errorf("verify: new request: %w", err)
	}

	req.Header = http.Header{
		"sec-ch-ua":                   {`"Google Chrome";v="113", "Chromium";v="113", "Not-A.Brand";v="24"`},
		"sec-ch-ua-mobile":            {`?0`},
		"sec-ch-ua-platform":          {`"Windows"`},
		"sec-ch-ua-platform-version":  {`"10.0.0"`},
		"sec-ch-ua-model":             {`""`},
		"sec-ch-ua-full-version-list": {`"Google Chrome";v="113.0.5672.92", "Chromium";v="113.0.5672.92", "Not-A.Brand";v="24.0.0.0"`},
		"upgrade-insecure-requests":   {`1`},
		"user-agent":                  {`Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/113.0.0.0 Safari/537.36`},
		"accept":                      {`text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7`},
		"sec-fetch-site":              {`same-origin`},
		"sec-fetch-mode":              {`navigate`},
		"sec-fetch-dest":              {`document`},
		"accept-language":             {`en-US,en;q=0.9`},
	}

	res, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("verify: do: %w", err)
	}
	defer res.Body.Close()

	return nil
}
func (c *Creator) registerAccount() error {
	pw, err := c.keyRing.EncryptData(c.info.email, c.info.password)
	if err != nil {
		return fmt.Errorf("registerAccount: enc: %w", err)
	}

	form := fmt.Sprintf(
		"ageGateJoin=true&countryCode=GB&dayOfMonth=25&emailEventAlarm=true&encnm=%s&encpw=%s&language=en&locale=en&loginType=EMAIL&month=5&nickname=%s&platform=APP_IPHONE&serviceZone=GLOBAL&year=2000&zoneId=Europe/London", c.keyRing.KeyName, pw, c.info.username,
	)

	req, err := http.NewRequest(http.MethodPost, SignRequest(registerEndpoint), strings.NewReader(form))
	if err != nil {
		return fmt.Errorf("registerAccount: new request: %w", err)
	}

	req.Header = http.Header{
		"content-type":    {"application/x-www-form-urlencoded; charset=utf-8"},
		"accept":          {`*/*`},
		"user-agent":      {`linewebtoon/2.12.4 (iPhone; iOS 15.6.1; Scale/2.00)`},
		"accept-language": {``},
		"referer":         {`https://m.webtoons.com/`},
	}

	res, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("registerAccount: do: %w", err)
	}
	defer res.Body.Close()

	//body, err := io.ReadAll(res.Body)
	//if err != nil {
	//	return fmt.Errorf("registerAccount: read: %w", err)
	//}

	if res.StatusCode != http.StatusOK {
		//fmt.Println(string(body))
		return fmt.Errorf("registerAccount: bad status code: %s", res.Status)
	}

	return nil
}
func (c *Creator) getKeys() error {
	req, err := http.NewRequest(http.MethodGet, SignRequest(getKeysEndpoint), nil)
	if err != nil {
		return fmt.Errorf("getKeys: new request: %w", err)
	}

	req.Header = http.Header{
		"accept":          {`*/*`},
		"user-agent":      {`linewebtoon/2.12.4 (iPhone; iOS 15.6.1; Scale/2.00)`},
		"accept-language": {`en-US,en;q=0.9`},
		"referer":         {`https://m.webtoons.com/`},
	}

	res, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("getKeys: do: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("getKeys: read: %w", err)
	}

	parsed := gjson.ParseBytes(body)
	result := parsed.Get("message.result")
	if !result.Exists() {
		return fmt.Errorf("getKeys: bad body")
	}

	c.keyRing = &KeyRing{
		SessionKey: result.Get("sessionKey").String(),
		Modulus:    result.Get("evalue").String(),
		Exponent:   result.Get("nvalue").String(),
		KeyName:    result.Get("keyName").String(),
	}

	return nil
}
func (c *Creator) doLogin() error {
	pw, err := c.keyRing.EncryptData(c.info.email, c.info.password)
	if err != nil {
		return fmt.Errorf("doLogin: enc: %w", err)
	}

	form := fmt.Sprintf(
		"encnm=%s&encpw=%s&language=en&locale=en&loginType=EMAIL&platform=APP_IPHONE&serviceZone=GLOBAL&v=2", c.keyRing.KeyName, pw,
	)

	req, err := http.NewRequest(http.MethodPost, SignRequest(loginEndpoint), strings.NewReader(form))
	if err != nil {
		return fmt.Errorf("doLogin: new request: %w", err)
	}

	req.Header = http.Header{
		"content-type":    {"application/x-www-form-urlencoded; charset=utf-8"},
		"accept":          {`*/*`},
		"user-agent":      {`linewebtoon/2.12.4 (iPhone; iOS 15.6.1; Scale/2.00)`},
		"accept-language": {``},
		"referer":         {`https://m.webtoons.com/`},
	}

	res, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("doLogin: do: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("doLogin: read: %w", err)
	}

	parsed := gjson.ParseBytes(body)

	result := parsed.Get("message.result")
	if !result.Exists() {
		return fmt.Errorf("doLogin: wrong body")
	}
	c.info.token = result.Get("ses").String()
	return nil
}
func (c *Creator) readAll() error {
	slave := func(num int) error {
		req, err := http.NewRequest(http.MethodGet, SignRequest(fmt.Sprintf(readEndpoint, num)), nil)
		if err != nil {
			return fmt.Errorf("new request: %w", err)
		}

		req.Header = http.Header{
			"accept":          {`*/*`},
			"user-agent":      {`linewebtoon/2.12.4 (iPhone; iOS 15.6.1; Scale/2.00)`},
			"accept-language": {``},
			"referer":         {`https://m.webtoons.com/`},
			"cookie":          {"NEO_SES=" + strconv.Quote(c.info.token)},
		}

		res, err := c.client.Do(req)
		if err != nil {
			return fmt.Errorf("do: %w", err)
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			return fmt.Errorf("bad status: %d", res.StatusCode)
		}

		return nil
	}

	for i := 1; i < 5; i++ {
		if err := slave(i); err != nil {
			return fmt.Errorf("slave: %w", err)
		}
	}

	return nil
}
func (c *Creator) RedeemCode() (string, error) {

	if err := c.readAll(); err != nil {
		return "", fmt.Errorf("RedeemCode: read chapters: %w", err)
	}

	log.Printf("[*] Redeeming code (%s)\n", c.info.email)

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf(codeEndpoint, url.QueryEscape(c.info.email)), nil)
	if err != nil {
		return "", fmt.Errorf("RedeemCode: new request: %w", err)
	}

	req.Header = http.Header{
		"accept":          {`application/json, text/plain, */*`},
		"user-agent":      {`Mozilla/5.0 (iPhone; CPU iPhone OS 15_6_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 linewebtoon/2.12.4 (iPhone; iOS 15.6.1; Scale/2.00)`},
		"accept-language": {`en-US,en;q=0.9`},
		"referer":         {`https://m.webtoons.com/app/promotion/read/en_discord_phase1_202305/progress?platform=APP_IPHONE`},
		"cookie":          {"NEO_SES=" + strconv.Quote(c.info.token)},
	}

	res, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("RedeemCode: do: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("RedeemCode: read: %w", err)
	}

	if string(body) != "true" {
		return "", fmt.Errorf("RedeemCode: wrong body (%s)", string(body))
	}

	counter := 0
	for {
		if counter >= 15 {
			return "", fmt.Errorf("RedeemCode: timeout receiving promo code")
		}

		time.Sleep(2 * time.Minute)

		content, err := c.mailer.GetContent(c.info.email)
		if err != nil {
			// We can ignore this error because at a certain point the tidal
			// api will delete mails
			if errors.Is(err, mail.ErrNotFound) {
				counter++
				continue
			}
			return "", fmt.Errorf("RedeemCode: mail: %w", err)
		}

		if !strings.Contains(content, "promos.discord.gg") {
			counter++
			continue
		}

		return promoFind.FindString(content), nil
	}
}
