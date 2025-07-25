package common

import (
	"fmt"
	"sync"
	"time"

	"github.com/EgorGorshen/bitget/config"
	"github.com/EgorGorshen/bitget/constants"
	"github.com/EgorGorshen/bitget/internal"
	"github.com/EgorGorshen/bitget/logging/applogger"
	"github.com/EgorGorshen/bitget/pkg/model"
	"github.com/gorilla/websocket"
	"github.com/robfig/cron"
)

type BitgetBaseWsClient struct {
	NeedLogin        bool
	Connection       bool
	LoginStatus      bool
	Listener         OnReceive
	ErrorListener    OnReceive
	Ticker           *time.Ticker
	SendMutex        *sync.Mutex
	WebSocketClient  *websocket.Conn
	LastReceivedTime time.Time
	AllSuribe        *model.Set
	Signer           *Signer
	ScribeMap        map[model.SubscribeReq]OnReceive
}

func (p *BitgetBaseWsClient) Init() *BitgetBaseWsClient {
	p.Connection = false
	p.AllSuribe = model.NewSet()
	p.Signer = new(Signer).Init(config.Config.SecretKey)
	p.ScribeMap = make(map[model.SubscribeReq]OnReceive)
	p.SendMutex = &sync.Mutex{}
	p.Ticker = time.NewTicker(constants.TimerIntervalSecond * time.Second)
	p.LastReceivedTime = time.Now()

	return p
}

func (p *BitgetBaseWsClient) SetListener(msgListener OnReceive, errorListener OnReceive) {
	p.Listener = msgListener
	p.ErrorListener = errorListener
}

func (p *BitgetBaseWsClient) Connect() {

	p.tickerLoop()
	p.ExecuterPing()
}

func (p *BitgetBaseWsClient) ConnectWebSocket() {
	var err error
	applogger.Info("WebSocket connecting...")
	p.WebSocketClient, _, err = websocket.DefaultDialer.Dial(constants.WsUrl, nil)
	if err != nil {
		fmt.Printf("WebSocket connected error: %s\n", err)
		return
	}
	applogger.Info("WebSocket connected")
	p.Connection = true
}

func (p *BitgetBaseWsClient) Login() {
	timesStamp := internal.TimesStampSec()
	sign := p.Signer.Sign(constants.WsAuthMethod, constants.WsAuthPath, "", timesStamp)
	if constants.RSA == config.Config.SignType {
		sign = p.Signer.SignByRSA(constants.WsAuthMethod, constants.WsAuthPath, "", timesStamp)
	}

	loginReq := model.WsLoginReq{
		ApiKey:     config.Config.ApiKey,
		Passphrase: config.Config.PASSPHRASE,
		Timestamp:  timesStamp,
		Sign:       sign,
	}
	var args []interface{}
	args = append(args, loginReq)

	baseReq := model.WsBaseReq{
		Op:   constants.WsOpLogin,
		Args: args,
	}
	p.SendByType(baseReq)
}

func (p *BitgetBaseWsClient) StartReadLoop() {
	go p.ReadLoop()
}

func (p *BitgetBaseWsClient) ExecuterPing() {
	c := cron.New()
	_ = c.AddFunc("*/15 * * * * *", p.ping)
	c.Start()
}
func (p *BitgetBaseWsClient) ping() {
	p.Send("ping")
}

func (p *BitgetBaseWsClient) SendByType(req model.WsBaseReq) {
	json, _ := internal.ToJson(req)
	p.Send(json)
}

func (p *BitgetBaseWsClient) Send(data string) {
	if p.WebSocketClient == nil {
		applogger.Error("WebSocket sent error: no connection available")
		return
	}
	applogger.Info("sendMessage:%s", data)
	p.SendMutex.Lock()
	err := p.WebSocketClient.WriteMessage(websocket.TextMessage, []byte(data))
	p.SendMutex.Unlock()
	if err != nil {
		applogger.Error("WebSocket sent error: data=%s, error=%s", data, err)
	}
}

func (p *BitgetBaseWsClient) tickerLoop() {
	applogger.Info("tickerLoop started")
	for {
		select {
		case <-p.Ticker.C:
			elapsedSecond := time.Now().Sub(p.LastReceivedTime).Seconds()

			if elapsedSecond > constants.ReconnectWaitSecond {
				applogger.Info("WebSocket reconnect...")
				p.disconnectWebSocket()
				p.ConnectWebSocket()
			}
		}
	}
}

func (p *BitgetBaseWsClient) disconnectWebSocket() {
	if p.WebSocketClient == nil {
		return
	}

	fmt.Println("WebSocket disconnecting...")
	err := p.WebSocketClient.Close()
	if err != nil {
		applogger.Error("WebSocket disconnect error: %s\n", err)
		return
	}

	applogger.Info("WebSocket disconnected")
}

func (p *BitgetBaseWsClient) ReadLoop() {
	for {

		if p.WebSocketClient == nil {
			applogger.Info("Read error: no connection available")
			//time.Sleep(TimerIntervalSecond * time.Second)
			continue
		}

		_, buf, err := p.WebSocketClient.ReadMessage()
		if err != nil {
			applogger.Info("Read error: %s", err)
			continue
		}
		p.LastReceivedTime = time.Now()
		message := string(buf)

		applogger.Info("rev:" + message)

		if message == "pong" {
			applogger.Info("Keep connected:" + message)
			continue
		}
		jsonMap := internal.JSONToMap(message)

		v, e := jsonMap["code"]

		if e && int(v.(float64)) != 0 {
			p.ErrorListener(message)
			continue
		}

		v, e = jsonMap["event"]
		if e && v == "login" {
			applogger.Info("login msg:" + message)
			p.LoginStatus = true
			continue
		}

		v, e = jsonMap["data"]
		if e {
			listener := p.GetListener(jsonMap["arg"])
			listener(message)
			continue
		}
		p.handleMessage(message)
	}

}

func (p *BitgetBaseWsClient) GetListener(argJson interface{}) OnReceive {

	mapData := argJson.(map[string]interface{})

	subscribeReq := model.SubscribeReq{
		InstType: fmt.Sprintf("%v", mapData["instType"]),
		Channel:  fmt.Sprintf("%v", mapData["channel"]),
		InstId:   fmt.Sprintf("%v", mapData["instId"]),
	}

	v, e := p.ScribeMap[subscribeReq]

	if !e {
		return p.Listener
	}
	return v
}

type OnReceive func(message string)

func (p *BitgetBaseWsClient) handleMessage(msg string) {
	fmt.Println("default:" + msg)
}
