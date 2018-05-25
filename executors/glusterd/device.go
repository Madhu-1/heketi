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
	"encoding/json"
	"errors"

	"github.com/heketi/heketi/executors"
)

const (
	VGDISPLAY_SIZE_KB                  = 11
	VGDISPLAY_PHYSICAL_EXTENT_SIZE     = 12
	VGDISPLAY_TOTAL_NUMBER_EXTENTS     = 13
	VGDISPLAY_ALLOCATED_NUMBER_EXTENTS = 14
	VGDISPLAY_FREE_NUMBER_EXTENTS      = 15
)

// Read:
// https://access.redhat.com/documentation/en-US/Red_Hat_Storage/3.1/html/Administration_Guide/Brick_Configuration.html
//
//
// Copyright (c) 2015 The heketi Authors
//
// This file is licensed to you under your choice of the GNU Lesser
// General Public License, version 3 or any later version (LGPLv3 or
// later), or the GNU General Public License, version 2 (GPLv2), in all
// cases as published by the Free Software Foundation.
//

// Read:
// https://access.redhat.com/documentation/en-US/Red_Hat_Storage/3.1/html/Administration_Guide/Brick_Configuration.html
//

// deviceInfo represents structure in which devices are to be store in Peer Metadata
type deviceInfo struct {
	Name          string `json:"name"`
	State         string `json:"state"`
	VgName        string `json:"vg-name"`
	AvailableSize uint64 `json:"available-size"`
	ExtentSize    uint64 `json:"extent-size"`
	Used          bool   `json:"used"`
}

func (g *GlusterdExecutor) DeviceSetup(host, device, vgid string, destroy bool) (*executors.DeviceInfo, error) {
	g.createClient(host)
	peerid := ""
	peerlist, err := g.Client.Peers()
	if err != nil {
		logger.Err(err)
		return nil, err
	}
	for _, peer := range peerlist {
		for _, addr := range peer.PeerAddresses {
			if addr == host+g.Config.ClientPORT {
				peerid = peer.ID.String()
			}
		}
	}
	//TODO implement device delete in failed scenario?
	dev, err := g.Client.DeviceAdd(peerid, device)
	if err != nil {
		logger.Err(err)
		return nil, err
	}
	var devices []deviceInfo
	var info = new(executors.DeviceInfo)
	if _, exist := dev.Metadata["_devices"]; exist {

		err = json.Unmarshal([]byte(dev.Metadata["_devices"]), &devices)
		if err != nil {
			logger.Err(err)
			return nil, err
		}
		for _, d := range devices {
			if device == d.Name {
				info.ExtentSize = d.ExtentSize
				info.Size = d.AvailableSize
			}
		}
	}
	if info == nil {
		logger.LogError("failed to fetch device details")
		return nil, errors.New("failed to fetch device details")
	}
	return info, nil
}

func (g *GlusterdExecutor) GetDeviceInfo(host, device, vgid string) (d *executors.DeviceInfo, e error) {
	//TODO need to replace this function by listing device
	d = &executors.DeviceInfo{}

	g.createClient(host)

	peerlist, err := g.Client.Peers()
	if err != nil {
		logger.Err(err)
		return nil, err
	}
	var devices []deviceInfo
	var info = new(executors.DeviceInfo)
	for _, peer := range peerlist {
		for _, addr := range peer.PeerAddresses {
			if addr == host+g.Config.ClientPORT {
				if _, exist := peer.Metadata["_devices"]; exist {

					err = json.Unmarshal([]byte(peer.Metadata["_devices"]), &devices)
					if err != nil {
						logger.Err(err)
						return nil, err
					}
					for _, d := range devices {
						if device == d.Name {
							info.ExtentSize = d.ExtentSize
							info.Size = d.AvailableSize
							break
						}
					}
				}
			}

		}
	}

	if info == nil {
		logger.LogError("failed to fetch device details")
		return nil, errors.New("failed to fetch device details")
	}
	return info, nil
}

func (g *GlusterdExecutor) DeviceTeardown(host, device, vgid string) error {
	//TODO need to implement this api
	return executors.NotSupportedError
}
