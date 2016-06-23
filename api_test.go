package main

import (
	"io/ioutil"
	"testing"
)

var (
	c = NewClient("http://docker.nicescale.com:1016", "f5a8964a8ad2899b080886cc6e03cc3e42e7e9b5")
)

func TestGetNode(t *testing.T) {
	return
	nids, err := c.GetAgentNodes()
	if err != nil {
		t.Fatalf("get nodt failed, error %s", err.Error())
	}
	t.Errorf("Body %+v", nids)
}

func TestAddContainer(t *testing.T) {
	insName, serName := "ckkkk", "busybox"
	resp, err := c.DoReq("PATCH", c.url("instances", insName, serName, "addcontainer?sync=true"), nil)
	if err != nil {
		t.Fatalf("get nodt failed, error %s", err.Error())
	}
	bs, _ := ioutil.ReadAll(resp.Body)
	t.Errorf("Body %s", bs)
	t.Errorf("status: %s", resp.Status)
}
