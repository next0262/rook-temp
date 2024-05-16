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

// Package nvmeofOSD to reconcile a NvmeOfOSD CR.
package nvmeofosd

import (
	"context"
	"fmt"
	"reflect"

	"emperror.dev/errors"
	"github.com/coreos/pkg/capnslog"

	"github.com/rook/rook/pkg/clusterd"
	opcontroller "github.com/rook/rook/pkg/operator/ceph/controller"
	"github.com/rook/rook/pkg/operator/ceph/nvmeof_recoverer/device_manager"
	"github.com/rook/rook/pkg/operator/ceph/reporting"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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

// List of object resources to watch by the controller

var objectsToWatch = []client.Object{
	&corev1.Pod{TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: corev1.SchemeGroupVersion.String()}},
}

var deploymentKind = reflect.TypeOf(appsv1.Deployment{}).Name()

// Sets the type meta for the controller main object
var controllerTypeMeta = metav1.TypeMeta{Kind: deploymentKind, APIVersion: appsv1.SchemeGroupVersion.String()}

var _ reconcile.Reconciler = &ReconcileNvmeOfOSD{}

// ReconcileNvmeOfOSD reconciles a NvmeOfOSD object
type ReconcileNvmeOfOSD struct {
	client           client.Client
	scheme           *runtime.Scheme
	context          *clusterd.Context
	opManagerContext context.Context
	recorder         record.EventRecorder
	deviceManager    *device_manager.DeviceManager
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
		deviceManager:    device_manager.GetInstance(),
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
		&appsv1.Deployment{TypeMeta: controllerTypeMeta})

	err = c.Watch(cmKind, &handler.EnqueueRequestForObject{}, opcontroller.WatchControllerPredicate())
	if err != nil {
		return err
	}

	// Watch all other resources
	for _, t := range objectsToWatch {
		ownerRequest := handler.EnqueueRequestForOwner(
			mgr.GetScheme(),
			mgr.GetRESTMapper(),
			&appsv1.Deployment{},
		)
		err = c.Watch(source.Kind(mgr.GetCache(), t), ownerRequest,
			opcontroller.WatchPredicateForNonCRDObject(&appsv1.Deployment{TypeMeta: controllerTypeMeta}, mgr.GetScheme()))
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *ReconcileNvmeOfOSD) Reconcile(context context.Context, request reconcile.Request) (reconcile.Result, error) {
	reconcileResponse, err := r.reconcile(request)
	return reconcileResponse, err
}

func (r *ReconcileNvmeOfOSD) reconcile(request reconcile.Request) (reconcile.Result, error) {
	logger.Debugf("reconciling NvmeOfOSD. Request.Namespace: %s, Request.Name: %s", request.Namespace, request.Name)

	// Fetch the NvmeOfOSD CRD object
	dp, pods, err := r.fetchNvmeOfOSD(request)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Handle the NvmeOfOSD based on status
	result, err := r.handleNvmeOfOSDStatus(dp, pods)

	// Placeholder for updating the crush map
	// TODO (cheolho.kang): Need to implement handler
	return reporting.ReportReconcileResult(logger, r.recorder, request, dp, result, err)
}

// fetchNvmeOfOSD retrieves the NvmeOfOSD instance by name and namespace.
func (r *ReconcileNvmeOfOSD) fetchNvmeOfOSD(request reconcile.Request) (*appsv1.Deployment, *corev1.PodList, error) {
	dp := &appsv1.Deployment{}
	err := r.client.Get(r.opManagerContext, request.NamespacedName, dp)
	if err != nil {
		logger.Errorf("unable to fetch Deployment, %v", err)
		return nil, nil, err
	}

	pods := &corev1.PodList{}
	err = r.client.List(r.opManagerContext, pods, client.InNamespace(request.Namespace), client.MatchingLabels(dp.Spec.Template.Labels))
	if err != nil {
		logger.Errorf("unable to fetch Pods of Deployment, %v", err)
		return nil, nil, err
	}

	return dp, pods, nil
}

func (r *ReconcileNvmeOfOSD) handleNvmeOfOSDStatus(dp *appsv1.Deployment, pods *corev1.PodList) (reconcile.Result, error) {
	if len(pods.Items) == 0 {
		// If there are no pods, return with an error
		return reconcile.Result{}, fmt.Errorf("no pods found for deployment %s", dp.Name)
	}

	pod := &pods.Items[0]
	switch pod.Status.Phase {
	// Placeholder for future status handling
	// TODO (cheolho.kang): Implement the logic to handle the NvmeOfOSD status
	case corev1.PodFailed:
		logger.Debugf("status changed to %s", pod.Status.Phase)
		r.deviceManager.FailingNvmeOfStorage(dp)
		return reconcile.Result{}, nil
	default:
		// Handle other statuses or do nothing
		logger.Errorf("other status: %s", pod.Status.Phase)
		return reconcile.Result{}, nil
	}
}
