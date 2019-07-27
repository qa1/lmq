package main

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
	Utils "./utils"
)

func main()  {
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

	filenames, err := ioutil.ReadDir(config.RecoveryDirPath)
	if err != nil {
		log.Fatalln(err)
	}
	var queuesMap = map[string]map[string]int{}
	for _, filename := range filenames {
		// Check open files (O_EXCL not work in some OS)
		err := os.Rename(config.RecoveryDirPath + filename.Name(), config.RecoveryDirPath + filename.Name() + ".tmp")
		if err != nil {
			log.Println(err)
			continue
		}
		file, err := os.OpenFile(config.RecoveryDirPath + filename.Name() + ".tmp", os.O_RDONLY, 0644)
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
		err = os.Remove(config.RecoveryDirPath + filename.Name() + ".tmp")
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
					file, err = os.OpenFile(config.RecoveryDirPath + strconv.FormatInt(time.Now().UnixNano(), 10), os.O_WRONLY|os.O_CREATE, 0644)
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
