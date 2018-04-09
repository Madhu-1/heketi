// +build functional

//
// Copyright (c) 2018 The heketi Authors
//
// This file is licensed to you under your choice of the GNU Lesser
// General Public License, version 3 or any later version (LGPLv3 or
// later), or the GNU General Public License, version 2 (GPLv2), in all
// cases as published by the Free Software Foundation.
//

package functional

import (
	"sync"
	"testing"

	"github.com/heketi/heketi/pkg/glusterfs/api"
	"github.com/heketi/tests"
)

func TestReqTrottling(t *testing.T) {
	setupCluster(t, 4, 8)
	defer teardownCluster(t)
	t.Run("testReqTrottlingCreateVolume", testReqTrottlingCreateVolume)

}

func testReqTrottlingCreateVolume(t *testing.T) {
	vl, err := heketi.VolumeList()
	tests.Assert(t, err == nil, "expected err == nil, got:", err)
	tests.Assert(t, len(vl.Volumes) == 0,
		"expected len(vl.Volumes) == 0, got:", len(vl.Volumes))

	volReq := &api.VolumeCreateRequest{}
	volReq.Size = 10
	volReq.Durability.Type = api.DurabilityReplicate
	volReq.Durability.Replicate.Replica = 3
	var wg sync.WaitGroup
	count := 0
	for i := 1; i < 30; i++ {
		go func(t *testing.T, wg *sync.WaitGroup, c *int) {
			wg.Add(1)
			defer wg.Done()
			*c = *c + 1
			_, err := heketi.VolumeCreate(volReq)
			tests.Assert(t, err == nil, "expected err == nil, got:", err)

		}(t, &wg, &count)
	}
	for count < 29 {
		continue
	}
	_, err = heketi.VolumeCreate(volReq)
	tests.Assert(t, err != nil, "expected err != nil, got:", err)
	go func() {
		for {
			vl, err = heketi.VolumeList()
			if len(vl.Volumes) == 0 {
				continue
			} else {
				_, err := heketi.VolumeCreate(volReq)
				tests.Assert(t, err == nil, "expected err == nil, got:", err)
				return
			}

		}
	}()
	wg.Wait()
	vl, err = heketi.VolumeList()
	tests.Assert(t, err == nil, "expected err == nil, got:", err)
	tests.Assert(t, len(vl.Volumes) == 30,
		"expected len(vl.Volumes) == 30, got:", len(vl.Volumes))
}
