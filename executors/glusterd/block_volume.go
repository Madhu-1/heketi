// Copyright (c) 2017 The heketi Authors
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

func (g *Gluster) BlockVolumeCreate(host string,
	volume *executors.BlockVolumeRequest) (*executors.BlockVolumeInfo, error) {

	return nil, executors.NotSupportedError
}

func (g *Gluster) BlockVolumeDestroy(host string, blockHostingVolumeName string, blockVolumeName string) error {
	return executors.NotSupportedError
}
