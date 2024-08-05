package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"

	cache "remote-diffusion-server/cache"
	fileserver "remote-diffusion-server/fileserver"
	frps "remote-diffusion-server/frps"
	logs "remote-diffusion-server/logs"
	model "remote-diffusion-server/model"
	nginx "remote-diffusion-server/nginx"
	utils "remote-diffusion-server/utils"
)

var wg sync.WaitGroup
var activeClients = cache.NewUserCache()
var onlineServers = cache.NewUserCache()
var registeredUsers = model.NewRegisteredKeys()
var frpsCache = frps.NewFrpsCache()
var downloadJobQueue = fileserver.NewDownloadJobQueue()

func main() {
	count, err := registeredUsers.Count()
	if err != nil {
		log.Println("Error getting registered keys count:", err)
		return
	}
	log.Println("Registered keys count:", count)

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	realoadCacheFromLogs()

	exitChan := make(chan struct{})
	go periodicCheck(exitChan)
	startHTTPServer()

	<-exitChan
}

func startHTTPServer() {
	router := gin.Default()

	router.POST("/rdapi/check-registered", func(c *gin.Context) {
		userKey := checkUserKeyInAuthHeader(c.Writer, c.Request)
		if userKey == "" {
			return
		}
		c.String(http.StatusOK, "Success")
	})

	router.POST("/rdapi/register", func(c *gin.Context) {
		userKey := utils.GenerateUserKey()
		if userKey == "" {
			c.String(http.StatusInternalServerError, "Invalid user key")
			return
		}

		userTime := onlineServers.Get(userKey)
		if !userTime.IsZero() {
			c.String(http.StatusInternalServerError, "User key already exists")
			return
		}

		id := utils.GenerateUserKey()
		user := model.User{
			ID:         id,
			Key:        userKey,
			Expiration: time.Now().Add(100 * 365 * 24 * time.Hour),
		}

		registeredUsers.Set(userKey, time.Now())

		log.Println("Registered user:", userKey)

		c.JSON(http.StatusOK, user)
	})

	router.POST("/rdapi/unregister", func(c *gin.Context) {
		userKey := checkUserKeyInAuthHeader(c.Writer, c.Request)
		if userKey == "" {
			return
		}

		registeredUsers.Remove(userKey)

		if isServerOnline(userKey) {
			err := deleteActiveUser(userKey)
			if err != nil {
				c.String(http.StatusInternalServerError, fmt.Sprintf("Error(s) stopping user: %v", err))
				return
			}
		}

		c.String(http.StatusOK, "Success")
	})

	router.POST("/rdapi/start", func(c *gin.Context) {
		userKey := checkUserKeyInAuthHeader(c.Writer, c.Request)
		if userKey == "" {
			return
		}

		skipFrpsStart := false
		userTime := onlineServers.Get(userKey)
		if !userTime.IsZero() {
			skipFrpsStart = true
		}

		if !skipFrpsStart {
			config, err := frps.ConfigureFrps(userKey)
			if err != nil {
				log.Println("Error creating frps files:", err)
				c.String(http.StatusInternalServerError, fmt.Sprintf("Error creating frps files: %v", err))
				return
			}
			log.Println("Created frps files for client key:", userKey)

			err = frps.StartFrps(&frpsCache, userKey)
			if err != nil {
				log.Println("Error starting frps:", err)
				c.String(http.StatusInternalServerError, fmt.Sprintf("Error starting frps: %v", err))
				return
			}
			log.Println("Started frps for client key:", userKey)

			frpsCache.SetFrpsConfig(userKey, config)

			err = nginx.AddNewServerToNginx(userKey, config.Ports.VHostPort, config.Ports.FileServerPort)
			if err != nil {
				log.Println("Error adding new server to nginx:", err)
				c.String(http.StatusInternalServerError, fmt.Sprintf("Error adding new server to nginx: %v", err))
				return
			}
			log.Println("Added new server to nginx for client key:", userKey)
		}

		onlineServers.Set(userKey, time.Now())
		if !skipFrpsStart {
			log.Println("Started frps for client key:", userKey)
		}

		response := struct {
			BindPort       string `json:"bindPort"`
			FileServerPort string `json:"fileServerPort"`
			Token          string `json:"token"`
		}{
			BindPort:       strconv.Itoa(frpsCache.GetFrpsConfig(userKey).Ports.BindPort),
			FileServerPort: strconv.Itoa(frpsCache.GetFrpsConfig(userKey).Ports.FileServerPort),
			Token:          frpsCache.GetFrpsConfig(userKey).AuthToken,
		}

		c.JSON(http.StatusOK, response)
	})

	router.POST("/rdapi/stop", func(c *gin.Context) {
		userKey := checkUserKeyInAuthHeader(c.Writer, c.Request)
		if userKey == "" {
			return
		}

		userTime := onlineServers.Get(userKey)
		if userTime.IsZero() {
			c.String(http.StatusNotFound, "User key not found")
			return
		}

		err := deleteActiveUser(userKey)

		if err != nil {
			c.String(http.StatusInternalServerError, fmt.Sprintf("Error(s) stopping user: %v", err))
			return
		}

		log.Println("Stopped server for client key:", userKey)

		c.String(http.StatusOK, "Success")
	})

	router.POST("/rdapi/report-client-activity", func(c *gin.Context) {
		userKey := checkUserKeyInAuthHeader(c.Writer, c.Request)
		if userKey == "" {
			return
		}

		activeClients.Set(userKey, time.Now())

		c.String(http.StatusOK, "Success")
	})

	router.GET("/rdapi/check-server-activity", func(c *gin.Context) {
		userKey := checkUserKeyInAuthHeader(c.Writer, c.Request)
		if userKey == "" {
			return
		}

		activeClients.Set(userKey, time.Now())

		userTime := onlineServers.Get(userKey)
		running := !userTime.IsZero()

		response := struct {
			Running        bool   `json:"running"`
			BindPort       string `json:"bindPort"`
			FileServerPort string `json:"fileServerPort"`
			Token          string `json:"token"`
		}{
			Running:        running,
			BindPort:       strconv.Itoa(frpsCache.GetFrpsConfig(userKey).Ports.BindPort),
			FileServerPort: strconv.Itoa(frpsCache.GetFrpsConfig(userKey).Ports.FileServerPort),
			Token:          frpsCache.GetFrpsConfig(userKey).AuthToken,
		}

		c.JSON(http.StatusOK, response)
	})

	router.GET("/rdapi/dashboard/:userKey/*path", func(c *gin.Context) {
		userKey := c.Param("userKey")
		//path := c.Param("path")
		if userKey == "" {
			c.String(http.StatusBadRequest, "Invalid user key")
			return
		}
		userTime := registeredUsers.Get(userKey)
		if userTime.IsZero() {
			c.String(http.StatusNotFound, "User key not registered")
			return
		}

		userRegistered := isUserRegistered(userKey)
		serverActive := isServerOnline(userKey)
		clientActive := isClientActive(userKey)

		data := model.StatusData{
			UserKey:        userKey,
			ServerActive:   serverActive,
			ClientActive:   clientActive,
			UserRegistered: userRegistered,
		}

		templates, err := template.ParseFiles("./static/dashboard.html", "./static/filemanager.html", "./static/status.html")
		if err != nil {
			c.String(http.StatusInternalServerError, "Server error")
			return
		}

		err = templates.ExecuteTemplate(c.Writer, "dashboard", data)
		if err != nil {
			log.Println("Error(s) executing template:", err)
			c.String(http.StatusInternalServerError, "Server error")
			return
		}
	})

	router.GET("/rdapi/status/:userKey", func(c *gin.Context) {
		userKey := c.Param("userKey")

		if userKey == "" {
			c.String(http.StatusBadRequest, "Invalid user key")
			return
		}
		// userTime := registeredUsers.Get(userKey)
		// if userTime.IsZero() {
		// 	c.String(http.StatusNotFound, "User key not registered")
		// 	return
		// }

		userRegistered := isUserRegistered(userKey)
		serverActive := isServerOnline(userKey)
		clientActive := isClientActive(userKey)

		data := model.StatusData{
			UserKey:        userKey,
			ServerActive:   serverActive,
			ClientActive:   clientActive,
			UserRegistered: userRegistered,
		}

		templates, err := template.ParseFiles("./static/status.html")
		if err != nil {
			c.String(http.StatusInternalServerError, "Server error")
			return
		}

		err = templates.ExecuteTemplate(c.Writer, "status", data)
		if err != nil {
			log.Println("Error(s) executing template:", err)
			c.String(http.StatusInternalServerError, "Server error")
			return
		}
	})

	router.GET("/rdapi/check-download-jobs", func(c *gin.Context) {
		userKey := checkUserKeyInAuthHeader(c.Writer, c.Request)
		if userKey == "" {
			return
		}

		jobs := downloadJobQueue.GetJobs(userKey)
		if jobs == nil {
			jobs = []*fileserver.DownloadJob{}
		}

		c.JSON(http.StatusOK, jobs)
	})

	type AddDownloadJobRequest struct {
		FilePath string `json:"filePath"`
		FileName string `json:"fileName"`
		Url      string `json:"url"`
	}

	router.POST("/rdapi/add-download-job", func(c *gin.Context) {
		userKey := checkUserKeyInAuthHeader(c.Writer, c.Request)
		if userKey == "" {
			return
		}

		var request AddDownloadJobRequest
		err := c.ShouldBindJSON(&request)
		if err != nil {
			c.String(http.StatusInternalServerError, "Error unmarshaling JSON request")
			return
		}

		existingJob := downloadJobQueue.JobAlreadyExists(userKey, request.FilePath, request.Url)
		if existingJob != nil {
			c.String(http.StatusConflict, "Job already exists")
			return
		}

		job := fileserver.DownloadJob{
			ID:        len(downloadJobQueue.GetJobs(userKey)) + 1,
			Status:    fileserver.PENDING,
			UserKey:   userKey,
			FileName:  request.FileName,
			FilePath:  request.FilePath,
			Url:       request.Url,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		downloadJobQueue.AddJob(userKey, job)

		c.String(http.StatusOK, "Success")
	})

	type ReportDownloadJobStatusRequest struct {
		JobID  int                          `json:"jobID"`
		Status fileserver.DownloadJobStatus `json:"status"`
	}

	router.POST("/rdapi/report-download-job-status", func(c *gin.Context) {
		userKey := checkUserKeyInAuthHeader(c.Writer, c.Request)
		if userKey == "" {
			return
		}

		var request ReportDownloadJobStatusRequest
		err := c.ShouldBindJSON(&request)
		if err != nil {
			c.String(http.StatusInternalServerError, "Error unmarshaling JSON request")
			return
		}

		existingJob := downloadJobQueue.GetJobByID(userKey, request.JobID)
		if existingJob == nil {
			c.String(http.StatusNotFound, "Job not found")
			return
		}

		existingJob.Status = request.Status
		existingJob.UpdatedAt = time.Now()

		println("Job status updated:", downloadJobQueue.GetJobByID(userKey, request.JobID).Status)

		c.String(http.StatusOK, "Success")
	})

	router.Static("/static", "./static")

	// Load HTML files
	router.LoadHTMLGlob("static/*.html")

	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	err := router.Run(":8080")
	if err != nil {
		log.Fatal(err)
	}
}

// check userKey in auth header and write error to provided writer if not found
func checkUserKeyInAuthHeader(w http.ResponseWriter, r *http.Request) string {
	userKey := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if userKey == "" {
		http.Error(w, "Invalid user key", http.StatusBadRequest)
		return ""
	}
	userTime := registeredUsers.Get(userKey)
	if userTime.IsZero() {
		http.Error(w, "User key not registered", http.StatusNotFound)
		return ""
	}
	return userKey
}

func isUserRegistered(key string) bool {
	userTime := registeredUsers.Get(key)
	return !userTime.IsZero()
}

func isServerOnline(key string) bool {
	userTime := onlineServers.Get(key)
	return !userTime.IsZero()
}

func isClientActive(key string) bool {
	userTime := activeClients.Get(key)
	return !userTime.IsZero() && time.Since(userTime).Seconds() < 60
}

func deleteActiveUser(key string) error {
	var combinedErr error

	err := nginx.RemoveServerFromNginx(key)
	if err != nil {
		combinedErr = errors.Join(combinedErr, err)
	}

	err = nginx.ReloadNginx()
	if err != nil {
		combinedErr = errors.Join(combinedErr, err)
	}

	err = frps.StopFrps(&frpsCache, key)
	if err != nil {
		combinedErr = errors.Join(combinedErr, err)
	}

	onlineServers.Remove(key)

	err = removeConfigFolder(key)
	if err != nil {
		combinedErr = errors.Join(combinedErr, err)
	}

	return combinedErr
}

func removeConfigFolder(key string) error {
	err := os.RemoveAll(fmt.Sprintf("./users/%v", key))
	if err != nil {
		log.Println("Error removing user folder:", err)
		return err
	}
	return nil
}

func generateTestUsersLogs() {
	for i := 0; i < 1; i++ {
		// generate random user ID and set to users cache
		userKey := utils.GenerateUserKey()
		onlineServers.Set(userKey, time.Now().Add(-2000*time.Hour))
		// create user folder
		err := os.Mkdir(fmt.Sprintf("./users/%v", userKey), 0755)
		if err != nil {
			log.Println("Error creating user folder:", err)
			return
		}
		// create user log file
		f, err := os.Create(fmt.Sprintf("./users/%v/frps.log", userKey))
		if err != nil {
			log.Println("Error creating user log file:", err)
			return
		}
		// write to user log file
		_, err = f.WriteString(fmt.Sprintf("2023/10/09 14:00:00 [I] [service.go:199] get new HTTP request host [%v.remotediffusion.com] path [/api/xxx] from [::1:12345]\n", userKey))
		if err != nil {
			log.Println("Error writing to user log file:", err)
			return
		}
		f.Close()
	}
}

func periodicCheck(exitChan chan struct{}) {
	wg.Add(1)
	defer wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			checkServerActivity()

		case <-exitChan:
			// Signal program termination
			return
		}
	}
}

func realoadCacheFromLogs() error {
	files, err := os.ReadDir("./users")
	if err != nil {
		log.Println("Error reading users folder:", err)
		return err
	}
	for _, file := range files {
		if file.IsDir() {
			userKey := file.Name()
			lastTime, err := logs.GetTimeOfTheLastRequestFromLogs(userKey)
			if err != nil {
				log.Println("Error getting time of the last request:", err)
				return err
			}
			onlineServers.Set(userKey, lastTime)
		}
	}
	return nil
}

func checkServerActivity() {
	var inactiveServers []string
	onlineServers.Range(func(key string, value time.Time) {
		isActive, err := isServerActive(key, value)
		if err != nil {
			log.Println("Error checking server activity:", err)
			return
		}
		if !isActive {
			inactiveServers = append(inactiveServers, key)
		}
	})

	for _, key := range inactiveServers {
		deleteActiveUser(key)
	}

}

func isServerActive(key string, lastTimeCached time.Time) (bool, error) {
	currentTime := time.Now()
	difference := currentTime.Sub(lastTimeCached)
	if difference.Minutes() > 60 {
		lastTime, err := logs.GetTimeOfTheLastRequestFromLogs(key)
		if err != nil {
			log.Println("Error getting time of the last request:", err)
			return false, err
		}
		lastTimeDiff := currentTime.Sub(lastTime)
		if lastTimeDiff.Minutes() > 60 {
			return false, nil
		}
	}
	return true, nil
}
