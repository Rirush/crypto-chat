package main

import (
	"crypto-chat/static"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"math/big"
	"sync"
)

var upgrader = websocket.Upgrader{}

var (
	broadcast      []*websocket.Conn
	broadcastMutex sync.Mutex
)

func broadcastEvent(event interface{}) {
	broadcastMutex.Lock()
	defer broadcastMutex.Unlock()

	for _, conn := range broadcast {
		err := conn.WriteJSON(messageResponse{
			Type:   "event",
			Result: event,
		})
		if err != nil {
			fmt.Println("publish failed:", err)
		}
	}
}

type incomingMessage struct {
	ID      string          `json:"id,omitempty"`
	Type    string          `json:"type,omitempty"`
	Request json.RawMessage `json:"request,omitempty"`
}

type messagePublication struct {
	Key       string `json:"key,omitempty"`
	Message   string `json:"message,omitempty"`
	Signature string `json:"signature,omitempty"`
}

type messageResponse struct {
	ID      string      `json:"id,omitempty"`
	Type    string      `json:"type,omitempty"`
	Success bool        `json:"success,omitempty"`
	Error   string      `json:"error,omitempty"`
	Result  interface{} `json:"result,omitempty"`
}

type requestHandler[req any] func(msg *incomingMessage, body req) (messageResponse, error)

type messageHandler func(conn *websocket.Conn, msg *incomingMessage) error

func handlerWrapper[req any](conn *websocket.Conn, msg *incomingMessage, next requestHandler[req]) error {
	body := new(req)

	err := json.Unmarshal(msg.Request, body)
	if err != nil {
		return err
	}

	resp, err := next(msg, *body)
	if err != nil {
		return err
	}

	resp.ID = msg.ID
	resp.Type = "response"

	err = conn.WriteJSON(resp)
	if err != nil {
		return err
	}

	return nil
}

func registerHandler[req any](msgType string, h requestHandler[req]) {
	handlers[msgType] = func(conn *websocket.Conn, msg *incomingMessage) error {
		return handlerWrapper(conn, msg, h)
	}
}

func dispatchMessage(conn *websocket.Conn, msg *incomingMessage) error {
	f, ok := handlers[msg.Type]
	if !ok {
		return fmt.Errorf("unknown message type: %v", msg.Type)
	}

	err := f(conn, msg)
	return err
}

var handlers = map[string]messageHandler{}

func main() {
	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	registerHandler("publish", func(msg *incomingMessage, body messagePublication) (messageResponse, error) {
		key, err := hex.DecodeString(body.Key)
		if err != nil {
			e.Logger.Error(err)
			return messageResponse{
				Error: fmt.Sprintf("key decode failed: %e", err),
			}, nil
		}

		signature, err := hex.DecodeString(body.Signature)
		if err != nil {
			e.Logger.Error(err)
			return messageResponse{
				Error: fmt.Sprintf("signature decode failed: %e", err),
			}, nil
		}

		x, y := elliptic.Unmarshal(elliptic.P384(), key)
		if x == nil {
			e.Logger.Error(errors.New("broken encoding"))
			return messageResponse{
				Error: fmt.Sprintf("key unmarshal failed"),
			}, nil
		}

		if len(signature) != 96 {
			return messageResponse{
				Error: fmt.Sprintf("signature of invalid length: %v", len(signature)),
			}, nil
		}

		pubkey := ecdsa.PublicKey{
			Curve: elliptic.P384(),
			X:     x,
			Y:     y,
		}

		hash := sha256.Sum256([]byte(body.Message))

		r := big.NewInt(0)
		r.SetBytes(signature[:48])

		s := big.NewInt(0)
		s.SetBytes(signature[48:])

		if !ecdsa.Verify(&pubkey, hash[:], r, s) {
			return messageResponse{
				Success: false,
				Error:   "invalid signature",
			}, nil
		}

		broadcastEvent(body)

		return messageResponse{
			Success: true,
			Result:  true,
		}, nil
	})

	e.FileFS("/", "data/index.html", static.Data)
	e.StaticFS("/static", echo.MustSubFS(static.Data, "data/static"))
	e.GET("/ws", func(c echo.Context) error {
		resp := c.Response()
		req := c.Request()

		conn, err := upgrader.Upgrade(resp.Writer, req, nil)
		if err != nil {
			return err
		}

		broadcastMutex.Lock()
		broadcast = append(broadcast, conn)
		id := len(broadcast) - 1
		broadcastMutex.Unlock()

		defer func() {
			broadcastMutex.Lock()
			_ = conn.Close()
			broadcast = append(broadcast[:id], broadcast[id+1:]...)
			broadcastMutex.Unlock()
		}()

		for {
			msg := &incomingMessage{}

			err = conn.ReadJSON(msg)
			if err != nil {
				e.Logger.Error(err)
				return nil
			}

			err = dispatchMessage(conn, msg)
			if err != nil {
				e.Logger.Error(err)
				return nil
			}
		}
	})

	e.Debug = true

	err := e.Start(":8080")
	if err != nil {
		panic(err)
	}
}
