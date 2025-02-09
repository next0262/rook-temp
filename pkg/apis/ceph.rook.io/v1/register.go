/*
Copyright 2020 The Rook Authors. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	bktv1alpha1 "github.com/kube-object-storage/lib-bucket-provisioner/pkg/apis/objectbucket.io/v1alpha1"
	cephrookio "github.com/rook/rook/pkg/apis/ceph.rook.io"
)

const (
	CustomResourceGroup = "ceph.rook.io"
	Version             = "v1"
)

// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = schema.GroupVersion{Group: cephrookio.CustomResourceGroupName, Version: Version}

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

var (
	// SchemeBuilder and AddToScheme will stay in k8s.io/kubernetes.
	SchemeBuilder      runtime.SchemeBuilder
	localSchemeBuilder = &SchemeBuilder
	AddToScheme        = localSchemeBuilder.AddToScheme
)

func init() {
	// We only register manually written functions here. The registration of the
	// generated functions takes place in the generated files. The separation
	// makes the code compile even when the generated files are missing.
	localSchemeBuilder.Register(addKnownTypes)
}

// Adds the list of known types to api.Scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&CephClient{},
		&CephClientList{},
		&CephCluster{},
		&CephClusterList{},
		&CephBlockPool{},
		&CephBlockPoolList{},
		&CephFilesystem{},
		&CephFilesystemList{},
		&CephNFS{},
		&CephNFSList{},
		&CephObjectStore{},
		&CephObjectStoreList{},
		&CephObjectStoreUser{},
		&CephObjectStoreUserList{},
		&CephObjectRealm{},
		&CephObjectRealmList{},
		&CephObjectZoneGroup{},
		&CephObjectZoneGroupList{},
		&CephObjectZone{},
		&CephObjectZoneList{},
		&CephBucketTopic{},
		&CephBucketTopicList{},
		&CephBucketNotification{},
		&CephBucketNotificationList{},
		&CephRBDMirror{},
		&CephRBDMirrorList{},
		&CephFilesystemMirror{},
		&CephFilesystemMirrorList{},
		&CephFilesystemSubVolumeGroup{},
		&CephFilesystemSubVolumeGroupList{},
		&CephBlockPoolRadosNamespace{},
		&CephBlockPoolRadosNamespaceList{},
		&CephCOSIDriver{},
		&CephCOSIDriverList{},
		&NvmeOfOSD{},
		&NvmeOfOSDList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	scheme.AddKnownTypes(bktv1alpha1.SchemeGroupVersion,
		&bktv1alpha1.ObjectBucketClaim{},
		&bktv1alpha1.ObjectBucketClaimList{},
		&bktv1alpha1.ObjectBucket{},
		&bktv1alpha1.ObjectBucketList{},
	)
	metav1.AddToGroupVersion(scheme, bktv1alpha1.SchemeGroupVersion)
	return nil
}
