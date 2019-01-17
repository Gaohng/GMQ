package controllers

import (
	"golang.org/x/net/websocket"
	"sync"
	"github.com/labstack/echo"
	"github.com/fpay/erp-client-s/modules/webModule/models"
	"github.com/fpay/erp-client-s/modules/webModule/common"
	context2 "golang.org/x/net/context"
	"encoding/json"
	"github.com/fpay/erp-client-s/interfaces"
)

type Response struct {
	Code int
	Msg  string
	Data interface{}
}

type IndexController struct {
	WebSocketClientMap map[string]*models.WsClientModel
	Mutex              sync.Mutex
	webModule          interfaces.Module
}

func NewIndexController(module interfaces.Module) *IndexController {
	temp := new(IndexController)
	temp.webModule = module
	temp.WebSocketClientMap = make(map[string]*models.WsClientModel, 10)
	return temp
}

func (c *IndexController) Index(ctx echo.Context) error {
	content, err := json.Marshal(c.WebSocketClientMap)
	if err != nil {
		ctx.HTML(503, err.Error())
		return err
	}
	ctx.HTML(200, string(content))
	return nil
}

func (c *IndexController) Message(ctx echo.Context) error {
	moduleName := ctx.QueryParam("moduleName")
	c.webModule.Info("New Message coming! moduleName:" + moduleName)
	if ctx.IsWebSocket() {
		websocket.Handler(func(ws *websocket.Conn) {
			c.webModule.Debug("ws handel ->")
			client, ok := c.WebSocketClientMap[moduleName]
			if ok {
				stopEvent := common.NewEvent("stop", "新的同名模块连接到来")
				client.SendMsg(stopEvent)
				client.Stop()
			}
			context := context2.Background()
			client = models.NewWsClientModel(ws, context, c.webModule)
			c.Mutex.Lock()
			c.WebSocketClientMap[moduleName] = client
			c.Mutex.Unlock()
			c.webModule.Debug("ws handel ->2")
			client.Run()
		}).ServeHTTP(ctx.Response(), ctx.Request())
	} else {
		c.webModule.Info("Message: 非法请求" )
		response := &Response{
			Code: 1,
			Msg:  "非法请求",
		}
		ctx.JSON(500, response)
	}
	return nil
}

func (c *IndexController) SendMessage(msg interfaces.Event) {
	for _, client := range c.WebSocketClientMap {
		if client == nil {
			continue
		}
		err := client.SendMsg(msg)
		if err != nil {
			c.webModule.Warning("snedMsg error " + err.Error())
		}
	}
}
