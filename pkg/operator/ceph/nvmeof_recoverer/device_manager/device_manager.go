package device_manager

import (
	"fmt"
	"sync"

	"github.com/coreos/pkg/capnslog"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
)

const (
	controllerName = "device-manager"
)

var logger = capnslog.NewPackageLogger("github.com/rook/rook", controllerName)

type DeviceManager struct {
	Log logr.Logger
}

var instance *DeviceManager
var once sync.Once

func GetInstance() *DeviceManager {
	once.Do(func() {
		instance = &DeviceManager{}
	})
	return instance
}

// ReassignFailedNode finds the next available node for a device and updates the mapping.
func (dm *DeviceManager) FailingNvmeOfStorage(dp *appsv1.Deployment) error {

	attachedNode, devicePath, err := dm.getAttachedNodeandDevicePath(dp)
	if err != nil {
		return err
	}

	return fmt.Errorf("fail to fail NvmeOfStorage about attachedNode %s and devicePath %s", attachedNode, devicePath)
}

func (dm *DeviceManager) getAttachedNodeandDevicePath(dp *appsv1.Deployment) (string, string, error) {
	var attachedNode, devicePath string

	for _, container := range dp.Spec.Template.Spec.Containers {
		for _, env := range container.Env {
			if env.Name == "ROOK_NODE_NAME" {
				logger.Infof("attached node : %s", env.Value)
				attachedNode = env.Value
			}

			if env.Name == "ROOK_BLOCK_PATH" {
				logger.Infof("attached node : %s", env.Value)
				devicePath = env.Value
			}
		}
	}

	if attachedNode == "" || devicePath == "" {
		return "", "", fmt.Errorf("ROOK_NODE_NAME or ROOK_BLOCK_PATH not found in the Deployment environment")
	}

	return attachedNode, devicePath, nil
}
