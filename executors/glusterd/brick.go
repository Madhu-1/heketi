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

func (g *GlusterdExecutor) BrickCreate(host string,
	brick *executors.BrickRequest) (*executors.BrickInfo, error) {
	b := &executors.BrickInfo{
		Path: brick.Path,
	}
	return b, nil
}

func (g *GlusterdExecutor) BrickDestroy(host string,
	brick *executors.BrickRequest) (bool, error) {

	return false, executors.NotSupportedError
}
