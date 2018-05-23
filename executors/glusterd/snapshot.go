//
// Copyright (c) 2018 The heketi Authors
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

func (g *GlusterdExecutor) snapshotActivate(host string, snapshot string) error {
	return executors.NotSupportedError
}

func (g *GlusterdExecutor) SnapshotCloneVolume(host string, vcr *executors.SnapshotCloneRequest) (*executors.Volume, error) {
	return nil, executors.NotSupportedError
}

func (g *GlusterdExecutor) SnapshotCloneBlockVolume(host string, vcr *executors.SnapshotCloneRequest) (*executors.BlockVolumeInfo, error) {
	// TODO: cloning of block volume is not implemented yet
	return nil, executors.NotSupportedError
}

func (g *GlusterdExecutor) SnapshotDestroy(host string, snapshot string) error {
	return executors.NotSupportedError
}
