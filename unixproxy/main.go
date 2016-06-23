package main

import (
	"io"
	"io/ioutil"
	"net"

	"github.com/Sirupsen/logrus"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	socket := "/var/run/csphere-failover.sock"
	// socket := "/Users/ckeyer/tmp/csphere-failover.sock"

	l, err := net.ListenUnix("unix", &net.UnixAddr{socket, "unix"})
	if err != nil {
		logrus.Errorf("create unix socker failed, error: %s", err.Error())
		return
	}
	defer l.Close()

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
}

func proxy(tc net.Conn) {
	// defer tc.Close()
	logrus.Debugf("run serve")

	uc, err := net.Dial("tcp", "docker.nicescale.com:1016")
	if err != nil {
		logrus.Errorf("get unix conn :%s", err.Error())
		uc.Close()
		return
	}
	logrus.Debugf("start copy")

	go io.Copy(tc, uc)
	go io.Copy(uc, tc)
}

func debugCopy(dst io.Writer, src io.Reader) {
	io.Copy(dst, src)
	logrus.Debug("...")
	return
	bs, _ := ioutil.ReadAll(src)
	logrus.Debugf("read: %s", bs)
	dst.Write(bs)
}
