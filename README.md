<p align="center">
  <a href="" rel="noopener">
 <img width=50px height=50px src="https://remotediffusion.com/static/images/logo.png" alt="Project logo"></a>
</p>

<h3 align="center">Remote Diffusion Server</h3>

<div align="center">

[![Status](https://img.shields.io/badge/status-active-success.svg)]()
![GitHub License](https://img.shields.io/github/license/HashedViking/remote-diffusion-server)
![Linux](https://img.shields.io/badge/avaliable_on-Linux-blue)

</div>

---

<p align="center"> 
    Backend for <a href="https://remotediffusion.com">Remote Diffusion</a>
    <br> 
</p>

## 📖 Table of Contents

- [📖 Table of Contents](#-table-of-contents)
- [🧐 About ](#-about-)
- [🏁 Getting Started ](#-getting-started-)
- [🚀 Deployment ](#-deployment-)
- [📝 TODO ](#-todo-)
- [⛏️ Built Using ](#️-built-using-)
- [✍️ Authors ](#️-authors-)

## 🧐 About <a name = "about"></a>

Remote Diffusion backend server. This server manages multiple connections. If you need just a one-click tunnel to your local [Stable Diffusion Web UI](https://github.com/AUTOMATIC1111/stable-diffusion-webui) check out the [Client](https://github.com/HashedViking/remote-diffusion-client).

## 🏁 Getting Started <a name = "getting_started"></a>
Tested on Ubuntu 22.04

Update apt
```
sudo apt update
sudo apt upgrade
```

Install go
```
sudo apt install golang-go
```

Install Nginx
```
sudo apt install nginx
```

Install PostgreSQL
```
sudo apt install postgresql
sudo systemctl start postgresql
sudo systemctl status postgresql
```

Download the backend
```
git clone https://github.com/HashedViking/remote-diffusion-server
```

Build
```
cd remote-diffusion-server
go build remote-diffusion-server
```
Run
```
./remote-diffusion-server
```

## 🚀 Deployment <a name = "deployment"></a>

Setup Nginx reloading script
```
sudo chmod -R 755 reload_nginx.sh
```

Run the binary as background job and collect logs.
```
nohup ./remote-diffusion-server > output.log &
```

Find the running process
```
lsof -i :8080 | grep remote
```

## 📝 TODO <a name = "todo"></a>

- [ ] Add Nginx, Certbot, PostgreSQL configuration steps.
- [ ] Explain in-depth how the system works.
- [ ] Dockerize.
- [ ] Add Monitoring.
- [ ] Stream SDWebUI console output.

## ⛏️ Built Using <a name = "built_using"></a>

- [PostgreSQL](https://www.postgresql.org/) - Database
- [Gin](https://github.com/gin-gonic/gin) - Server Framework
- [Frp](https://github.com/fatedier/frp) - Tunnel
- [Nginx](https://nodejs.org/en/) - Server Reverse Proxy

## ✍️ Authors <a name = "authors"></a>

- [@hashedviking](https://github.com/HashedViking) - Idea & Initial work
