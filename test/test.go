package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
)

func call(payload string) string {
	response, err := http.Get("http://127.0.0.1:3000/" + payload)
	if err != nil {
		log.Fatalln(err)
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatalln(err)
	}

	return string(body)
}

func main()  {
	for i := 0; i < 1000; i ++ {
		fmt.Println(i)
		call("set/test/x" + strconv.Itoa(i))
	}

	for i := 0; i < 500; i ++ {
		fmt.Println(i)
		call("get/test")
	}
}
