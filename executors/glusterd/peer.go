//
// Copyright (c) 2015 The heketi Authors
//
// This file is licensed to you under your choice of the GNU Lesser
// General Public License, version 3 or any later version (LGPLv3 or
// later), or the GNU General Public License, version 2 (GPLv2), in all
// cases as published by the Free Software Foundation.
//

package glusterd

import (
	"errors"

	"github.com/lpabon/godbc"
)

//TODO need to hanlde glusterd2 port also
//currently its hardcoded

var (
	port = ":24008"
)

// :TODO: Rename this function to NodeInit or something
func (g *Gluster) PeerProbe(host, newnode string) error {

	godbc.Require(host != "")
	godbc.Require(newnode != "")
	g.createClient(host)
	logger.Info("Probing: %v -> %v", host, newnode)
	// create the commands
	_, err := g.Client.PeerAdd(newnode)
	if err != nil {
		return err
	}

	//TODO need to check this

	// Determine if there is a snapshot limit configuration setting
	// if s.RemoteExecutor.SnapShotLimit() > 0 {
	// 	logger.Info("Setting snapshot limit")
	// 	commands = []string{
	// 		fmt.Sprintf("gluster --mode=script snapshot config snap-max-hard-limit %v",
	// 			s.RemoteExecutor.SnapShotLimit()),
	// 	}
	// 	_, err := s.RemoteExecutor.RemoteCommandExecute(host, commands, 10)
	// 	if err != nil {
	// 		return err
	// 	}
	// }

	return nil
}

func (g *Gluster) PeerDetach(host, detachnode string) error {
	godbc.Require(host != "")
	godbc.Require(detachnode != "")
	g.createClient(host)

	var peerid string
	// create the commands
	logger.Info("Detaching node %v", detachnode)
	//list nodes in cluster
	peerlist, err := g.Client.Peers()
	if err != nil {
		logger.Err(err)
		return err
	}
	for _, peer := range peerlist {
		for _, addr := range peer.PeerAddresses {
			if addr == detachnode+port {
				peerid = peer.ID.String()
			}
		}
	}
	if peerid == "" {
		logger.LogError("not able to find peer info for %s", detachnode)
		return errors.New("unable to detatch peer")
	}
	err = g.Client.PeerRemove(peerid)
	if err != nil {
		logger.Err(err)
	}

	return nil
}

func (g *Gluster) GlusterdCheck(host string) error {
	godbc.Require(host != "")

	logger.Info("Check Glusterd service status in node %v", host)
	g.createClient(host)
	//TODO change this to health check URL
	_, err := g.Client.Peers()
	if err != nil {
		logger.Err(err)
		return err
	}

	return nil
}
