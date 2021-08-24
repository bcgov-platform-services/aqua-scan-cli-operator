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
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	mamoadevopsgovbccav1alpha1 "github.com/bcgov-platform-services/aqua-scan-cli-operator/api/v1alpha1"
	"github.com/bcgov-platform-services/aqua-scan-cli-operator/utils"
	"github.com/m1/go-generate-password/generator"
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
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile

func (r *AquaScannerAccountReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// Fetch the aqua scanner account instance
	aquaScannerAccount := &mamoadevopsgovbccav1alpha1.AquaScannerAccount{}

	err := r.Get(ctx, req.NamespacedName, aquaScannerAccount)
	namespacePrefix := strings.TrimSuffix(req.Namespace, "-tools")
	aquaScannerAccountName := "ScannerCLI_" + namespacePrefix

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
			ctrl.Log.Error(updateErr, "Failed to update aquaScannerAccount status")
			return ctrl.Result{Requeue: true}, updateErr
		}

		return ctrl.Result{Requeue: true}, err
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
				return ctrl.Result{Requeue: true}, err
			}

			// Remove aquaScannerAccountFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(aquaScannerAccount, aquaScannerAccountFinalizer)
			err := r.Update(ctx, aquaScannerAccount)
			if err != nil {
				ctrl.Log.Error(err, "Failed to remove aquaScannerAccount Finalizer")
				return ctrl.Result{Requeue: true}, err
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
			return ctrl.Result{Requeue: true}, err
		}
	}

	if aquaScannerAccount.Status.CurrentState != "Complete" {
		aquaScannerAccount.Status.CurrentState = "Running"
		aquaScannerAccount.Status.Timestamp = v1.Timestamp{Seconds: time.Now().Unix(), Nanos: int32(time.Now().UnixNano())}
		updateErr := r.Status().Update(ctx, aquaScannerAccount)
		if updateErr != nil {
			ctrl.Log.Error(err, "Failed to update aquaScannerAccount Status")
			return ctrl.Result{Requeue: true}, err
		}

		applicationScope := utils.ApplicationScope{
			Name:               aquaScannerAccountName,
			Description:        "Scanner scoped to " + namespacePrefix + "-* and DockerHub only.",
			TechnicalLeadEmail: "",
			NamespacePrefix:    namespacePrefix,
		}

		role := utils.Role{
			Name:             aquaScannerAccountName,
			Description:      "AquaScannerAccount created Role to allow Scanning of resources scoped to " + namespacePrefix + "-* and DockerHub only.",
			ApplicationScope: applicationScope,
		}

		applicationScopeErr := utils.CreateAquaApplicationScope(ctrl.Log, applicationScope)

		if applicationScopeErr != nil {
			ctrl.Log.Error(applicationScopeErr, "Failed to create application scope")
			aquaScannerAccount.Status.CurrentState = "Failure"
			aquaScannerAccount.Status.Timestamp = v1.Timestamp{Seconds: time.Now().Unix(), Nanos: int32(time.Now().UnixNano())}
			updateErr = r.Status().Update(ctx, aquaScannerAccount)

			if updateErr != nil {
				ctrl.Log.Error(updateErr, "Failed to update aquaScannerAccount status")
				return ctrl.Result{Requeue: true}, updateErr
			}
			return ctrl.Result{Requeue: true}, applicationScopeErr
		}

		roleErr := utils.CreateAquaRole(ctrl.Log, role)

		if roleErr != nil {
			ctrl.Log.Error(roleErr, "Failed to create role")
			aquaScannerAccount.Status.CurrentState = "Failure"
			aquaScannerAccount.Status.Timestamp = v1.Timestamp{Seconds: time.Now().Unix(), Nanos: int32(time.Now().UnixNano())}
			updateErr = r.Status().Update(ctx, aquaScannerAccount)

			if updateErr != nil {
				ctrl.Log.Error(updateErr, "Failed to update aquaScannerAccount status")
				return ctrl.Result{Requeue: true}, updateErr
			}

			return ctrl.Result{Requeue: true}, roleErr
		}

		var pwd *string

		if aquaScannerAccount.Status.AccountSecret != "" {
			pwd = &aquaScannerAccount.Status.AccountSecret
		} else {
			config := generator.Config{
				Length:                     16,
				IncludeSymbols:             true,
				IncludeNumbers:             true,
				IncludeLowercaseLetters:    true,
				IncludeUppercaseLetters:    true,
				ExcludeSimilarCharacters:   false,
				ExcludeAmbiguousCharacters: true,
			}
			g, _ := generator.New(&config)

			pwd, _ = g.Generate()
		}

		user := utils.User{
			Name:     aquaScannerAccountName,
			Password: *pwd,
			Role:     role,
		}

		aquaScannerAccount.Status.AccountName = user.Name
		aquaScannerAccount.Status.AccountSecret = user.Password
		aquaScannerAccount.Status.Timestamp = v1.Timestamp{Seconds: time.Now().Unix(), Nanos: int32(time.Now().UnixNano())}
		updateErr = r.Status().Update(ctx, aquaScannerAccount)

		if updateErr != nil {
			ctrl.Log.Error(updateErr, "Failed to update aquaScannerAccount status")
			return ctrl.Result{Requeue: true}, updateErr
		}

		userErr := utils.CreateAquaAccount(ctrl.Log, user)

		if userErr != nil {
			ctrl.Log.Error(userErr, "Failed to create user")
			aquaScannerAccount.Status.CurrentState = "Failure"
			aquaScannerAccount.Status.Timestamp = v1.Timestamp{Seconds: time.Now().Unix(), Nanos: int32(time.Now().UnixNano())}
			updateErr = r.Status().Update(ctx, aquaScannerAccount)

			if updateErr != nil {
				ctrl.Log.Error(updateErr, "Failed to update aquaScannerAccount status")
				return ctrl.Result{Requeue: true}, updateErr
			}

			return ctrl.Result{Requeue: true}, userErr
		}

		// set status to Complete
		aquaScannerAccount.Status.CurrentState = "Complete"
		aquaScannerAccount.Status.Timestamp = v1.Timestamp{Seconds: time.Now().Unix(), Nanos: int32(time.Now().UnixNano())}
		updateErr = r.Status().Update(ctx, aquaScannerAccount)

		if updateErr != nil {
			ctrl.Log.Error(updateErr, "Failed to update aquaScannerAccount status")
			return ctrl.Result{Requeue: true}, updateErr
		}

	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AquaScannerAccountReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mamoadevopsgovbccav1alpha1.AquaScannerAccount{}).
		Complete(r)
}

func (r *AquaScannerAccountReconciler) finalizeAquaScannerAccount(reqLogger *log.DelegatingLogger, m *mamoadevopsgovbccav1alpha1.AquaScannerAccount, aquaScannerName string) error {

	delAcctErr := utils.DeleteAquaAccount(ctrl.Log, aquaScannerName)
	if delAcctErr != nil {
		return delAcctErr
	}

	delRoleErr := utils.DeleteAquaRole(ctrl.Log, aquaScannerName)
	if delRoleErr != nil {
		return delRoleErr
	}

	delAppScopeErr := utils.DeleteAquaApplicationScope(ctrl.Log, aquaScannerName)
	if delAppScopeErr != nil {
		return delAppScopeErr
	}

	reqLogger.Info("Successfully finalized AquaScannerAccount")
	return nil
}
