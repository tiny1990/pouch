package mgr

import (
	"context"
	"os"
	"strings"

	"github.com/alibaba/pouch/pkg/errtypes"
	"github.com/alibaba/pouch/volume"
	"github.com/alibaba/pouch/volume/types"

	"github.com/pkg/errors"
)

// VolumeMgr defines interface to manage container volume.
type VolumeMgr interface {
	// Create is used to create volume.
	Create(ctx context.Context, name, driver string, options, labels map[string]string) error

	// Remove is used to delete an existing volume.
	Remove(ctx context.Context, name string) error

	// List returns all volumes on this host.
	List(ctx context.Context, labels map[string]string) ([]*types.Volume, error)

	// Get returns the information of volume that specified name/id.
	Get(ctx context.Context, name string) (*types.Volume, error)

	// Path returns the mount path of volume.
	Path(ctx context.Context, name string) (string, error)

	// Attach is used to bind a volume to container.
	Attach(ctx context.Context, name string, options map[string]string) (*types.Volume, error)

	// Detach is used to unbind a volume from container.
	Detach(ctx context.Context, name string, options map[string]string) (*types.Volume, error)
}

// VolumeManager is the default implement of interface VolumeMgr.
type VolumeManager struct {
	core *volume.Core
}

// NewVolumeManager creates a brand new volume manager.
func NewVolumeManager(cfg volume.Config) (*VolumeManager, error) {
	// init voluem config
	cfg.RemoveVolume = true
	cfg.DefaultBackend = "local"

	core, err := volume.NewCore(cfg)
	if err != nil {
		return nil, err
	}

	return &VolumeManager{
		core: core,
	}, nil
}

// Create is used to create volume.
func (vm *VolumeManager) Create(ctx context.Context, name, driver string, options, labels map[string]string) error {
	id := types.VolumeID{
		Name:      name,
		Driver:    driver,
		Options:   map[string]string{},
		Selectors: map[string]string{},
	}

	if labels != nil {
		id.Labels = labels
	} else {
		id.Labels = map[string]string{}
	}

	for key, opt := range options {
		if strings.HasPrefix(key, "selector.") {
			key = strings.TrimPrefix(key, "selector.")
			id.Selectors[key] = opt
			continue
		}

		id.Options[key] = opt
	}

	return vm.core.CreateVolume(id)
}

// Remove is used to delete an existing volume.
func (vm *VolumeManager) Remove(ctx context.Context, name string) error {
	// TODO: check container use.
	id := types.VolumeID{
		Name: name,
	}
	if err := vm.core.RemoveVolume(id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return errors.Wrap(errtypes.ErrNotfound, err.Error())
		}
	}
	return nil
}

// List returns all volumes on this host.
func (vm *VolumeManager) List(ctx context.Context, labels map[string]string) ([]*types.Volume, error) {
	if _, ok := labels["hostname"]; !ok {
		hostname, err := os.Hostname()
		if err != nil {
			return nil, err
		}

		labels["hostname"] = hostname
	}

	return vm.core.ListVolumes(labels)
}

// Get returns the information of volume that specified name/id.
func (vm *VolumeManager) Get(ctx context.Context, name string) (*types.Volume, error) {
	id := types.VolumeID{
		Name: name,
	}
	vol, err := vm.core.GetVolume(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, errors.Wrap(errtypes.ErrNotfound, err.Error())
		}
		return nil, err
	}
	return vol, nil
}

// Path returns the mount path of volume.
func (vm *VolumeManager) Path(ctx context.Context, name string) (string, error) {
	id := types.VolumeID{
		Name: name,
	}
	return vm.core.VolumePath(id)
}

// Attach is used to bind a volume to container.
func (vm *VolumeManager) Attach(ctx context.Context, name string, options map[string]string) (*types.Volume, error) {
	id := types.VolumeID{
		Name: name,
	}
	return vm.core.AttachVolume(id, options)
}

// Detach is used to unbind a volume from container.
func (vm *VolumeManager) Detach(ctx context.Context, name string, options map[string]string) (*types.Volume, error) {
	id := types.VolumeID{
		Name: name,
	}
	return vm.core.DetachVolume(id, options)
}
