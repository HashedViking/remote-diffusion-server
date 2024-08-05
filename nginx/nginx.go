package nginx

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

const nginxUserConfigDirectoryPath = "/etc/nginx/conf.d/rd-users/"

func AddNewServerToNginx(key string, port int, fileServerPort int) error {
	newServerDirective := fmt.Sprintf(`
server {
	listen 443 ssl;
	server_name %s.remotediffusion.com;
	include /root/remote-diffusion/remotediffusion.ssl.conf; 
	location / {
		proxy_redirect off;
		proxy_http_version 1.1;
		proxy_pass http://127.0.0.1:%d/;
		proxy_set_header Host $host;
		proxy_set_header Upgrade $http_upgrade;
		proxy_set_header Connection "upgrade";
		proxy_set_header X-Real-IP $remote_addr;
		proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
		proxy_set_header X-Forwarded-Proto $scheme;
	}
	location /files {
		proxy_redirect off;
		proxy_http_version 1.1;
		proxy_pass http://127.0.0.1:%d/files;
		proxy_set_header Host $host;
		proxy_set_header Upgrade $http_upgrade;
		proxy_set_header Connection "upgrade";
		proxy_set_header X-Real-IP $remote_addr;
		proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
		proxy_set_header X-Forwarded-Proto $scheme;
	}
	location /dashboard {
		proxy_redirect off;
		proxy_http_version 1.1;
		proxy_pass http://127.0.0.1:8080/rdapi/dashboard/%s;
		proxy_set_header Host $host;
		proxy_set_header Upgrade $http_upgrade;
		proxy_set_header Connection "upgrade";
		proxy_set_header X-Real-IP $remote_addr;
		proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
		proxy_set_header X-Forwarded-Proto $scheme;
	}
	location /rdapi/ {
		proxy_redirect off;
		proxy_http_version 1.1;
		proxy_pass http://127.0.0.1:8080/rdapi/;
		proxy_set_header Host $host;
		proxy_set_header Upgrade $http_upgrade;
		proxy_set_header Connection "upgrade";
		proxy_set_header X-Real-IP $remote_addr;
		proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
		proxy_set_header X-Forwarded-Proto $scheme;
	}
}`, key, port, fileServerPort, key)

	err := createUserNginxConfig(key, newServerDirective)
	if err != nil {
		log.Println("modifyNginxConfig: Error creating user nginx config:", err)
		return err
	}

	err = ReloadNginx()
	if err != nil {
		log.Println("modifyNginxConfig: Error reloading nginx:", err)
		return err
	}
	log.Println("Added new server to nginx")
	return nil
}

func ReloadNginx() error {
	// cmd := exec.Command("sudo", "/usr/sbin/service", "nginx", "reload")
	cmd := exec.Command("sh", "reload_nginx.sh")
	err := cmd.Run()
	if err != nil {
		return err
	}
	log.Println("Successfully reloaded nginx")
	return nil
}

func RemoveServerFromNginx(key string) error {
	filePath := filepath.Join(nginxUserConfigDirectoryPath, key+".conf")
	err := os.Remove(filePath)
	if err != nil {
		return fmt.Errorf("removeServerFromNginx: error removing file: %v", err)
	}
	return nil
}

func createUserNginxConfig(userKey string, configContent string) error {
	filePath := filepath.Join(nginxUserConfigDirectoryPath, userKey+".conf")
	err := os.WriteFile(filePath, []byte(configContent), 0744)
	if err != nil {
		return fmt.Errorf("createUserNginxConfig: error writing to file: %v", err)
	}
	return nil
}
