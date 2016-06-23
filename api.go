package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Node struct {
	ID       string              `json:"id"`
	Hostname string              `json:"hostname"`
	IPs      map[string][]string `json:"ips"`
	Status   string              `json:"status"`
}

type Jar struct {
	cookies []*http.Cookie
}

func (jar *Jar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	jar.cookies = cookies
}

func (jar *Jar) Cookies(u *url.URL) []*http.Cookie {
	return jar.cookies
}

type Client struct {
	*http.Client
	controllerURL string
	apiKey        string
}

func NewClient(controllerUrl, apikey string) *Client {
	socket := "/var/run/csphere/failover.sock"
	// socket := "/Users/ckeyer/tmp/csphere-failover.sock"
	unixDial := func(proto, addr string) (net.Conn, error) {
		return net.DialTimeout("unix", socket, time.Second*3)
	}
	unic := &http.Client{
		Transport: &http.Transport{
			Dial: unixDial,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	client := &Client{
		controllerURL: strings.TrimSuffix(controllerUrl, "/") + "/api/",
		apiKey:        apikey,
		Client:        unic,
	}

	client.Jar = new(Jar)

	if err := client.check(); err != nil {
		log.Fatalf("can not connect controller %s, error: %s", controllerUrl, err.Error())
	}

	return client
}

func (c *Client) DoReq(method, url string, data io.Reader) (resp *http.Response, err error) {
	req, _ := http.NewRequest(method, url, data)
	req.Header.Add("Csphere-Api-Key", c.apiKey)
	return c.Client.Do(req)
}

func (c *Client) url(s ...string) string {
	return c.controllerURL + strings.Join(s, "/")
}

func (c *Client) check() error {
	res, err := c.DoReq("GET", c.url("_ping"), nil)
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		return fmt.Errorf("ApiKey failed")
	}
	return err
}

func (c *Client) GetNodes() ([]Node, error) {
	resp, err := c.DoReq("GET", c.url("nodes?status=normal"), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var ret []Node
	err = json.NewDecoder(resp.Body).Decode(&ret)
	if err != nil {
		return nil, fmt.Errorf("decode nodes failed, error: %s", err.Error())
	}

	return ret, nil
}

func (c *Client) GetAgentNodes() ([]Node, error) {
	nodes, err := c.GetNodes()
	if err != nil {
		return nil, err
	}

	resp, err := c.DoReq("GET", c.url("svrpools"), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var ret []struct {
		Name  string   `json:"name"`
		Nodes []string `json:"nodes"`
		Nat   bool     `json:"nat"`
	}
	err = json.NewDecoder(resp.Body).Decode(&ret)
	if err != nil {
		return nil, fmt.Errorf("decode svrpool nodes failed, error: %s", err.Error())
	}
	retNodes := []Node{}
	for _, svrp := range ret {
		if svrp.Name == "cSphere系统" {
			continue
		}
		if svrp.Nat {
			continue
		}
		for _, nid := range svrp.Nodes {
			for _, node := range nodes {
				if node.ID == nid {
					retNodes = append(retNodes, node)
				}
			}
		}
	}
	return retNodes, nil
}
func (c *Client) AddContainer(insName, serName string) error {
	resp, err := c.DoReq("PATCH", c.url("instances", insName, serName, "addcontainer?sync=true"), nil)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		bs, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("add container failse, error; %s", bs)
	}
	return nil
}
