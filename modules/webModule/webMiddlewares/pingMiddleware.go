package webMiddlewares

import (
	"github.com/labstack/echo"
	"github.com/gw123/GMQ/core/interfaces"
	"time"
	"strconv"
	"github.com/gw123/GMQ/modules/webModule/models"
	"fmt"
)

func NewPingMiddleware(app interfaces.App) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			req := c.Request()
			res := c.Response()
			start := time.Now()
			if err = next(c); err != nil {
				c.Error(err)
			}
			stop := time.Now()
			clientId := c.QueryParam("clientId")
			payload := c.QueryParam("payload")
			sendAt := c.QueryParam("sendAt")

			sendAtTIme, _ := time.Parse("2019-08-24", sendAt)
			if sendAtTIme.IsZero() {
				sendAtTIme = time.Now()
			}
			l := stop.Sub(start)
			if l != 0 {
				l = l / time.Millisecond
			}

			byteIn, _ := strconv.Atoi(req.Header.Get(echo.HeaderContentLength))
			pingLog := &models.PingLog{
				Ip:           c.RealIP(),
				ClientSendAt: sendAtTIme,
				CreatedAt:    time.Now(),
				Payload:      payload,
				ClientId:     clientId,
				Latency:      uint(l),
				BytesIn:      uint(byteIn),
				BytesOut:     uint(res.Size),
			}
			db, err := app.GetDefaultDb()
			if err != nil {
				fmt.Println(err)
			} else {
				if err = db.Save(pingLog).Error; err != nil {
					fmt.Println(err)
				}
			}
			return
		}
	}
}
