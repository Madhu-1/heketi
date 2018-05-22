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
	"fmt"
	"strings"

	"github.com/gluster/glusterd2/pkg/api"
	"github.com/heketi/heketi/executors"
	"github.com/lpabon/godbc"
)

func (g *Gluster) prepareBrick(inSet int, brick []executors.BrickInfo) []api.BrickReq {
	var bricks []string
	peers, err := g.Client.Peers()
	if err != nil {
		return nil
	}
	for i, b := range brick[:inSet] {
		bricks[i] = fmt.Sprintf("%v:%v ", b.Host, b.Path)
	}

	var Bricks []api.BrickReq

	for _, brick := range bricks {
		host := strings.Split(brick, ":")[0]
		path := strings.Split(brick, ":")[1]
		for _, peer := range peers {
			for _, addr := range peer.PeerAddresses {
				// TODO: Normalize presence/absence of port in peer address
				if strings.Split(addr, ":")[0] == strings.Split(host, ":")[0] {
					Bricks = append(Bricks, api.BrickReq{
						PeerID: peer.ID.String(),
						Path:   path,
					})
				}
			}
		}
	}
	return Bricks
}
func (g *Gluster) VolumeCreate(host string,
	volume *executors.VolumeRequest) (*executors.Volume, error) {

	godbc.Require(volume != nil)
	godbc.Require(host != "")
	godbc.Require(len(volume.Bricks) > 0)
	godbc.Require(volume.Name != "")
	subvols := []api.SubvolReq{}
	var inSet = 1
	req := api.VolCreateReq{}
	g.createClient(host)
	//arrange bicks in form of <uuid>:<path>

	switch volume.Type {
	case executors.DurabilityNone:
		logger.Info("Creating volume %v with no durability", volume.Name)
		inSet = 1

	case executors.DurabilityDispersion:
		logger.Info("Creating volume %v dispersion %v+%v",
			volume.Name, volume.Data, volume.Redundancy)
		inSet = volume.Data + volume.Redundancy

	case executors.DurabilityReplica:
		logger.Info("Creating volume %v replica %v", volume.Name, volume.Replica)

		inSet = volume.Replica
	}
	Bricks := g.prepareBrick(inSet, volume.Bricks)

	switch volume.Type {
	case executors.DurabilityNone:
		subvols = []api.SubvolReq{
			{
				Type:               "Distribute",
				Bricks:             Bricks,
				DisperseData:       volume.Data,
				DisperseRedundancy: volume.Redundancy,
			},
		}
	case executors.DurabilityReplica:
		logger.Info("Creating volume %v replica %v", volume.Name, volume.Replica)
		numBricks := len(Bricks)
		numSubvols := numBricks / volume.Replica

		for i := 0; i < numSubvols; i++ {
			idx := i * volume.Replica

			// If Arbiter is set, set it as Brick Type for last brick
			if volume.Arbiter {
				Bricks[idx+volume.Replica-1].Type = "arbiter"
			}

			subvols = append(subvols, api.SubvolReq{
				Type:         "Replicate",
				Bricks:       Bricks[idx : idx+volume.Replica],
				ReplicaCount: volume.Replica,
				ArbiterCount: 1,
			})
		}
	case executors.DurabilityDispersion:
		logger.Info("Creating volume %v dispersion %v+%v",
			volume.Name, volume.Data, volume.Redundancy)
		subvols = []api.SubvolReq{
			{
				Type:               "Disperse",
				Bricks:             Bricks,
				DisperseData:       volume.Data,
				DisperseRedundancy: volume.Redundancy,
			},
		}
	}

	req.Name = volume.Name
	req.Subvols = subvols
	req.Force = true

	vol, err := g.Client.VolumeCreate(req)
	if err != nil {
		return nil, err
	}
	//set volume options
	err = g.createVolumeOptionsCommand(volume)
	if err != nil {
		g.Client.VolumeDelete(volume.Name)
		return nil, err
	}

	//TODO need to fill all 17 fields
	err = g.Client.VolumeStart(vol.Name)
	if err != nil {
		g.Client.VolumeDelete(vol.Name)
		return nil, err
	}

	return g.formatVolumeResp(api.VolumeInfo(vol)), nil
}

func (g *Gluster) createVolumeOptionsCommand(volume *executors.VolumeRequest) error {

	// Go through all the Options and create volume set command

	vopt := make(map[string]string)
	for op, volOption := range volume.GlusterVolumeOptions {
		if op%2 == 0 {
			vopt[volOption] = volume.GlusterVolumeOptions[op+1]
		}
	}
	err := g.Client.VolumeSet(volume.Name, api.VolOptionReq{
		Options: vopt,
		//TODO do we need to set this advance flags in heketi
		Advanced: true,
		// Experimental: flagSetExp,
		// Deprecated:   flagSetDep,
	})
	return err

}

func (g *Gluster) VolumeDestroy(host string, volume string) error {
	godbc.Require(host != "")
	godbc.Require(volume != "")
	g.createClient(host)
	// First stop the volume, then delete it
	//TODO stop forcefully
	err := g.Client.VolumeStop(volume)

	if err != nil {
		logger.LogError("Unable to stop volume %v: %v", volume, err)
		return err
	}
	//TODO delete forcefully
	err = g.Client.VolumeDelete(volume)
	if err != nil {
		logger.LogError("Unable to delete volume %v: %v", volume, err)
		return logger.Err(fmt.Errorf("Unable to delete volume %v: %v", volume, err))
	}

	return nil
}

func (g *Gluster) VolumeInfo(host string, volume string) (*executors.Volume, error) {

	godbc.Require(volume != "")
	godbc.Require(host != "")
	g.createClient(host)
	volumes, err := g.Client.Volumes(volume, nil)
	if err != nil {
		return nil, fmt.Errorf("Unable to get volume info of volume name: %v", volume)
	}

	//logger.Debug("%+v\n", volumeInfo)
	return g.formatVolumeResp(api.VolumeInfo(volumes[0])), nil
}

func (g *Gluster) VolumeExpand(host string,
	volume *executors.VolumeRequest) (*executors.Volume, error) {

	godbc.Require(volume != nil)
	godbc.Require(host != "")
	godbc.Require(len(volume.Bricks) > 0)
	godbc.Require(volume.Name != "")
	g.createClient(host)
	var (
		inSet int
	)
	req := api.VolExpandReq{
		Force: true,
	}
	switch volume.Type {
	case executors.DurabilityNone:
		inSet = 1

	case executors.DurabilityReplica:
		inSet = volume.Replica
		req.ReplicaCount = volume.Replica
	case executors.DurabilityDispersion:
		inSet = volume.Data + volume.Redundancy

	}
	Bricks := g.prepareBrick(inSet, volume.Bricks)
	req.Bricks = Bricks
	vol, err := g.Client.VolumeExpand(volume.Name, req)

	if err != nil {
		return nil, err
	}

	//TODO rebalance is not implemented in GD2
	// if s.RemoteExecutor.RebalanceOnExpansion() {
	// 	commands = []string{fmt.Sprintf("gluster --mode=script volume rebalance %v start", volume.Name)}
	// 	_, err := s.RemoteExecutor.RemoteCommandExecute(host, commands, 10)
	// 	if err != nil {
	// 		// This is a hack. We fake success if rebalance fails.
	// 		// Mainly because rebalance may fail even if one brick is down for the given volume.
	// 		// The probability is just too high to undo the work done to create and attach bricks.
	// 		// Admins should be able to get new size to reflect by executing the rebalance cmd manually.
	// 		logger.LogError("Unable to start rebalance on the volume %v: %v", volume, err)
	// 		logger.LogError("Action Required: run rebalance manually on the volume %v", volume)
	// 		return &executors.Volume{}, nil
	// 	}
	// }

	return g.formatVolumeResp(api.VolumeInfo(vol)), nil
}

func (g *Gluster) formatVolumeResp(vol api.VolumeInfo) *executors.Volume {
	volumeResp := &executors.Volume{}
	volumeResp.VolumeName = vol.Name
	volumeResp.ID = vol.ID.String()
	volumeResp.ArbiterCount = vol.ArbiterCount
	volumeResp.ReplicaCount = vol.ReplicaCount

	//collect brick list
	for _, svol := range vol.Subvols {
		for _, brick := range svol.Bricks {
			b := executors.Brick{}
			b.Name = brick.ID.String()
			b.UUID = brick.ID.String()
			b.IsArbiter = int(brick.Type)
			b.HostUUID = brick.PeerID.String()
			volumeResp.Bricks.BrickList = append(volumeResp.Bricks.BrickList, b)
			volumeResp.BrickCount++
		}
	}
	volumeResp.DistCount = vol.DistCount
	volumeResp.Status = int(vol.State)
	volumeResp.StatusStr = vol.State.String()

	//options
	options := executors.Options{}
	for key, value := range vol.Options {
		opt := executors.Option{Name: key, Value: value}
		options.OptionList = append(options.OptionList, opt)
	}
	volumeResp.Options = options
	volumeResp.OptCount = len(vol.Options)
	//need to add transport here (with switch)
	//volumeResp.Transport = int(vol.Transport)
	volumeResp.Type = int(vol.Type)
	volumeResp.TypeStr = vol.Type.String()
	return volumeResp
}
