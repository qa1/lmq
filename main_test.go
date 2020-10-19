package main

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/assert/v2"
	Utils "github.com/justmisam/lmq/utils"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func init()  {
	config = Utils.Config{
		Debug: false,
		BindAddressList: []string{},
		FileBasePath: "./raw/",
		MsqlConnectionString: "",
		QueueInitSize: 1000000,
		RecoveryDirPath: "./rec/",
		RecoveryFileSize: 1000000,
		IpWhiteList: []string{},
		GzipEnable: false,
	}
	go writingRecovery()
	initialRecovery()
	gin.SetMode(gin.TestMode)
}

func TestSetTextMessage(t *testing.T)  {
	router := setupRouter()

	for i := 0; i < 1000; i++ {
		req, _ := http.NewRequest(http.MethodGet, "/set/tmp/message-" + strconv.Itoa(i), nil)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "OK.", w.Body.String())
	}
}

func TestGetTextMessage(t *testing.T)  {
	router := setupRouter()

	for i := 0; i < 1000; i++ {
		req, _ := http.NewRequest(http.MethodGet, "/get/tmp", nil)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "message-" + strconv.Itoa(i), w.Body.String())
	}
}
