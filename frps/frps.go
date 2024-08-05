package frps

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	queue "remote-diffusion-server/queue"
	utils "remote-diffusion-server/utils"
	"sync"
	"syscall"
	"time"
)

type FrpsCache struct {
	sync.RWMutex
	userPid        map[string]int
	pidTime        map[int]time.Time
	userFrpsConfig map[string]FrpsConfig
}

func NewFrpsCache() FrpsCache {
	return FrpsCache{
		userPid:        make(map[string]int),
		pidTime:        make(map[int]time.Time),
		userFrpsConfig: make(map[string]FrpsConfig),
	}
}

func (cache *FrpsCache) GetFrpsConfig(key string) FrpsConfig {
	cache.RLock()
	defer cache.RUnlock()

	return cache.userFrpsConfig[key]
}

func (cache *FrpsCache) SetFrpsConfig(key string, config FrpsConfig) {
	cache.Lock()
	defer cache.Unlock()

	cache.userFrpsConfig[key] = config
}

func (cache *FrpsCache) deletePid(key string, pid int) {
	cache.Lock()
	defer cache.Unlock()

	delete(cache.pidTime, pid)
	delete(cache.userPid, key)
}

func (cache *FrpsCache) putBackPorts(key string) {
	cache.Lock()
	defer cache.Unlock()

	config := cache.userFrpsConfig[key]
	bindPortsQueue.PutBack(config.Ports.BindPort)
	vhostPortsQueue.PutBack(config.Ports.VHostPort)
	fileServerPortsQueue.PutBack(config.Ports.FileServerPort)

	delete(cache.userFrpsConfig, key)
}

func (cache *FrpsCache) savePid(pid int, key string) {
	cache.Lock()
	defer cache.Unlock()

	cache.pidTime[pid] = time.Now()
	cache.userPid[key] = pid
}

type FrpsConfig struct {
	Ports       FrpsPorts
	AuthToken   string
	LogFilePath string
}

type FrpsPorts struct {
	BindPort       int
	VHostPort      int
	FileServerPort int
}

var bindPortsQueue = queue.NewPortQueue(10001, 10100)
var vhostPortsQueue = queue.NewPortQueue(20001, 20100)
var fileServerPortsQueue = queue.NewPortQueue(30001, 30100)

func ConfigureFrps(userKey string) (FrpsConfig, error) {
	userFolderPath := filepath.Join(".", "users", userKey)
	err := os.MkdirAll(userFolderPath, 0755)
	if err != nil {
		return FrpsConfig{}, fmt.Errorf("error creating user folder: %w", err)
	}

	logFilePath := filepath.Join(userFolderPath, "frps.log")
	_, err = os.Create(logFilePath)
	if err != nil {
		return FrpsConfig{}, fmt.Errorf("error creating user log file: %w", err)
	}

	configFilePath := filepath.Join(userFolderPath, "frps.toml")
	toml, err := os.Create(configFilePath)
	if err != nil {
		return FrpsConfig{}, fmt.Errorf("error creating user config file: %w", err)
	}
	defer toml.Close()

	freePorts, err := getFreePorts()
	if err != nil {
		return FrpsConfig{}, fmt.Errorf("error getting free port: %w", err)
	}

	bindPort := freePorts.BindPort
	vHostPort := freePorts.VHostPort

	fmt.Fprintf(toml, "bindPort = %v\n", bindPort)
	fmt.Fprintf(toml, "vhostHTTPPort = %v\n", vHostPort)
	fmt.Fprintf(toml, "auth.method = \"token\"\n")
	authToken := utils.GenerateUserKey()
	fmt.Fprintf(toml, "auth.token = \"%v\"\n", authToken)
	fmt.Fprintf(toml, "log.to = \"./users/%v/frps.log\"\n", userKey)
	// important to set log level to debug to determine user activity and stop frps if no activity
	fmt.Fprintf(toml, "log.level = \"debug\"")

	return FrpsConfig{
		Ports:       freePorts,
		AuthToken:   authToken,
		LogFilePath: logFilePath,
	}, nil
}

func getFreePorts() (FrpsPorts, error) {
	var ports FrpsPorts
	var err error

	ports.BindPort, err = bindPortsQueue.Pop()
	if err != nil {
		return FrpsPorts{}, err
	}

	ports.VHostPort, err = vhostPortsQueue.Pop()
	if err != nil {
		bindPortsQueue.PutBack(ports.BindPort)
		return FrpsPorts{}, err
	}

	ports.FileServerPort, err = fileServerPortsQueue.Pop()
	if err != nil {
		bindPortsQueue.PutBack(ports.BindPort)
		vhostPortsQueue.PutBack(ports.VHostPort)
		return FrpsPorts{}, err
	}

	return ports, nil
}
func StartFrps(cache *FrpsCache, key string) error {
	cmd := exec.Command("./frps/frps", "-c", fmt.Sprintf("./users/%v/frps.toml", key))

	f, err := os.Open(fmt.Sprintf("./users/%v/frps.log", key))
	if err != nil {
		log.Println("Error opening frps log file:", err)
		return err
	}
	defer f.Close()
	cmd.Stdout = f
	cmd.Stderr = f

	if err := cmd.Start(); err != nil {
		return err
	}

	cache.savePid(cmd.Process.Pid, key)

	return nil
}

func StopFrps(cache *FrpsCache, key string) error {
	// Get the PID from the cache
	cache.RLock()
	pid, ok := cache.userPid[key]
	cache.RUnlock()

	if !ok {
		return fmt.Errorf("stopFrps: unknown user key: %s", key)
	}

	// Check if the process is still running
	process, err := os.FindProcess(pid)
	if err != nil {
		cache.deletePid(key, pid)
		return nil
	}

	// Kill the process
	err = process.Signal(syscall.SIGTERM)
	if err != nil {
		return err
	}

	cache.deletePid(key, pid)
	cache.putBackPorts(key)

	log.Println("Ports count:", bindPortsQueue.Length(), vhostPortsQueue.Length(), fileServerPortsQueue.Length())

	return nil
}
