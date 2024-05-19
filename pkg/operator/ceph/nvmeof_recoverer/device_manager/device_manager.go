package device_manager

import (
	"fmt"
	"sync"

	"github.com/coreos/pkg/capnslog"

	"github.com/go-logr/logr"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	corev1 "k8s.io/api/core/v1"
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

func (dm *DeviceManager) FailingNvmeOfStorage(pod *corev1.Pod, nvmeOfStorages []cephv1.NvmeOfStorage) error {

	attachedNode, devicePath, err := dm.getAttachedNodeandDevicePath(pod)
	if err != nil {
		return err
	}

	for _, nvmeOfStorage := range nvmeOfStorages {
		for _, device := range nvmeOfStorage.Spec.Devices {
			if device.AttachedNode == attachedNode && device.DeviceName == devicePath {
				nvmeOfStorage.Status.Status = "Failed"
				logger.Infof("Updated NvmeOfStorage status for node %s and device %s", attachedNode, devicePath)
				return nil
			}
		}
	}

	return fmt.Errorf("failed to update NvmeOfStorage for attachedNode %s and devicePath %s", attachedNode, devicePath)

}

func (dm *DeviceManager) getAttachedNodeandDevicePath(pod *corev1.Pod) (string, string, error) {
	var attachedNode, devicePath string

	for _, container := range pod.Spec.Containers {
		for _, env := range container.Env {
			if env.Name == "ROOK_NODE_NAME" {
				logger.Infof("attached node : %s", env.Value)
				attachedNode = env.Value
			}

			if env.Name == "ROOK_BLOCK_PATH" {
				logger.Infof("block path : %s", env.Value)
				devicePath = env.Value
			}
		}
	}

	if attachedNode == "" || devicePath == "" {
		return "", "", fmt.Errorf("ROOK_NODE_NAME or ROOK_BLOCK_PATH not found in the Deployment environment")
	}

	return attachedNode, devicePath, nil
}
