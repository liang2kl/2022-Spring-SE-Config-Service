package push

import (
	"fmt"
	"log"
	"net/http"
	"service/internal/model"
	"service/internal/router/resp"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/spf13/viper"
)

var Router *gin.Engine

var service pushService

func Setup() {
	service = pushService{
		upgrader: websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
		clients:  make(map[string][]*client, 10),
		closeCh:  make(chan int),
	}
	Router = gin.Default()
	Router.GET("/push/:config_id", handleConnectionRequest)
	Router.POST("/update/:config_id", handleUpdate)
	Router.POST("/report/:config_id/:code_id", handleErrorReport)
}

func Run() {
	hostname := viper.GetString("hostname")
	port := viper.GetInt("port")
	Router.Run(fmt.Sprintf("%s:%v", hostname, port))
}

func handleUpdate(c *gin.Context) {
	configId := c.Param("config_id")
	if configId == "" {
		resp.Error(c, http.StatusBadRequest, fmt.Sprintf("invalid config id '%s'", configId))
		return
	}

	// validate secret
	secret := c.GetHeader("Secret")
	if secret == "" {
		resp.Error(c, http.StatusBadRequest, "missing secret")
		return
	}

	validSecret := viper.GetString("update-secret")
	if validSecret != secret {
		resp.Error(c, http.StatusForbidden, "mismatched access secret")
		return
	}

	// send update notification
	num := sendUpdateNotification(ConfigUpdateNotification{
		ConfigID:   configId,
		UpdateTime: time.Now().Unix(),
	})

	// send response
	resp.Ok(c, http.StatusOK, map[string]int{"client_num": num})
}

type ErrorReportBody struct {
	ErrTime int    `json:"err_time"`
	Message string `json:"message"`
}

func handleErrorReport(c *gin.Context) {
	codeId := c.Param("code_id")
	if codeId == "" {
		resp.Error(c, http.StatusBadRequest, "missing code id")
		return
	}

	body := ErrorReportBody{}
	if err := c.ShouldBindJSON(&body); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid request body:"+err.Error())
		return
	}

	config := handleSecretRequest(c)
	if config == nil {
		return
	}

	if !config.IsValid() {
		resp.Error(c, http.StatusForbidden, "the config is not active")
		return
	}

	if codeId != config.ReleasedCode && codeId != config.GrayReleaseCode {
		resp.Error(c, http.StatusBadRequest, fmt.Sprintf(
			"code id %s does not associated to any current version of config %s",
			codeId, config.ConfigID))
		return
	}

	var code model.Code
	if err := model.DB.First(&code, "code_id = ?", codeId).Error; err != nil {
		log.Printf("code record %s not found", codeId)
		resp.Error(c, http.StatusInternalServerError,
			fmt.Sprintf("code record for id %s not found", codeId))
		return
	}

	if code.IsBroken {
		resp.Error(c, http.StatusForbidden, "the config is inactive due to errors")
		return
	}

	report := model.ErrorReport{
		Time:    body.ErrTime,
		Message: body.Message,
	}

	if err := model.DB.Model(&code).Association("ErrorReports").Append(&report); err != nil {
		resp.Error(c, http.StatusInternalServerError,
			fmt.Sprintln("fail to associate error record:", err))
		return
	}

	reportNum := code.ErrorCount
	threshold := viper.GetInt("code-break-threshold")

	if err := model.DB.Model(&code).Updates(map[string]interface{}{
		"is_broken": reportNum > threshold,
		"err_count": reportNum + 1,
	}).Error; err != nil {
		log.Println(err)
	}

	resp.Ok(c, http.StatusOK, "")
}

func handleConnectionRequest(c *gin.Context) {
	config := handleSecretRequest(c)
	if config == nil {
		return
	}

	// finally, setup connection
	conn, err := service.upgrader.Upgrade(c.Writer, c.Request, nil)

	if err != nil {
		log.Print("upgrade:", err)
		resp.Error(c, http.StatusInternalServerError, "fail to setup websocket connection")
		return
	}

	client := &client{
		ch:       make(chan ConfigUpdateNotification),
		configId: config.ConfigID,
	}
	service.addClient(client)

	go writer(conn, client)
}

func handleSecretRequest(c *gin.Context) *model.Config {
	// get secret
	secret := c.Request.Header.Get("Secret")
	if secret == "" {
		resp.Error(c, http.StatusBadRequest, "missing secret")
		return nil
	}

	// get config id from path
	configId := c.Param("config_id")
	if configId == "" {
		resp.Error(c, http.StatusBadRequest, fmt.Sprintf("invalid config id '%s'", configId))
		return nil
	}

	// get config record from db
	var config model.Config
	if err := model.DB.First(&config, "config_id = ?", configId).Error; err != nil {
		resp.Error(c, http.StatusBadRequest, "config record does not exist")
		return nil
	}

	// validate secret
	if config.Secret != secret {
		resp.Error(c, http.StatusForbidden, "mismatched access secret")
		return nil
	}

	return &config
}

func writer(conn *websocket.Conn, c *client) {
	defer conn.Close()
	defer service.removeClient(c)

	config := viper.GetStringMap("websocket")
	pingInterval := time.Duration(config["ping"].(int))
	pongInterval := time.Duration(config["pong"].(int))

	ticker := time.NewTicker(pingInterval * time.Millisecond)
	defer ticker.Stop()

loop:
	for {
		select {
		case noti := <-c.ch:
			err := conn.WriteJSON(noti)
			if err != nil {
				break loop
			}
		case <-service.closeCh:
			break loop
		case <-ticker.C:
			deadline := time.Now().Add(pongInterval * time.Millisecond)
			err := conn.WriteControl(websocket.PingMessage, []byte{}, deadline)
			if err != nil {
				break loop
			}
		}
	}
}

func sendUpdateNotification(content ConfigUpdateNotification) int {
	service.RLock()
	clients := getClients(content.ConfigID)
	service.RUnlock()

	for _, client := range clients {
		client.ch <- content
	}

	return len(clients)
}
