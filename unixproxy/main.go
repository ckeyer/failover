package main

import (
	"io"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Sirupsen/logrus"
)

var (
	controllerAddr string
)

func init() {
	controllerAddr = os.Getenv("CONTROLLER_ADDR")
	if controllerAddr == "" {
		logrus.Errorf("env CONTROLLER_ADDR is reauired")
		os.Exit(1)
	}

	if strings.Contains(controllerAddr, "/") {
		logrus.Errorf("CONTROLLER_ADDR is host:port	")
		os.Exit(2)
	}

}

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	socket := "/var/run/csphere/failover.sock"
	os.MkdirAll("/var/run/csphere", 0755)

	if _, err := os.Stat(socket); err == nil {
		err = os.RemoveAll(socket)
		if err != nil {
			logrus.Errorf("rm socket file exista failes, error: %s", err.Error())
			return
		}
	} else {
		logrus.Debugf("open failed, error: %s", err.Error())
	}

	l, err := net.ListenUnix("unix", &net.UnixAddr{socket, "unix"})
	if err != nil {
		logrus.Errorf("create unix socker failed, error: %s", err.Error())
		return
	}
	defer l.Close()

	go func() {
		logrus.Info("start")
		for {
			tc, err := l.Accept()
			if err != nil {
				logrus.Debugf("connect error: %s", err.Error())
				tc.Close()
				continue
			}
			logrus.Info("connecting...")
			go proxy(tc)
		}
	}()

	allExit := make(chan os.Signal, 10)
	signal.Notify(allExit, os.Interrupt, os.Kill, syscall.SIGTERM)
	select {
	case <-allExit:
		os.RemoveAll(socket)
		logrus.Info("exit 0")
		os.Exit(0)
	}
}

func proxy(tc net.Conn) {
	logrus.Debugf("run serve")

	uc, err := net.Dial("tcp", controllerAddr)
	if err != nil {
		logrus.Errorf("get unix conn :%s", err.Error())
		uc.Close()
		return
	}
	logrus.Debugf("start copy")

	go io.Copy(tc, uc)
	go io.Copy(uc, tc)
}
