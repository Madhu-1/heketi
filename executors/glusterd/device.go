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

func (g *Gluster) DeviceSetup(host, device, vgid string, destroy bool) (d *executors.DeviceInfo, e error) {

	return nil, executors.NotSupportedError
}

func (g *Gluster) getVgSizeFromNode(
	d *executors.DeviceInfo,
	host, device, vgid string) error {

	return executors.NotSupportedError
}
