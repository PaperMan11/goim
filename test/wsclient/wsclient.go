package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"flag"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PaperMan11/goim/pkg/utils/randx"
	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
)

var (
	addr     = flag.String("addr", "localhost:50001", "server address")
	userID   = flag.String("user", "test_user_001", "user ID")
	token    = flag.String("token", "test_token", "auth token")
	platform = flag.Int("platform", 1, "platform ID")
)

type HandshakeResponse struct {
	ErrCode int    `json:"errCode"`
	ErrMsg  string `json:"errMsg"`
}

type Request struct {
	ReqIdentifier int    `json:"reqIdentifier"`
	Token         string `json:"token"`
	SenderID      string `json:"senderID"`
	OperationID   string `json:"operationID"`
	MsgIncr       string `json:"msgIncr"`
	Data          []byte `json:"data"`
}

type Response struct {
	ReqIdentifier int    `json:"reqIdentifier"`
	MsgIncr       string `json:"msgIncr"`
	OperationID   string `json:"operationID"`
	ErrCode       int    `json:"errCode"`
	ErrMsg        string `json:"errMsg"`
	Data          []byte `json:"data"`
}

func main() {
	flag.Parse()

	logx.MustSetup(logx.LogConf{
		Stat:     false,
		Mode:     "console",
		Encoding: "plain",
		Level:    "debug",
	})

	// 构建 WebSocket URL（不带查询参数）
	u := url.URL{
		Scheme: "ws",
		Host:   *addr,
		Path:   "/ws",
	}
	query := u.Query()
	query.Set("userID", *userID)
	query.Set("token", *token)
	query.Set("platformID", strconv.Itoa(*platform))
	query.Set("deviceID", "test_device_001")
	query.Set("sendResponse", "true")
	query.Set("isBackground", "false")
	query.Set("operationID", "test_operation_001")
	query.Set("compression", "none")
	query.Set("sdkType", "go")
	query.Set("sdkVersion", "1.0.0")
	u.RawQuery = query.Encode()

	logx.Infof("Connecting to %s", u.String())

	// 建立连接
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		logx.Errorf("Dial failed: %v", err)
		return
	}
	defer c.Close()

	logx.Infof("Connected successfully")

	// 启动接收协程
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				logx.Errorf("Read error: %v", err)
				return
			}
			logx.Infof("Received message: %s", string(message))

			// 尝试解析响应
			var resp Response
			if err := json.Unmarshal(message, &resp); err == nil {
				logx.Infof("Response: req=%d, err=%d, msg=%s",
					resp.ReqIdentifier, resp.ErrCode, resp.ErrMsg)
			}
		}
	}()

	// 设置心跳
	c.SetPongHandler(func(appData string) error {
		logx.Debugf("Received pong")
		return nil
	})

	// 发送心跳
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				logx.Debugf("Sending ping")
				if err := c.WriteMessage(websocket.PingMessage, nil); err != nil {
					logx.Errorf("Ping error: %v", err)
					return
				}
			case <-done:
				return
			}
		}
	}()

	// 命令行交互
	scanner := bufio.NewScanner(os.Stdin)
	logx.Info("Enter command (send/pull/logout/help/quit):")

	for {
		logx.Info("> ")
		if !scanner.Scan() {
			break
		}
		cmd := strings.TrimSpace(scanner.Text())
		if cmd == "" {
			continue
		}

		switch cmd {
		case "quit", "exit":
			logx.Info("Closing connection...")
			sendCloseMessage(c)
			return
		case "help":
			printHelp()
		case "send":
			sendMessage(c)
		case "pull":
			pullMessage(c)
		case "logout":
			logout(c)
		case "ping":
			testPing(c)
		case "status":
			testStatus(c)
		default:
			logx.Errorf("Unknown command: %s", cmd)
			printHelp()
		}
	}

	if err := scanner.Err(); err != nil {
		logx.Errorf("Scanner error: %v", err)
	}
}

func sendMessage(c *websocket.Conn) {
	req := Request{
		ReqIdentifier: 1003, // WSSendMsg
		Token:         *token,
		SenderID:      *userID,
		OperationID:   generateOperationID(),
		MsgIncr:       "1",
		Data:          []byte(`{"toUserID":"user2","content":"hello"}`),
	}

	sendRequest(c, req)
}

func pullMessage(c *websocket.Conn) {
	req := Request{
		ReqIdentifier: 1005, // WSPullMsg
		Token:         *token,
		SenderID:      *userID,
		OperationID:   generateOperationID(),
		MsgIncr:       "1",
		Data:          []byte(`{"conversationID":"conv001","maxSeq":0,"count":10}`),
	}

	sendRequest(c, req)
}

func logout(c *websocket.Conn) {
	req := Request{
		ReqIdentifier: 2003, // WSLogoutMsg
		Token:         *token,
		SenderID:      *userID,
		OperationID:   generateOperationID(),
		MsgIncr:       "1",
	}

	sendRequest(c, req)
	time.Sleep(1 * time.Second)
	logx.Info("Logged out, exiting...")
	os.Exit(0)
}

func testPing(c *websocket.Conn) {
	req := Request{
		ReqIdentifier: 1001, // WSGetNewestSeq
		Token:         *token,
		SenderID:      *userID,
		OperationID:   generateOperationID(),
		MsgIncr:       "1",
	}

	sendRequest(c, req)
}

func testStatus(c *websocket.Conn) {
	req := Request{
		ReqIdentifier: 2004, // WSSetBackgroundStatus
		Token:         *token,
		SenderID:      *userID,
		OperationID:   generateOperationID(),
		MsgIncr:       "1",
		Data:          []byte(`{"status":false}`),
	}

	sendRequest(c, req)
}

func sendRequest(c *websocket.Conn, req Request) {
	data, err := json.Marshal(req)
	if err != nil {
		logx.Errorf("Marshal error: %v", err)
		return
	}

	logx.Infof("Sending request: %s", string(data))
	if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
		logx.Errorf("Write error: %v", err)
	}
}

func sendCloseMessage(c *websocket.Conn) {
	c.WriteMessage(websocket.CloseMessage, nil)
}

func generateOperationID() string {
	b, err := randx.SecureBytes(16)
	if err != nil || len(b) < 16 {
		buf := make([]byte, 16)
		for i := range buf {
			buf[i] = byte(i * 31)
		}
		b = buf
	}
	return base64.URLEncoding.EncodeToString(b)[:20]
}

func printHelp() {
	logx.Info("Available commands:")
	logx.Info("  send    - Send a message")
	logx.Info("  pull    - Pull messages")
	logx.Info("  logout  - Logout and exit")
	logx.Info("  ping    - Test connection (get newest seq)")
	logx.Info("  status  - Set background status")
	logx.Info("  help    - Show this help")
	logx.Info("  quit    - Exit")
}
