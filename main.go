package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/coreos/etcd/client"
	"golang.org/x/net/context"
)

var (
	log        = logrus.New()
	config     *Config
	kapi       client.KeysAPI
	controller *Client

	exitAll = make(chan int)
)

func init() {
	log.Level = logrus.DebugLevel

	configFile := flag.String("c", "config.json", "config file path")
	flag.Parse()

	config = LoadConfig(*configFile)
	controller = NewClient(config.ControllerURL, config.ApiKey)

	agentNodes, err := controller.GetAgentNodes()
	if err != nil {
		log.Fatalf("can not get agent, error: %s", err.Error())
	}
	etcIPs := []string{}
	for _, node := range agentNodes {
		for ifc, ips := range node.IPs {
			if !strings.HasPrefix(ifc, "docker") {
				etcIPs = append(etcIPs, ips...)
			}
		}
	}
	kapi = ConnectKAPI(etcIPs)
}

func ConnectKAPI(ips []string) client.KeysAPI {
	endpoints := strings.Split("http://"+strings.Join(ips, ":2379/,http://")+":2379", ",")
	log.Debugf("%+v", endpoints)
	cfg := client.Config{
		Endpoints:               endpoints,
		Transport:               client.DefaultTransport,
		HeaderTimeoutPerRequest: time.Second,
	}
	c, err := client.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	return client.NewKeysAPI(c)
}

func LoadCheckList() error {
	for inst, checkers := range config.Monitor {
		for _, checker := range checkers {
			path := fmt.Sprintf("/lb/backends/%s-%s/ips/", inst, checker.Service)
			resp, err := kapi.Get(context.Background(), path, nil)
			if err != nil {
				return fmt.Errorf("can not get %s/%s ips, error: %s", inst, checker.Service, err.Error())
			}
			checker.ips = []string{}
			checker.insName = inst
			checker.c = controller
			for _, node := range resp.Node.Nodes {
				ip := strings.TrimPrefix(node.Key, path)
				checker.ips = append(checker.ips, ip)
			}
			checker.status.startLen = len(checker.ips)
		}
	}
	return nil
}

func main() {
	log.Info("start")
	err := LoadCheckList()
	if err != nil {
		log.Fatal(err.Error())
	}

	for _, servs := range config.Monitor {
		for _, c := range servs {
			go c.Run()
		}
	}

	allExit := make(chan os.Signal, 10)
	signal.Notify(allExit, os.Interrupt, os.Kill, syscall.SIGTERM)
	select {
	case <-allExit:
		log.Info("exit 0")
		os.Exit(0)
	}
}
