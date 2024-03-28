package cluster_manager

import (
	"fmt"
	"sync"

	"github.com/go-logr/logr"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
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

// ReassignFailedNode finds the next available node for a device and updates the mapping.
func (cm *ClusterManager) ReassignFailedNode(nvmeOfOSD *cephv1.NvmeOfOSD) error {
	storageName := nvmeOfOSD.Spec.NvmeOfStorageName
	failedNode := nvmeOfOSD.Spec.AttachNode
	nodeMaps, exists := cm.StorageToDevice[storageName]
	if !exists {
		return fmt.Errorf("no storage found with name %s", storageName)
	}

	for i, nodeMap := range nodeMaps {
		if nodeMap.Node == failedNode {
			// Find next node in a round-robin fashion
			nextIndex := (i + 1) % len(nodeMaps)
			nextNode := nodeMaps[nextIndex].Node
			nodeMaps[i].Node = nextNode // Update the mapping

			nvmeOfOSD.Spec.AttachNode = nextNode
			nvmeOfOSD.Status.Status = "creating"
			return nil
		}
	}

	return fmt.Errorf("node %s not found in mappings for failed target node %s", failedNode, storageName)
}
