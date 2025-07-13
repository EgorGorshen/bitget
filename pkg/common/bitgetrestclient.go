package common

import (
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/EgorGorshen/bitget/config"
	"github.com/EgorGorshen/bitget/constants"
	"github.com/EgorGorshen/bitget/internal"
)

type BitgetRestClient struct {
	ApiKey       string
	ApiSecretKey string
	Passphrase   string
	BaseUrl      string
	HttpClient   http.Client
	Signer       *Signer
}

func (p *BitgetRestClient) Init() *BitgetRestClient {
	p.ApiKey = config.Config.ApiKey
	p.ApiSecretKey = config.Config.SecretKey
	p.BaseUrl = constants.BaseUrl
	p.Passphrase = config.Config.PASSPHRASE
	p.Signer = new(Signer).Init(config.Config.SecretKey)
	p.HttpClient = http.Client{
		Timeout: time.Duration(config.Config.TimeoutMilisecond) * time.Millisecond,
	}
	return p
}

func (p *BitgetRestClient) DoPost(uri string, params string) (string, error) {
	timesStamp := internal.TimesStamp()
	//body, _ := internal.BuildJsonParams(params)

	sign := p.Signer.Sign(constants.POST, uri, params, timesStamp)
	if constants.RSA == config.Config.SignType {
		sign = p.Signer.SignByRSA(constants.POST, uri, params, timesStamp)
	}
	requestUrl := constants.BaseUrl + uri

	buffer := strings.NewReader(params)
	request, err := http.NewRequest(constants.POST, requestUrl, buffer)

	internal.Headers(request, p.ApiKey, timesStamp, sign, p.Passphrase)
	if err != nil {
		return "", err
	}
	response, err := p.HttpClient.Do(request)

	if err != nil {
		return "", err
	}

	defer response.Body.Close()

	bodyStr, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	responseBodyString := string(bodyStr)
	return responseBodyString, err
}

func (p *BitgetRestClient) DoGet(uri string, params map[string]string) (string, error) {
	timesStamp := internal.TimesStamp()
	body := internal.BuildGetParams(params)
	//fmt.Println(body)

	sign := p.Signer.Sign(constants.GET, uri, body, timesStamp)

	requestUrl := p.BaseUrl + uri + body

	request, err := http.NewRequest(constants.GET, requestUrl, nil)
	if err != nil {
		return "", err
	}
	internal.Headers(request, p.ApiKey, timesStamp, sign, p.Passphrase)

	response, err := p.HttpClient.Do(request)

	if err != nil {
		return "", err
	}

	defer response.Body.Close()

	bodyStr, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	responseBodyString := string(bodyStr)
	return responseBodyString, err
}
