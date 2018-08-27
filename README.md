**Example**
```
go get github.com/gobwas/ws
go get github.com/gzatxm/websocket

serv := server.NewServer(9008, "/", 3600, false, "", "")

server.OnOpen = func(r *http.Request) (isOpen bool, code int) {
    isOpen = true
    code = 101
    return
}

serv.OnMessage = func(fd int64, data []byte) {
    fmt.Printf("%dCliend send:%s\n", fd, string(data))
    serv.Send(fd, data)
}

serv.OnClose = func(fd int64) {
    //deal close
}

serv.Run()

```
