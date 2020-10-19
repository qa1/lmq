package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	Utils "github.com/justmisam/lmq/utils"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

func increaseQueueSize(queues map[string]chan string, queueName string, size int) {
	threshold := 0
	if cap(queues[queueName])-len(queues[queueName]) <= threshold {
		out := make(chan string, cap(queues[queueName])+size)
		for v := range queues[queueName] {
			out <- v
			if len(queues[queueName]) == 0 {
				break
			}
		}
		queues[queueName] = out
	}
}

func initialRecovery(queues map[string]chan string, config Utils.Config) {
	fileInfos, err := ioutil.ReadDir(config.RecoveryDirPath)
	if err != nil {
		log.Fatalln(err)
	}
	filenames := make([]string, len(fileInfos))
	for i := 0; i < len(fileInfos); i++ {
		filenames[i] = fileInfos[i].Name()
	}
	sort.Strings(filenames)
	var queuesMap = map[string]map[string]int{}
	for _, filename := range filenames {
		file, err := os.OpenFile(config.RecoveryDirPath+filename, os.O_RDONLY|os.O_EXCL, 0644)
		if err != nil {
			log.Fatalln(err)
		}
		scanner := bufio.NewScanner(file)
		if err := scanner.Err(); err != nil {
			log.Println(err)
		}
		for scanner.Scan() {
			lineEscape := scanner.Text()
			line, err := url.QueryUnescape(lineEscape)
			if err != nil {
				log.Println(err)
				continue
			}
			parts := strings.SplitN(line, " ", 3)
			if len(parts) < 3 {
				log.Println("Incorrect recovery line.")
				continue
			}
			method, queueName, message := parts[0], parts[1], parts[2]
			_, isQueueExist := queuesMap[queueName]
			if !isQueueExist {
				queuesMap[queueName] = map[string]int{}
			}
			count, isMessageExist := queuesMap[queueName][message]
			if !isMessageExist {
				count = 0
			}
			switch method {
			case "SET":
				queuesMap[queueName][message] = count + 1
			case "GET":
				queuesMap[queueName][message] = count - 1
			case "DEL":
				delete(queuesMap, queueName)
			default:
				log.Println("Incorrect recovery line.")
				continue
			}
		}
		file.Close()
	}
	for queueName, queueMap := range queuesMap {
		queues[queueName] = make(chan string, config.QueueInitSize)
		for message, count := range queueMap {
			for i := 0; i < count; i++ {
				increaseQueueSize(queues, queueName, config.QueueInitSize)
				select {
				case queues[queueName] <- message:
					log.Println("Initial ok SET " + queueName + " " + message)
				default:
					log.Println("Initial error SET " + queueName + " " + message)
				}
			}
		}
	}
}

func writingRecovery(recoveryCh chan string, config Utils.Config) {
	var file *os.File = nil
	var err error = nil
	recoveryFileSize := config.RecoveryFileSize
	for recovery := range recoveryCh {
		if recoveryFileSize >= config.RecoveryFileSize {
			if file != nil {
				file.Close()
			}
			recoveryFileSize = 0
			recoveryFileName := strconv.FormatInt(time.Now().UnixNano(), 10)
			file, err = os.OpenFile(config.RecoveryDirPath+recoveryFileName, os.O_WRONLY|os.O_CREATE, 0644)

			if err != nil {
				log.Fatalln(err)
			}
		}
		_, err := file.WriteString(url.QueryEscape(recovery) + "\n")
		if err != nil {
			log.Println(err)
		}
		recoveryFileSize++
	}
}

func cleanupRecovery() {
	configPath := "config.json"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	configBytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Fatalln(err)
	}
	var config Utils.Config
	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		log.Fatalln(err)
	}

	fileInfos, err := ioutil.ReadDir(config.RecoveryDirPath)
	if err != nil {
		log.Fatalln(err)
	}
	filenames := make([]string, len(fileInfos))
	for i := 0; i < len(fileInfos); i++ {
		filename := fileInfos[i].Name()
		filenames[i] = filename
	}
	sort.Strings(filenames)
	var queuesMap = map[string]map[string]int{}
	for i := 0; i < len(filenames)-1; i++ {
		filename := filenames[i]
		file, err := os.OpenFile(config.RecoveryDirPath+filename, os.O_RDONLY, 0644)
		if err != nil {
			log.Println(err)
			continue
		}
		scanner := bufio.NewScanner(file)
		if err := scanner.Err(); err != nil {
			log.Println(err)
			continue
		}
		for scanner.Scan() {
			lineEscape := scanner.Text()
			line, err := url.QueryUnescape(lineEscape)
			if err != nil {
				log.Println(err)
				continue
			}
			parts := strings.SplitN(line, " ", 3)
			if len(parts) < 3 {
				log.Println("Incorrect recovery line.")
				continue
			}
			method, queueName, message := parts[0], parts[1], parts[2]
			_, isQueueExist := queuesMap[queueName]
			if !isQueueExist {
				queuesMap[queueName] = map[string]int{}
			}
			count, isMessageExist := queuesMap[queueName][message]
			if !isMessageExist {
				count = 0
			}
			switch method {
			case "SET":
				queuesMap[queueName][message] = count + 1
			case "GET":
				queuesMap[queueName][message] = count - 1
			case "DEL":
				delete(queuesMap, queueName)
			default:
				log.Println("Incorrect recovery line.")
				continue
			}
		}
		file.Close()
		err = os.Remove(config.RecoveryDirPath + filename)
		if err != nil {
			log.Println(err)
		}
	}

	var file *os.File = nil
	recoveryFileSize := config.RecoveryFileSize
	for queueName, queueMap := range queuesMap {
		for message, count := range queueMap {
			recovery := Utils.GetRecovery("SET", queueName, message)
			for i := 0; i < count; i++ {
				if recoveryFileSize >= config.RecoveryFileSize {
					if file != nil {
						file.Close()
					}
					recoveryFileSize = 0
					file, err = os.OpenFile(config.RecoveryDirPath+"0"+strconv.FormatInt(time.Now().UnixNano(), 10), os.O_WRONLY|os.O_CREATE, 0644)
					if err != nil {
						log.Fatalln(err)
					}
				}
				_, err := file.WriteString(url.QueryEscape(recovery) + "\n")
				if err != nil {
					log.Println(err)
				}
				recoveryFileSize++
			}
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

func setHandler(queues map[string]chan string, recoveryCh chan string, config Utils.Config) gin.HandlerFunc {
	return func(context *gin.Context) {
		queueName := context.Param("queue")
		_, ok := queues[queueName]
		if !ok {
			queues[queueName] = make(chan string, config.QueueInitSize)
		}
		increaseQueueSize(queues, queueName, config.QueueInitSize)
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
		select {
		case queues[queueName] <- message:
			select {
			case recoveryCh <- Utils.GetRecovery("SET", queueName, message):
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
			case recoveryCh <- Utils.GetRecovery("GET", queueName, message):
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

func responseMessage(context *gin.Context, config Utils.Config, message string) {
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
				context.Header("Message", message)
				context.Data(http.StatusOK, "text/plain", data)
				return
			} else {
				context.String(http.StatusNotAcceptable, "Record not exists!")
				context.Abort()
				return
			}
		default:
			context.String(http.StatusOK, message)
			return
		}
	}
}

func fetchHandler(queues map[string]chan string, recoveryCh chan string, config Utils.Config) gin.HandlerFunc {
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
			case recoveryCh <- Utils.GetRecovery("GET", queueName, message):
				responseMessage(context, config, message)
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

func downloadHandler(config Utils.Config) gin.HandlerFunc {
	return func(context *gin.Context) {
		message := context.Param("message")
		message = message[1:]
		if message == "" {
			context.String(http.StatusBadRequest, "Message is empty!")
			context.Abort()
			return
		} else {
			responseMessage(context, config, message)
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
		case recoveryCh <- Utils.GetRecovery("DEL", queueName, ""):
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
	if len(os.Args) > 2 {
		if os.Args[2] == "cleanup" {
			cleanupRecovery()
			return
		}
	}

	configBytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Fatal(err)
	}
	var config Utils.Config
	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		log.Fatal(err)
	}

	queues := make(map[string]chan string)
	recoveryCh := make(chan string, 1000)

	go writingRecovery(recoveryCh, config)

	initialRecovery(queues, config)

	if !config.Debug {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.Default()

	if config.GzipEnable {
		router.Use(gzip.Gzip(gzip.DefaultCompression))
	}

	if len(config.IpWhiteList) > 0 {
		ipWhiteList := make(map[string]bool)
		for _, ip := range config.IpWhiteList {
			ipWhiteList[ip] = true
		}
		router.Use(iPWhiteList(ipWhiteList))
	}

	router.GET("/list", listHandler(queues))
	router.GET("/count/:queue", countHandler(queues))
	router.GET("/skip/:queue/:number", skipHandler(queues))
	router.GET("/set/:queue/*message", setHandler(queues, recoveryCh, config))
	router.GET("/get/:queue", getHandler(queues, recoveryCh))
	router.GET("/fetch/:queue", fetchHandler(queues, recoveryCh, config))
	router.GET("/download/*message", downloadHandler(config))
	router.GET("/delete/:queue", deleteHandler(queues, recoveryCh))

	i := 0
	for ; i < len(config.BindAddressList)-1; i++ {
		go router.Run(config.BindAddressList[i])
	}
	if err = router.Run(config.BindAddressList[i]); err != nil {
		panic(err)
	}
}
