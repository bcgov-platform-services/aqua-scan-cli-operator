/*
Copyright 2021.

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

package controllers

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	mamoadevopsgovbccav1alpha1 "github.com/bcgov-platform-services/aqua-scan-cli-operator/api/v1alpha1"
)

const aquaScannerAccountFinalizer = "mamoa.devops.gov.bc.ca.devops.gov.bc.ca/finalizer"

// AquaScannerAccountReconciler reconciles a AquaScannerAccount object
type AquaScannerAccountReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=mamoa.devops.gov.bc.ca.devops.gov.bc.ca,resources=aquascanneraccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=mamoa.devops.gov.bc.ca.devops.gov.bc.ca,resources=aquascanneraccounts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=mamoa.devops.gov.bc.ca.devops.gov.bc.ca,resources=aquascanneraccounts/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AquaScannerAccount object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *AquaScannerAccountReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// Fetch the aqua scanner account instance
	aquaScannerAccount := &mamoadevopsgovbccav1alpha1.AquaScannerAccount{}
	err := r.Get(ctx, req.NamespacedName, aquaScannerAccount)
	aquaScannerAccountName := "ScannerCLI_" + req.Namespace

	aquaUrl := os.Getenv("AQUA_URL")
	aquaSecret := os.Getenv("AQUA_SECRET")

	if err != nil {
		if errors.IsNotFound(err) {
			ctrl.Log.Error(err, "AquaScannerAccount not found. Ignoring as this object is deleted")
			return ctrl.Result{}, nil
		}
		ctrl.Log.Error(err, "Failed to get AquaScannerAccount %v", req.NamespacedName)
		// if another error it means we failed to get the AquaScannerAccount
		return ctrl.Result{}, err
	}
	// if in wrong namespace
	if !strings.HasSuffix(req.Namespace, "-tools") {
		ctrl.Log.Error(err, "AquaScannerAccount can only be created in namespaces ending in -tools. It was created in %v", req.Namespace)
		err := errors.NewBadRequest("AquaScannerAccount not allowed to be created in a non '-tools' namespace")

		aquaScannerAccount.Status.CurrentState = "Failure"

		aquaScannerAccount.Status.Timestamp = v1.Timestamp{Seconds: time.Now().Unix(), Nanos: int32(time.Now().UnixNano())}
		updateErr := r.Status().Update(ctx, aquaScannerAccount)

		if updateErr != nil {
			ctrl.Log.Error(err, "Failed to update aquaScannerAccount status")
			return ctrl.Result{}, err
		}

		return ctrl.Result{Requeue: false}, err
	}

	// Check if the AquaScannerAccount instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isAquaScannerAccountMarkedToBeDeleted := aquaScannerAccount.GetDeletionTimestamp() != nil
	if isAquaScannerAccountMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(aquaScannerAccount, aquaScannerAccountFinalizer) {
			// Run finalization logic for aquaScannerAccountFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalizeAquaScannerAccount(ctrl.Log, aquaScannerAccount, aquaScannerAccountName); err != nil {
				return ctrl.Result{}, err
			}

			// Remove aquaScannerAccountFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(aquaScannerAccount, aquaScannerAccountFinalizer)
			err := r.Update(ctx, aquaScannerAccount)
			if err != nil {
				ctrl.Log.Error(err, "Failed to remove aquaScannerAccount Finalizer")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer for this CR
	if !controllerutil.ContainsFinalizer(aquaScannerAccount, aquaScannerAccountFinalizer) {
		controllerutil.AddFinalizer(aquaScannerAccount, aquaScannerAccountFinalizer)
		err = r.Update(ctx, aquaScannerAccount)
		if err != nil {
			ctrl.Log.Error(err, "Failed to add aquaScannerAccount Finalizer")
			return ctrl.Result{}, err
		}
	}

	if aquaScannerAccount.Status.CurrentState != "Complete" {
		aquaScannerAccount.Status.CurrentState = "Running"
		aquaScannerAccount.Status.Timestamp = v1.Timestamp{Seconds: time.Now().Unix(), Nanos: int32(time.Now().UnixNano())}
		updateErr := r.Status().Update(ctx, aquaScannerAccount)
		if updateErr != nil {
			ctrl.Log.Error(err, "Failed to update aquaScannerAccount Status")
			return ctrl.Result{}, err
		}

		// if there are existing accounts with the same name in aqua they must be deleted first

	}
	// your logic here
	// possible states are Running New Complete Failure
	// if the status of the aqua scanner account is !Complete or !Failure or !Running
	// update state to Running
	// create the aqua scanner user and embed credentials in status
	// change status to complete or failure
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AquaScannerAccountReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mamoadevopsgovbccav1alpha1.AquaScannerAccount{}).
		Complete(r)
}

func (r *AquaScannerAccountReconciler) finalizeAquaScannerAccount(reqLogger *log.DelegatingLogger, m *mamoadevopsgovbccav1alpha1.AquaScannerAccount, aquaScannerName string) error {
	// TODO(user): Add the cleanup steps that the operator
	aquaUrl := os.Getenv("AQUA_URL")
	aquaSecret := os.Getenv("AQUA_SECRET")

	// needs to do before the CR can be deleted. Examples
	// of finalizers include performing backups and deleting
	// resources that are not owned by this CR, like a PVC.
	reqLogger.Info("Successfully finalized AquaScannerAccount")
	return nil
}

func doesAquaAccountAlreadyExist(reqLogger *log.DelegatingLogger, accountName string) (bool, error) {
	reqLogger.Info("Checking if %v was created previously in aqua", accountName)
	reqUrl := os.Getenv("AQUA_URL") + "/users/" + accountName
	client := &http.Client{}
	req, _ := http.NewRequest("GET", reqUrl, nil)
	req.Header.Set("Authorization", "Bearer "+os.Getenv("AQUA_SECRET"))
	req.Header.Set("Accept", "application/json")

	res, err := client.Do(req)

	if err != nil {
		reqLogger.Error(err, "Failed request to GET %v from aqua", accountName)
		return false, err
	}

	if res.StatusCode == 404 {
		reqLogger.Info("User %v not found in aqua", accountName)
		return false, nil
	}

	if res.StatusCode == 200 {
		reqLogger.Info("User %v already exists in aqua", accountName)
		return true, nil
	}

	errorMsg := "There was an issue making the request to GET user " + accountName + " from aqua"
	qualifiedResource := schema.GroupResource{mamoadevopsgovbccav1alpha1.GroupVersion.Group, "AquaScannerAccount"}

	return false, errors.NewGenericServerResponse(res.StatusCode, "GET", qualifiedResource, "Generic Error", errorMsg, 10, true)
}
