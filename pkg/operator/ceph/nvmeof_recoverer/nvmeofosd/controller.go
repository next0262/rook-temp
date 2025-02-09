/*
Copyright 2016 The Rook Authors. All rights reserved.

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

// Package nvmeofosd to reconcile a NvmeOfOSD CR.
package nvmeofosd

import (
	"context"
	"fmt"
	"reflect"

	"emperror.dev/errors"
	"github.com/coreos/pkg/capnslog"

	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	"github.com/rook/rook/pkg/clusterd"
	opcontroller "github.com/rook/rook/pkg/operator/ceph/controller"
	"github.com/rook/rook/pkg/operator/ceph/nvmeof_recoverer/cluster_manager"
	"github.com/rook/rook/pkg/operator/ceph/reporting"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	controllerName = "nvmeofosd-controller"
)

var logger = capnslog.NewPackageLogger("github.com/rook/rook", controllerName)

var nvmeOfOSDKind = reflect.TypeOf(cephv1.NvmeOfOSD{}).Name()

// Sets the type meta for the controller main object
var controllerTypeMeta = metav1.TypeMeta{
	Kind:       nvmeOfOSDKind,
	APIVersion: fmt.Sprintf("%s/%s", cephv1.CustomResourceGroup, cephv1.Version),
}

var _ reconcile.Reconciler = &ReconcileNvmeOfOSD{}

// ReconcileNvmeOfOSD reconciles a NvmeOfOSD object
type ReconcileNvmeOfOSD struct {
	client           client.Client
	scheme           *runtime.Scheme
	context          *clusterd.Context
	opManagerContext context.Context
	recorder         record.EventRecorder
	clusterManager   *cluster_manager.ClusterManager
}

// Add creates a new NvmeOfOSD Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, context *clusterd.Context, opManagerContext context.Context, opConfig opcontroller.OperatorConfig) error {
	return add(mgr, newReconciler(mgr, context, opManagerContext))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, context *clusterd.Context, opManagerContext context.Context) reconcile.Reconciler {
	return &ReconcileNvmeOfOSD{
		client:           mgr.GetClient(),
		context:          context,
		scheme:           mgr.GetScheme(),
		opManagerContext: opManagerContext,
		recorder:         mgr.GetEventRecorderFor("rook-" + controllerName),
		clusterManager:   cluster_manager.GetInstance(),
	}
}

func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New(controllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return errors.Wrapf(err, "failed to create %s controller", controllerName)
	}
	logger.Info("successfully started")

	// Watch for changes on the NvmeOfOSD CRD object
	cmKind := source.Kind(
		mgr.GetCache(),
		&cephv1.NvmeOfOSD{TypeMeta: controllerTypeMeta})
	err = c.Watch(cmKind, &handler.EnqueueRequestForObject{}, opcontroller.WatchControllerPredicate())
	if err != nil {
		return err
	}

	return nil
}

func (r *ReconcileNvmeOfOSD) Reconcile(context context.Context, request reconcile.Request) (reconcile.Result, error) {
	// workaround because the rook logging mechanism is not compatible with the controller-runtime logging interface
	reconcileResponse, err := r.reconcile(request)
	return reconcileResponse, err
}

func (r *ReconcileNvmeOfOSD) reconcile(request reconcile.Request) (reconcile.Result, error) {
	logger.Debug("reconciling NvmeOfOSD", "Request.Namespace", request.Namespace, "Request.Name", request.Name)

	// Fetch the NvmeOfOSD CRD object
	nvmeOfOSD, err := r.fetchNvmeOfOSD(request)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Handle the NvmeOfOSD based on status
	result, err := r.handleNvmeOfOSDStatus(nvmeOfOSD)
	return reporting.ReportReconcileResult(logger, r.recorder, request, nvmeOfOSD, result, err)
}

// fetchNvmeOfOSD retrieves the NvmeOfOSD instance by name and namespace.
func (r *ReconcileNvmeOfOSD) fetchNvmeOfOSD(request reconcile.Request) (*cephv1.NvmeOfOSD, error) {
	nvmeOfOSD := &cephv1.NvmeOfOSD{}
	err := r.client.Get(r.opManagerContext, request.NamespacedName, nvmeOfOSD)
	if err != nil {
		logger.Error(err, "unable to fetch NvmeOfOSD", "Request.Namespace", request.Namespace, "Request.Name", request.Name)
		return nil, err
	}
	return nvmeOfOSD, nil
}

// handleNvmeOfOSDStatus processes the NvmeOfOSD based on its current status.
func (r *ReconcileNvmeOfOSD) handleNvmeOfOSDStatus(nvmeOfOSD *cephv1.NvmeOfOSD) (reconcile.Result, error) {
	switch nvmeOfOSD.Status.Status {
	case "failed":
		err := r.clusterManager.ReassignFailedNode(nvmeOfOSD)
		if err != nil {
			logger.Error(err, "failed to reassign node for NvmeOfOSD", "NvmeOfOSD", nvmeOfOSD.Name)
			return reconcile.Result{}, err
		}

		// Update the NvmeOfOSD status
		r.updateCR(nvmeOfOSD)
		logger.Info("successfully reassigned node for NvmeOfOSD. nextNode: %s, attachedDevice: %s", nvmeOfOSD.Spec.AttachNode, nvmeOfOSD.Spec.Device)
	default:
		// Handle other statuses or do nothing
		logger.Debug("This NvmeOfOSD status does not need to action", "Status", nvmeOfOSD.Status.Status)
		return reconcile.Result{}, nil
	}
	return reconcile.Result{}, nil
}

// updateCR updates a NvmeOfOSD CR with the given status and spec.
func (r *ReconcileNvmeOfOSD) updateCR(nvmeOfOSD *cephv1.NvmeOfOSD) {
	if err := reporting.UpdateStatus(r.client, nvmeOfOSD); err != nil {
		logger.Warningf("failed to update nvmeOfOSD %q. %v", nvmeOfOSD.Name, err)
		return
	}
	logger.Debugf("nvmeOfOSD %q updated successfully", nvmeOfOSD.Name)
}
