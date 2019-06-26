/*
 Lightweight Message Queue, version 1.2.0

 Copyright (C) 2018 Misam Saki, http://misam.ir
 Do not Change, Alter, or Remove this Licence
*/

package main

import (
	"bufio"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Debug                bool     `json:"debug"`
	BindAddressList      []string `json:"bind_address_list"`
	IpWhiteList          []string `json:"ip_white_list"`
	RecoveryFilePath     string   `json:"recovery_file_path"`
	FileBasePath         string   `json:"file_base_path"`
	MsqlConnectionString string   `json:"msql_connection_string"`
	MessageChannelSize   int      `json:"message_channel_size"`
	RecoveryChannelSize  int      `json:"recovery_channel_size"`
}

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
	defer file.Close()
	for recovery := range recoveryCh {
		_, err := file.WriteString(base64.StdEncoding.EncodeToString([]byte(recovery)) + "\n")
		if err != nil {
			log.Println(err)
		}
	}
}

func listHandler(queues map[string]chan string) gin.HandlerFunc {
	return func(context *gin.Context) {
		var queueNamesLines = ""
		for queueName := range queues {
			queueNamesLines += queueName + "\n"
		}
		context.String(http.StatusOK, queueNamesLines)
		return
	}
}

func countHandler(queues map[string]chan string) gin.HandlerFunc {
	return func(context *gin.Context) {
		queueName := context.Param("queue")
		_, ok := queues[queueName]
		if !ok {
			context.String(http.StatusNotFound, "Queue not exists!")
			context.Abort()
			return
		}
		context.String(http.StatusOK, strconv.Itoa(len(queues[queueName])))
		return
	}
}

func skipHandler(queues map[string]chan string) gin.HandlerFunc {
	return func(context *gin.Context) {
		queueName := context.Param("queue")
		_, ok := queues[queueName]
		if !ok {
			context.String(http.StatusNotFound, "Queue not exists!")
			context.Abort()
			return
		}
		number := context.Param("number")
		n, err := strconv.Atoi(number)
		if err != nil {
			context.String(http.StatusBadRequest, "Number must be a integer!")
			context.Abort()
			return
		}
		var messages []string
		for i := 0; i < n; i++ {
			select {
			case message := <-queues[queueName]:
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
	return func(context *gin.Context) {
		queueName := context.Param("queue")
		_, ok := queues[queueName]
		if !ok {
			queues[queueName] = make(chan string, config.MessageChannelSize)
		}
		message := context.Param("message")
		message = message[1:]
		if message == "" {
			context.String(http.StatusBadRequest, "Message is empty!")
			context.Abort()
			return
		}
		messageParts := strings.SplitN(message, ":", 2)
		if len(messageParts) > 1 {
			switch messageParts[0] {
			case "file":
				if _, err := os.Stat(config.FileBasePath + messageParts[1]); os.IsNotExist(err) {
					context.String(http.StatusNotAcceptable, "File not exists!")
					context.Abort()
					return
				}
			case "mysql":
				recordName := strings.SplitN(messageParts[1], "/", 2)
				if len(recordName) != 2 {
					context.String(http.StatusNotAcceptable, "Record name not valid!")
					context.Abort()
					return
				}
				table, id := recordName[0], recordName[1]
				db, err := sql.Open("mysql", config.MsqlConnectionString)
				if err != nil {
					log.Println(err)
					context.String(http.StatusInternalServerError, "Internal server error!")
					context.Abort()
					return
				}
				defer db.Close()
				rows, err := db.Query("SELECT data FROM " + table + " WHERE id = " + id + ";")
				if err != nil {
					log.Println(err)
					context.String(http.StatusInternalServerError, "Internal server error!")
					context.Abort()
					return
				}
				defer rows.Close()
				if !rows.Next() {
					context.String(http.StatusNotAcceptable, "Record not exists!")
					context.Abort()
					return
				}
			}
		}
		uid := strconv.FormatInt(time.Now().UnixNano(), 10)
		message = uid + "@" + message
		select {
		case queues[queueName] <- message:
			select {
			case recoveryCh <- getRecovery("SET", queueName, message):
				context.String(http.StatusOK, "OK.")
				return
			default:
				context.String(http.StatusInternalServerError, "Internal server error!")
				context.Abort()
				return
			}
		default:
			context.String(http.StatusInternalServerError, "Internal server error!")
			context.Abort()
			return
		}
	}
}

func getHandler(queues map[string]chan string, recoveryCh chan string) gin.HandlerFunc {
	return func(context *gin.Context) {
		queueName := context.Param("queue")
		_, ok := queues[queueName]
		if !ok {
			context.String(http.StatusNotFound, "Queue not exists!")
			context.Abort()
			return
		}
		select {
		case message := <-queues[queueName]:
			select {
			case recoveryCh <- getRecovery("GET", queueName, message):
				messageParts := strings.SplitN(message, "@", 2)
				uid, message := messageParts[0], messageParts[1]
				context.Header("Uid", uid)
				context.String(http.StatusOK, message)
				return
			default:
				context.String(http.StatusInternalServerError, "Internal server error!")
				context.Abort()
				return
			}
		default:
			context.String(http.StatusGone, "Queue is empty!")
			context.Abort()
			return
		}
	}
}

func responseMessage(context *gin.Context, config Config, uid string, message string) {
	messageParts := strings.SplitN(message, ":", 2)
	if len(messageParts) > 1 {
		switch messageParts[0] {
		case "file":
			bytes, err := ioutil.ReadFile(config.FileBasePath + messageParts[1])
			if err != nil {
				log.Println(err)
				if os.IsNotExist(err) {
					context.String(http.StatusNotFound, "File not found!")
					context.Abort()
					return
				}
				context.String(http.StatusInternalServerError, "Internal server error!")
				context.Abort()
				return
			}
			if uid != "" {
				context.Header("Uid", uid)
			}
			context.Header("Message", message)
			contentType := http.DetectContentType(bytes)
			context.Data(http.StatusOK, contentType, bytes)
			return
		case "mysql":
			recordName := strings.SplitN(messageParts[1], "/", 2)
			if len(recordName) != 2 {
				context.String(http.StatusNotAcceptable, "Record name not valid!")
				context.Abort()
				return
			}
			table, id := recordName[0], recordName[1]
			db, err := sql.Open("mysql", config.MsqlConnectionString)
			if err != nil {
				log.Println(err)
				context.String(http.StatusInternalServerError, "Internal server error!")
				context.Abort()
				return
			}
			defer db.Close()
			rows, err := db.Query("SELECT data FROM " + table + " WHERE id = " + id + ";")
			if err != nil {
				log.Println(err)
				context.String(http.StatusInternalServerError, "Internal server error!")
				context.Abort()
				return
			}
			defer rows.Close()
			if rows.Next() {
				var data []byte
				err = rows.Scan(&data)
				if err != nil {
					log.Println(err)
					context.String(http.StatusInternalServerError, "Internal server error!")
					context.Abort()
					return
				}
				context.Header("Message", id)
				context.Data(http.StatusOK, "text/plain", data)
				return
			} else {
				context.String(http.StatusNotAcceptable, "Record not exists!")
				context.Abort()
				return
			}
		default:
			if uid != "" {
				context.Header("Uid", uid)
			}
			context.String(http.StatusOK, message)
			return
		}
	}
}

func fetchHandler(queues map[string]chan string, recoveryCh chan string, config Config) gin.HandlerFunc {
	return func(context *gin.Context) {
		queueName := context.Param("queue")
		_, ok := queues[queueName]
		if !ok {
			context.String(http.StatusNotFound, "Queue not exists!")
			context.Abort()
			return
		}
		select {
		case message := <-queues[queueName]:
			select {
			case recoveryCh <- getRecovery("GET", queueName, message):
				messageParts := strings.SplitN(message, "@", 2)
				uid, message := messageParts[0], messageParts[1]
				responseMessage(context, config, uid, message)
				return
			default:
				context.String(http.StatusInternalServerError, "Internal server error!")
				context.Abort()
				return
			}
		default:
			context.String(http.StatusGone, "Queue is empty!")
			context.Abort()
			return
		}
	}
}

func downloadHandler(config Config) gin.HandlerFunc {
	return func(context *gin.Context) {
		message := context.Param("message")
		message = message[1:]
		if message == "" {
			context.String(http.StatusBadRequest, "Message is empty!")
			context.Abort()
			return
		} else {
			responseMessage(context, config, "", message)
			return
		}
	}
}

func deleteHandler(queues map[string]chan string, recoveryCh chan string) gin.HandlerFunc {
	return func(context *gin.Context) {
		queueName := context.Param("queue")
		_, ok := queues[queueName]
		if !ok {
			context.String(http.StatusNotFound, "Queue not exists!")
			context.Abort()
			return
		}
		delete(queues, queueName)
		select {
		case recoveryCh <- getRecovery("DEL", queueName, ""):
			context.String(http.StatusOK, "OK.")
			return
		default:
			context.String(http.StatusInternalServerError, "Internal server error!")
			context.Abort()
			return
		}
	}
}

func iPWhiteList(whitelist map[string]bool) gin.HandlerFunc {
	return func(context *gin.Context) {
		if !whitelist[context.ClientIP()] {
			context.String(http.StatusForbidden, "Permission denied!")
			context.Abort()
			return
		}
	}
}

func main() {
	configPath := "config.json"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	configBytes, err := ioutil.ReadFile(configPath)
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
	router.GET("/get/:queue", getHandler(queues, recoveryCh))
	router.GET("/fetch/:queue", fetchHandler(queues, recoveryCh, config))
	router.GET("/download/*message", downloadHandler(config))
	router.GET("/delete/:queue", deleteHandler(queues, recoveryCh))

	router.GET("/help", func(context *gin.Context) {
		help := "Methods:\n\n"
		help += "GET /list\n" +
			"\tList of the queues.\n\n"
		help += "GET /count/[queue]\n" +
			"\tNumber of messages in a queue.\n\n"
		help += "GET /skip/[queue]/[number]\n" +
			"\tSkip messages in the queue.\n\n"
		help += "GET /set/[queue]/[message]\n" +
			"\tSet a message in the queue.\n\n"
		help += "GET /get/[queue]\n" +
			"\tGet a message in the queue.\n\n"
		help += "GET /fetch/[queue]\n" +
			"\tFetch a message with content in a queue.\n\n"
		help += "GET /download/[message]\n" +
			"\tDownload content of the message.\n\n"
		help += "GET /delete/[queue]\n" +
			"\tDelete the queue.\n\n"

		help += "Message types:\n\n"
		help += "[message]\n" +
			"\tPure text message (for short messages).\n\n"
		help += "file:[file_path]\n" +
			"\tFile content as the message.\n\n"
		help += "mysql:[table_name]/[id]\n" +
			"\tMysql record as the message ('id' field as identification and 'data' field as content).\n"
		context.String(http.StatusOK, help)
		return
	})
	router.GET("/version", func(context *gin.Context) {
		context.String(http.StatusOK, "1.2.0")
		return
	})
	router.GET("/copyright", func(context *gin.Context) {
		copyright := `
			***
			Lightweight Message Queue, version 1.2.0

			Copyright (C) 2018 Misam Saki, http://misam.ir
			Do not Change, Alter, or Remove this Licence
			***
		`
		context.String(http.StatusOK, strings.Replace(copyright, "\t", "", -1))
		return
	})

	i := 0
	for ; i < len(config.BindAddressList)-1; i++ {
		go router.Run(config.BindAddressList[i])
	}
	if err = router.Run(config.BindAddressList[i]); err != nil {
		panic(err)
	}
}
