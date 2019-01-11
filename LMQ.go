/*
 Lightweight Message Queue, version 1.0.0

 Copyright (C) 2018 Misam Saki, http://misam.ir
 Do not Change, Alter, or Remove this Licence
*/

package main

import (
	"bufio"
	"encoding/json"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"strconv"
	"encoding/base64"
	"time"
)

type Config struct {
	Debug bool `json:"debug"`
	Bind string `json:"bind"`
	IpWhiteList[] string `json:"ip_white_list"`
	RecoveryFilePath string `json:"recovery_file_path"`
	FileBasePath string `json:"file_base_path"`
	MessageChannelSize int `json:"message_channel_size"`
	RecoveryChannelSize int `json:"recovery_channel_size"`
}

const (
	TEXT_MESSAGE_TYPE	byte = 0
	FILE_MESSAGE_TYPE	byte = 1
)

func getRecovery(method string, queueName string, message string) string {
	return method + " " + queueName + " " + message
}

func initialRecovery(queues map[string]chan string, recoveryCh chan string, config Config) {
	file, err := os.OpenFile(config.RecoveryFilePath, os.O_RDONLY, 0644)
	if err != nil {
		log.Println(err)
		return
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	if err := scanner.Err(); err != nil {
		log.Println(err)
	}
	var queuesMap = map[string]map[string]string{}
	for scanner.Scan() {
		lineBase64 := scanner.Text()
		lineBytes, err := base64.StdEncoding.DecodeString(lineBase64)
		if err != nil {
			log.Println(err)
			continue
		}
		line := string(lineBytes)
		parts := strings.SplitN(line, " ", 3)
		if len(parts) < 3 {
			log.Println("Incorrect recovery line.")
			continue
		}
		method, queueName, message := parts[0], parts[1], parts[2]
		_, ok := queuesMap[queueName]
		if !ok {
			queuesMap[queueName] = map[string]string{}
		}
		switch method {
		case "SET":
			queuesMap[queueName][message] = message
		case "GET":
			_, ok := queuesMap[queueName][message]
			if ok {
				delete(queuesMap[queueName], message)
			}
		case "DEL":
			delete(queuesMap, queueName)
		default:
			log.Println("Incorrect recovery line.")
			continue
		}
	}
	for queueName, queueMap := range queuesMap {
		queues[queueName] = make(chan string, config.MessageChannelSize)
		for messageKey, message := range queueMap {
			select {
			case queues[queueName] <- message:
				select {
				case recoveryCh <- getRecovery("SET", queueName, messageKey):
					log.Println("Initial ok SET " + queueName + " " + messageKey)
				default:
					log.Println("Initial error (recovery) SET " + queueName + " " + messageKey)
				}
			default:
				log.Println("Initial error SET " + queueName + " " + messageKey)
			}
		}
	}
	err = os.Remove(config.RecoveryFilePath)
	if err != nil {
		log.Println(err)
	}
}

func writingRecovery(recoveryCh chan string, config Config) {
	file, err := os.OpenFile(config.RecoveryFilePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Println(err)
	}
	for recovery := range recoveryCh {
		_, err := file.WriteString(base64.StdEncoding.EncodeToString([]byte(recovery)) + "\n")
		if err != nil {
			log.Println(err)
		}
	}
	file.Close()
}

func listHandler(queues map[string]chan string) gin.HandlerFunc {
	return func (context *gin.Context) {
		var queueNamesLines = ""
		for queueName := range queues {
			queueNamesLines += queueName + "\n"
		}
		context.String(http.StatusOK, queueNamesLines)
		return
	}
}

func countHandler(queues map[string]chan string) gin.HandlerFunc {
	return func (context *gin.Context) {
		queueName := context.Param("queue")
		_, ok := queues[queueName]
		if !ok {
			context.String(http.StatusNotFound, "Queue not exists!")
			return
		}
		context.String(http.StatusOK, strconv.Itoa(len(queues[queueName])))
		return
	}
}

func skipHandler(queues map[string]chan string) gin.HandlerFunc {
	return func (context *gin.Context) {
		queueName := context.Param("queue")
		_, ok := queues[queueName]
		if !ok {
			context.String(http.StatusNotFound, "Queue not exists!")
			return
		}
		number := context.Param("number")
		n, err := strconv.Atoi(number)
		if err != nil {
			context.String(http.StatusBadRequest, "Number must be a integer!")
			return
		}
		var messages[] string
		for i := 0; i < n; i++ {
			select {
			case message := <- queues[queueName]:
				messages = append(messages, message)
				queues[queueName] <- message
			default:
				break
			}
		}
		context.String(http.StatusOK, "OK.")
		return
	}
}

func setHandler(queues map[string]chan string, recoveryCh chan string, config Config) gin.HandlerFunc {
	return func (context *gin.Context) {
		queueName := context.Param("queue")
		_, ok := queues[queueName]
		if !ok {
			queues[queueName] = make(chan string, config.MessageChannelSize)
		}
		message := context.Param("message")
		message = message[1:]
		if message == "" {
			context.String(http.StatusBadRequest, "Message is empty!")
			return
		}
		messageType := TEXT_MESSAGE_TYPE
		messageParts := strings.SplitN(message, ":", 2)
		if len(messageParts) > 1 {
			switch messageParts[0] {
			case "file":
				messageType = FILE_MESSAGE_TYPE
			}
		}
		if messageType == FILE_MESSAGE_TYPE {
			if _, err := os.Stat(config.FileBasePath + messageParts[1]); os.IsNotExist(err) {
				context.String(http.StatusNotAcceptable, "File not exists!")
				return
			}
		}
		uid := strconv.FormatInt(time.Now().UnixNano(), 10)
		message = uid + "@" + message
		select {
		case queues[queueName] <- message:
			select {
			case recoveryCh <-  getRecovery("SET", queueName, message):
				context.String(http.StatusOK, "OK.")
				return
			default:
				context.String(http.StatusInternalServerError, "Internal server error!")
				return
			}
		default:
			context.String(http.StatusInternalServerError, "Internal server error!")
			return
		}
	}
}

func getHandler(queues map[string]chan string, recoveryCh chan string, config Config) gin.HandlerFunc {
	return func (context *gin.Context) {
		queueName := context.Param("queue")
		_, ok := queues[queueName]
		if !ok {
			context.String(http.StatusNotFound, "Queue not exists!")
			return
		}
		select {
		case message := <- queues[queueName]:
			select {
			case recoveryCh <- getRecovery("GET", queueName, message):
				messageParts := strings.SplitN(message, "@", 2)
				uid, message := messageParts[0], messageParts[1]
				context.Header("Uid", uid)
				context.String(http.StatusOK, message)
				return
			default:
				context.String(http.StatusInternalServerError, "Internal server error!")
				return
			}
		default:
			context.String(http.StatusGone, "Queue is empty!")
			return
		}
	}
}

func responseMessage(context *gin.Context, config Config, uid string, message string) {
	messageType := TEXT_MESSAGE_TYPE
	messageParts := strings.SplitN(message, ":", 2)
	if len(messageParts) > 1 {
		switch messageParts[0] {
		case "file":
			messageType = FILE_MESSAGE_TYPE
		}
	}
	switch messageType {
	case FILE_MESSAGE_TYPE:
		bytes, err := ioutil.ReadFile(config.FileBasePath + messageParts[1])
		if err != nil {
			log.Println(err)
			if os.IsNotExist(err) {
				context.String(http.StatusNotFound, "File not found!")
				return
			}
			context.String(http.StatusInternalServerError, "Internal server error!")
			return
		}
		if uid != "" {
			context.Header("Uid", uid)
		}
		context.Header("Message", message)
		contentType := http.DetectContentType(bytes)
		context.Data(http.StatusOK, contentType, bytes)
		return
	default:
		if uid != "" {
			context.Header("Uid", uid)
		}
		context.String(http.StatusOK, message)
		return
	}
}

func fetchHandler(queues map[string]chan string, recoveryCh chan string, config Config) gin.HandlerFunc {
	return func (context *gin.Context) {
		queueName := context.Param("queue")
		_, ok := queues[queueName]
		if !ok {
			context.String(http.StatusNotFound, "Queue not exists!")
			return
		}
		select {
		case message := <- queues[queueName]:
			select {
			case recoveryCh <- getRecovery("GET", queueName, message):
				messageParts := strings.SplitN(message, "@", 2)
				uid, message := messageParts[0], messageParts[1]
				responseMessage(context, config, uid, message)
				return
			default:
				context.String(http.StatusInternalServerError, "Internal server error!")
				return
			}
		default:
			context.String(http.StatusGone, "Queue is empty!")
			return
		}
	}
}

func downloadHandler(config Config) gin.HandlerFunc {
	return func (context *gin.Context) {
		message := context.Param("message")
		message = message[1:]
		if message == "" {
			context.String(http.StatusBadRequest, "Message is empty!")
			return
		} else {
			responseMessage(context, config, "", message)
			return
		}
	}
}

func deleteHandler(queues map[string]chan string, recoveryCh chan string, config Config) gin.HandlerFunc {
	return func (context *gin.Context) {
		queueName := context.Param("queue")
		_, ok := queues[queueName]
		if !ok {
			context.String(http.StatusNotFound, "Queue not exists!")
			return
		}
		delete(queues, queueName)
		select {
		case recoveryCh <- getRecovery("DEL", queueName, ""):
			context.String(http.StatusOK, "OK.")
			return
		default:
			context.String(http.StatusInternalServerError, "Internal server error!")
			return
		}
	}
}

func iPWhiteList(whitelist map[string]bool) gin.HandlerFunc {
	return func (context *gin.Context) {
		if !whitelist[context.ClientIP()] {
			context.String(http.StatusForbidden, "Permission denied!")
			return
		}
	}
}

func main() {
	config_path := "config.json"
	if len(os.Args) > 1 {
		config_path = os.Args[1]
	}

	configBytes, err := ioutil.ReadFile(config_path)
	if err != nil {
		log.Fatal(err)
	}
	var config Config
	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		log.Fatal(err)
	}

	queues := make(map[string]chan string)
	recoveryCh := make(chan string, config.RecoveryChannelSize)

	initialRecovery(queues, recoveryCh, config)

	go writingRecovery(recoveryCh, config)

	if !config.Debug {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.Default()
	router.Use(gzip.Gzip(gzip.DefaultCompression))
	ipWhiteList := make(map[string]bool)
	for _, ip := range config.IpWhiteList {
		ipWhiteList[ip] = true
	}
	router.Use(iPWhiteList(ipWhiteList))

	router.GET("/list", listHandler(queues))
	router.GET("/count/:queue", countHandler(queues))
	router.GET("/skip/:queue/:number", skipHandler(queues))
	router.GET("/set/:queue/*message", setHandler(queues, recoveryCh, config))
	router.GET("/get/:queue", getHandler(queues, recoveryCh, config))
	router.GET("/fetch/:queue", fetchHandler(queues, recoveryCh, config))
	router.GET("/download/*message", downloadHandler(config))
	router.GET("/delete/:queue", deleteHandler(queues, recoveryCh, config))

	router.GET("/help", func (context *gin.Context) {
		help := "Methods:\n"
		help += "/list						List of the queues.\n"
		help += "/count/:queue				Number of messages in a queue.\n"
		help += "/skip/:queue/:number		Skip messages in the queue.\n"
		help += "/set/:queue/:message		Set a message in the queue.\n"
		help += "/get/:queue				Get a message in the queue.\n"
		help += "/fetch/:queue				Fetch a message with content in a queue.\n"
		help += "/download/:message			Download content of the message.\n"
		help += "/delete/:queue				Delete the queue.\n"
		context.String(http.StatusOK, help)
		return
	})
	router.GET("/version", func (context *gin.Context) {
		context.String(http.StatusOK, "1.0.0")
		return
	})
	router.GET("/copyright", func (context *gin.Context) {
		copyright := `
			***
			Lightweight Message Queue, version 1.0.0

			Copyright (C) 2018 Misam Saki, http://misam.ir
			Do not Change, Alter, or Remove this Licence
			***
		`
		context.String(http.StatusOK, strings.Replace(copyright, "\t", "", -1))
		return
	})

	router.Run(config.Bind)
}
