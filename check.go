package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/context"
)

type Checker struct {
	sync.Mutex
	Service  string
	Port     string
	Protocol string

	insName string
	ips     []string
	c       *Client

	stopCh chan int
	status Status
}

type Status struct {
	startLen int
	add      int
	status   map[string]string
}

func (s *Status) ShouldAdd() bool {
	if len(s.status) < s.startLen {
		return true
	}

	okCount := 0
	for _, ok := range s.status {
		if ok == "ok" || ok == "adding" {
			okCount++
		}
	}
	if okCount < s.startLen {
		return true
	}

	return false
}

func (c *Checker) Run() {
	c.stopCh = make(chan int)
	c.status.status = make(map[string]string)
	for _, ip := range c.ips {
		go c.loop(ip)
	}
}
func (c *Checker) loop(ip string) {
	log.Debug("add a checker")
	for {
		select {
		case <-time.Tick(2 * time.Second):
			c.Check(ip)
		case <-c.stopCh:
			return
		}
	}
}

func (c *Checker) Stop() {
	close(c.stopCh)
}

func (c *Checker) Check(ip string) {
	err := c.ping(ip)
	if err != nil {
		log.Warnf("%s/%s:%s not connected,error: %s", c.insName, c.Service, ip, err.Error())
		c.Lock()
		c.status.status[ip] = "down"
		c.Unlock()
		if !c.status.ShouldAdd() {
			log.Info("connected failed, but not need add.")
			return
		}
		c.Lock()
		c.status.status[ip] = "adding"
		c.Unlock()
		c.addContainer()
		return
	} else {
		c.Lock()
		c.status.status[ip] = "ok"
		c.Unlock()
	}

	return
}

func (c *Checker) addContainer() {
	if err := c.c.AddContainer(c.insName, c.Service); err != nil {
		log.Errorf("in checking, add container, error: %s", err.Error())
		return
	}
	c.Lock()
	c.status.add++
	c.Unlock()
	log.Infof("%s/%s add a container.", c.insName, c.Service)

	if newIPs, err := c.ReacquireIPs(); err != nil {
		log.Errorf("in checking, reacquire service %s/%s ips failed, error: %s", c.insName, c.Service, err.Error())
		return
	} else {
		for _, newip := range newIPs {
			exi := false
			for _, oldip := range c.ips {
				if oldip == newip {
					exi = true
					break
				}
			}
			if !exi {
				go c.loop(newip)
			}
		}
	}
}

func (c *Checker) ping(ip string) (err error) {
	switch c.Protocol {
	case "http", "https":
		err = checkHttp(c.Protocol, ip, c.Port)
	case "tcp":
		err = checkTCP(ip, c.Port)
	}
	return
}

func (c *Checker) ReacquireIPs() ([]string, error) {
	path := fmt.Sprintf("/lb/backends/%s-%s/ips/", c.insName, c.Service)
	resp, err := kapi.Get(context.Background(), path, nil)
	if err != nil {
		return nil, err
	}
	newIPs := []string{}
	for _, node := range resp.Node.Nodes {
		ip := strings.TrimPrefix(node.Key, path)
		newIPs = append(newIPs, ip)
	}
	return newIPs, nil
}

func checkHttp(scheme, ip, port string) error {
	dialConn, dialError := net.DialTimeout("tcp", ip+":"+port, time.Second*3)
	dial := func(_, _ string) (net.Conn, error) {
		return dialConn, dialError
	}

	client := &http.Client{
		Transport: &http.Transport{
			Dial: dial,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	url := scheme + "://" + ip + ":" + port
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("%s", err.Error())
	}

	_, err = client.Do(request)
	return err
}

func checkTCP(ip, port string) error {
	dialer := &net.Dialer{
		DualStack: true,
		Deadline:  time.Now().Add(time.Second * 5), // set default deadline
	}

	addr := ip + ":" + port
	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return err
	}
	conn.Close()

	return err
}
