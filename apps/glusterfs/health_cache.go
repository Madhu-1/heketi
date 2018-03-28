//
// Copyright (c) 2018 The heketi Authors
//
// This file is licensed to you under your choice of the GNU Lesser
// General Public License, version 3 or any later version (LGPLv3 or
// later), or the GNU General Public License, version 2 (GPLv2), in all
// cases as published by the Free Software Foundation.
//

package glusterfs

import (
	_"fmt"
	"strings"
	"sync"
	"time"

	"github.com/heketi/heketi/executors"
	wdb "github.com/heketi/heketi/pkg/db"

	"github.com/boltdb/bolt"
)

var (
	healthNow func() time.Time = time.Now
)

type NodeHealthStatus struct {
	NodeId     string
	Host       string
	Up         bool
	LastUpdate time.Time
}

type NodeHealthCache struct {
	// tunables
	CheckInterval time.Duration

	db    wdb.RODB
	exec  executors.Executor
	nodes map[string]*NodeHealthStatus
	lock  sync.Mutex
}

func NewNodeHealthCache(db wdb.RODB, e executors.Executor) *NodeHealthCache {
	return &NodeHealthCache{
		db:            db,
		exec:          e,
		nodes:         map[string](*NodeHealthStatus){},
		CheckInterval: time.Second * 5,
	}
}

func (hc *NodeHealthCache) Status() map[string]bool {
	hc.lock.Lock()
	defer hc.lock.Unlock()
	healthy := map[string]bool{}
	for k, v := range hc.nodes {
		healthy[k] = v.Up
	}
	return healthy
}

func (hc *NodeHealthCache) Refresh() error {
	logger.Info("Starting Node Health Status refresh")
	sl, err := hc.toProbe()
	if err != nil {
		return err
	}
	for _, s := range sl {
		hc.updateNode(s)
	}
	hc.cleanOld()
	return nil
}

func (hc *NodeHealthCache) updateNode(s *NodeHealthStatus) {
	hc.lock.Lock()
	defer hc.lock.Unlock()
	if prev, found := hc.nodes[s.NodeId]; found {
		s = prev
	} else {
		hc.nodes[s.NodeId] = s
	}
	s.update(hc.exec)
}

func (hc *NodeHealthCache) cleanOld() {
	hc.lock.Lock()
	defer hc.lock.Unlock()
	// purge any items that are stale
	cleaned := 0
	for k, v := range hc.nodes {
		if v.old() {
			delete(hc.nodes, k)
			cleaned++
		}
	}
	logger.Info("Cleaned %v nodes from health cache", cleaned)
}

func (hc *NodeHealthCache) Monitor() chan<- interface{} {
	ticker := time.NewTicker(hc.CheckInterval)
	stop := make(chan interface{})

	go func() {
		logger.Info("Started Node Health Cache Monitor")
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				logger.Info("Stopping Node Health Cache Monitor")
				return
			case <-ticker.C:
				err := hc.Refresh()
				if err != nil {
					logger.LogError("Node Heath Cache Monitor: %v", err.Error())
				}
			}
		}
	}()

	return stop
}

func (hc *NodeHealthCache) toProbe() ([]*NodeHealthStatus, error) {
	probeNodes := []*NodeHealthStatus{}
	err := hc.db.View(func(tx *bolt.Tx) error {
		n, err := NodeList(tx)
		if err != nil {
			return err
		}
		for _, nodeId := range n {
			if (strings.HasPrefix(nodeId, "MANAGE") ||
				strings.HasPrefix(nodeId, "STORAGE")) {
				continue
			}
			node, err := NewNodeEntryFromId(tx, nodeId)
			if err != nil {
				return err
			}
			nhs := &NodeHealthStatus{
				NodeId: nodeId,
				Host:   node.Info.Hostnames.Manage[0],
			}
			probeNodes = append(probeNodes, nhs)
		}
		return nil
	})
	return probeNodes, err
}

func (s *NodeHealthStatus) update(e executors.Executor) {
	// TODO: add ability to skip check if node was already recently checked
	err := e.GlusterdCheck(s.Host)
	s.Up = (err == nil)
	s.LastUpdate = time.Now()
	logger.Info("Periodic health check status: node %v up=%v",
		s.NodeId, s.Up)
}

func (s *NodeHealthStatus) old() bool {
	// TODO: math to say if this cache entry is stale
	return false
}
