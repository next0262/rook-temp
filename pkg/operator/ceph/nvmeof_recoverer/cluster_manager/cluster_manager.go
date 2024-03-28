package cluster_manager

import (
	"sync"

	"github.com/go-logr/logr"
)

type NodeDeviceMap struct {
	Node           string // Node that the device is attached to
	AttachedDevice string // Device that is attached to the node
}

type ClusterManager struct {
	Log             logr.Logger
	StorageToDevice map[string][]NodeDeviceMap // key: NvmeOfStorage.Name, value: node to device map
}

var instance *ClusterManager
var once sync.Once

func GetInstance() *ClusterManager {
	once.Do(func() {
		instance = &ClusterManager{
			StorageToDevice: make(map[string][]NodeDeviceMap),
		}
	})
	return instance
}
