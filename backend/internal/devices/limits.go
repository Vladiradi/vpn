package devices

import (
	"context"
	"errors"
	"fmt"
)

// Hard limits for the MVP.
const (
	MaxDevicesPerNode = 100
	MaxUsersPerNode   = 50
	MaxDevicesPerUser = 2
)

// ErrLimitExceeded is returned when a device limit would be exceeded.
var ErrLimitExceeded = errors.New("device limit exceeded")

// CheckLimits verifies that creating a device for userID on nodeID does not exceed limits.
func (r *Repository) CheckLimits(ctx context.Context, userID string, nodeID *string) error {
	// Global per-user limit.
	userDevices, err := r.CountActiveByUser(ctx, userID)
	if err != nil {
		return err
	}
	if userDevices >= MaxDevicesPerUser {
		return fmt.Errorf("%w: user already has %d active device(s), max %d", ErrLimitExceeded, userDevices, MaxDevicesPerUser)
	}

	if nodeID == nil || *nodeID == "" {
		return nil
	}

	// Per-node device limit.
	nodeDevices, err := r.CountActiveByNode(ctx, *nodeID)
	if err != nil {
		return err
	}
	if nodeDevices >= MaxDevicesPerNode {
		return fmt.Errorf("%w: node already has %d active device(s), max %d", ErrLimitExceeded, nodeDevices, MaxDevicesPerNode)
	}

	// Per-node distinct user limit. Existing user devices on this node do not count again.
	alreadyOnNode, err := r.UserHasDeviceOnNode(ctx, userID, *nodeID)
	if err != nil {
		return err
	}
	if !alreadyOnNode {
		nodeUsers, err := r.CountActiveUsersByNode(ctx, *nodeID)
		if err != nil {
			return err
		}
		if nodeUsers >= MaxUsersPerNode {
			return fmt.Errorf("%w: node already has %d user(s), max %d", ErrLimitExceeded, nodeUsers, MaxUsersPerNode)
		}
	}

	return nil
}
