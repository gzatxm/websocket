package websocket

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

type config struct {
	port     int
	path     string //routing path
	isWss    bool
	certFile string
	keyFile  string
	timeout  time.Duration //Disconnected from the client after n seconds of disconnection
}

type logInfo struct {
	startime time.Time
	run      time.Duration
	sum      int64
	current  int64
	close    int64
}

type server struct {
	config config
	conns  map[int64]net.Conn
	fd     int64
	log    logInfo

	OnOpen    func(r *http.Request) (isopen bool, code int)
	OnMessage func(fd int64, data []byte)
	OnClose   func(fd int64)
}

func NewServer(port int, path string, timeout time.Duration, isWss bool, certFile, keyFile string) *server {
	s := new(server)
	conf := config{port, path, isWss, certFile, keyFile, timeout}
	s.config = conf
	s.conns = make(map[int64]net.Conn)
	return s
}

func (s *server) Run() {
	http.HandleFunc(s.config.path, s.connect)
	port := strconv.Itoa(s.config.port)
	s.log.startime = time.Now()
	fmt.Printf("websocket run on %s\n", port)
	if s.config.isWss {
		http.ListenAndServeTLS(":"+port, s.config.certFile, s.config.keyFile, nil) //wss
	} else {
		http.ListenAndServe(":"+port, nil)
	}
}

func (s *server) GetLog() string {
	s.log.run = time.Now().Sub(s.log.startime)
	return fmt.Sprintf("Start Time:%s\nRunning time:%s\nConnections:%d\nCurrent connections:%d\nClose connections:%d\n", s.log.startime, s.log.run, s.log.sum, s.log.current, s.log.close)
}

func (s *server) Send(fd int64, data []byte) error {
	return wsutil.WriteServerMessage(s.conns[fd], 1, data)
}

func (s *server) connect(w http.ResponseWriter, r *http.Request) {
	if s.OnOpen != nil {
		isOpen, code := s.OnOpen(r)
		if isOpen == false {
			w.WriteHeader(code)
			return
		}
	}

	conn, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		log.Print("UpgradeHTTP:", err)
		return
	}

	fd := s.createFd()
	s.conns[fd] = conn
	s.log.sum++
	s.log.current++
	go s.receive(fd, conn)
}

func (s *server) receive(fd int64, conn net.Conn) {
	timeout := s.config.timeout * time.Second
	t := time.NewTimer(timeout)
	ch := make(chan []byte)

	go func() {
		for {
			data, _, err := wsutil.ReadClientData(conn)
			if err != nil {
				fmt.Println("ReadClientData:", err)
				s.close(fd)
				break
			}

			if s.OnMessage != nil {
				s.OnMessage(fd, data)
			}
			ch <- data
		}
	}()

	for {
		select {
		case <-ch:
			t.Reset(timeout)
		case <-t.C:
			s.close(fd)
			fmt.Println("Time out")
			break
		}
	}
}

func (s *server) close(fd int64) {
	if s.conns[fd] == nil {
		return
	}
	s.conns[fd].Close()
	s.log.current--
	s.log.close++
	delete(s.conns, fd)
	if s.OnClose != nil {
		s.OnClose(fd)
	}
}

func (s *server) createFd() int64 {
	s.fd++
	return s.fd
}
