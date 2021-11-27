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
	"os"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	asa "github.com/bcgov-platform-services/aqua-scan-cli-operator/api/v1alpha1"
	"github.com/bcgov-platform-services/aqua-scan-cli-operator/utils"
)

const aquaScannerAccountFinalizer = "mamoa.devops.gov.bc.ca/finalizer"

// AquaScannerAccountReconciler reconciles a AquaScannerAccount object
type AquaScannerAccountReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

type AquaObjectState struct {
	ApplicationScope string
	PermissionSet    string
	User             string
	Role             string
}

//+kubebuilder:rbac:groups=mamoa.devops.gov.bc.ca,resources=aquascanneraccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=mamoa.devops.gov.bc.ca,resources=aquascanneraccounts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=mamoa.devops.gov.bc.ca,resources=aquascanneraccounts/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile

func (r *AquaScannerAccountReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// Fetch the aqua scanner account instance
	aquaScannerAccount := &asa.AquaScannerAccount{}

	err := r.Get(ctx, req.NamespacedName, aquaScannerAccount)
	namespacePrefix := strings.TrimSuffix(req.Namespace, "-tools")
	aquaScannerAccountName := "ScannerCLI_" + namespacePrefix

	// set env var for aqua auth check when the variable is unset
	if os.Getenv("ASA_LOGIN_CHECK_DID_FAIL") == "" {
		utils.SetEnvForAsaLoginCheck(utils.GetAquaAuth().GetJWT, ctrl.Log)
	}

	aquaLoginCheckFailed, boolCastErr := strconv.ParseBool(os.Getenv("ASA_LOGIN_CHECK_DID_FAIL"))

	if boolCastErr != nil {
		ctrl.Log.Error(boolCastErr, "failed to parse boolean from env var ASA_LOGIN_CHECK_DID_FAIL")
		return ctrl.Result{}, boolCastErr
	}

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
		errorMessage := "AquaScannerAccount not allowed to be created in a non '-tools' namespace"
		err := errors.NewBadRequest(errorMessage)

		ctrl.Log.Error(err, "AquaScannerAccount can only be created in namespaces ending in -tools. It was created in %v", req.Namespace)

		updateErr := utils.UpdateStatus(ctx, aquaScannerAccount, asa.AquaScannerAccountStatus{State: "Failed", Message: errorMessage}, r.Status(), ctrl.Log)

		if updateErr != nil {
			return ctrl.Result{Requeue: true}, updateErr
		}

		return ctrl.Result{Requeue: true}, err
	}

	// initialize desired state
	if aquaScannerAccount.Status.DesiredState == (asa.AquaScannerAccountAquaObjectState{}) {
		aquaObjectState := asa.AquaScannerAccountAquaObjectState{ApplicationScope: "Created", PermissionSet: "Created", Role: "Created", User: "Created"}

		updateErr := utils.UpdateStatus(ctx, aquaScannerAccount, asa.AquaScannerAccountStatus{DesiredState: aquaObjectState}, r.Status(), ctrl.Log)

		if updateErr != nil {
			return ctrl.Result{Requeue: true}, updateErr
		}
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

	if aquaLoginCheckFailed {
		errorMessage := "AquaScannerAccount failed to authenticate with Aqua API. Reconcilliation Failed."

		updateErr := utils.UpdateStatus(ctx, aquaScannerAccount, asa.AquaScannerAccountStatus{State: "Failed", Message: errorMessage}, r.Status(), ctrl.Log)

		if updateErr != nil {
			return ctrl.Result{Requeue: true}, updateErr
		}

		err := errors.NewUnauthorized(errorMessage)

		aquaUrl := os.Getenv("AQUA_URL")
		aquaUsername := os.Getenv("AQUA_USER")

		ctrl.Log.Error(err, "AquaScannerAccount did not authenticate with Aqua. This is required for reconciliation. Does the manager have the correct credentials to authenticate with Aqua ( url: "+aquaUrl+" user: "+aquaUsername+")?")
		return ctrl.Result{Requeue: false}, err
	}

	if aquaScannerAccount.Status.State != "Complete" {

		newStatus := asa.AquaScannerAccountStatus{State: "Running", Message: "Beginning reconcilliation"}

		// if this is the first time reconciling the CR the currentState will be empty and needs to be initialized
		if aquaScannerAccount.Status.CurrentState == (asa.AquaScannerAccountAquaObjectState{}) {
			newStatus.CurrentState = asa.AquaScannerAccountAquaObjectState{ApplicationScope: "Not Created", PermissionSet: "Not Created", Role: "Not Created", User: "Not Created"}
		}

		updateErr := utils.UpdateStatus(ctx, aquaScannerAccount, newStatus, r.Status(), ctrl.Log)

		if updateErr != nil {
			return ctrl.Result{Requeue: true}, updateErr
		}

		applicationScope := utils.ApplicationScope{
			Name:               aquaScannerAccountName,
			Description:        "Application Scoped to " + namespacePrefix + "-* and DockerHub only.",
			TechnicalLeadEmail: "",
			NamespacePrefix:    namespacePrefix,
		}

		permissionSet := utils.PermissionSet{
			Name:               aquaScannerAccountName,
			Description:        "Permission Set for AquaScannerAccount: UI read and scan read/write priviledges only",
			TechnicalLeadEmail: "",
		}

		role := utils.Role{
			Name:             aquaScannerAccountName,
			Description:      "AquaScannerAccount created Role to allow Scanning of resources scoped to " + namespacePrefix + "-* and DockerHub only.",
			ApplicationScope: applicationScope,
			PermissionSet:    permissionSet,
		}

		if aquaScannerAccount.Status.CurrentState.ApplicationScope != "Created" {
			applicationScopeErr := utils.CreateAquaApplicationScope(ctrl.Log, applicationScope)

			if applicationScopeErr != nil {
				ctrl.Log.Error(applicationScopeErr, "Failed to create application scope")

				newCurrentState := aquaScannerAccount.Status.CurrentState
				newCurrentState.ApplicationScope = "Not Created"
				newStatus := asa.AquaScannerAccountStatus{State: "Failed", Message: "Reconcilliation failed. Was unable to create application scope. Will re-attempt.", CurrentState: newCurrentState}

				updateErr := utils.UpdateStatus(ctx, aquaScannerAccount, newStatus, r.Status(), ctrl.Log)

				if updateErr != nil {
					return ctrl.Result{Requeue: true}, updateErr
				}

				return ctrl.Result{Requeue: true}, applicationScopeErr
			} else {
				newCurrentState := aquaScannerAccount.Status.CurrentState
				newCurrentState.ApplicationScope = "Created"
				newStatus := asa.AquaScannerAccountStatus{CurrentState: newCurrentState}

				updateErr := utils.UpdateStatus(ctx, aquaScannerAccount, newStatus, r.Status(), ctrl.Log)

				if updateErr != nil {
					return ctrl.Result{Requeue: true}, updateErr
				}
			}

		}

		if aquaScannerAccount.Status.CurrentState.Role != "Created" {
			roleErr := utils.CreateAquaRole(ctrl.Log, role)

			if roleErr != nil {
				ctrl.Log.Error(roleErr, "Failed to create role")

				newCurrentState := aquaScannerAccount.Status.CurrentState
				newCurrentState.Role = "Not Created"
				newStatus := asa.AquaScannerAccountStatus{State: "Failed", Message: "Reconcilliation failed. Was unable to create role. Will re-attempt.", CurrentState: newCurrentState}

				updateErr := utils.UpdateStatus(ctx, aquaScannerAccount, newStatus, r.Status(), ctrl.Log)

				if updateErr != nil {
					return ctrl.Result{Requeue: true}, updateErr
				}

				return ctrl.Result{Requeue: true}, roleErr
			} else {
				newCurrentState := aquaScannerAccount.Status.CurrentState
				newCurrentState.Role = "Created"
				newStatus := asa.AquaScannerAccountStatus{CurrentState: newCurrentState}

				updateErr := utils.UpdateStatus(ctx, aquaScannerAccount, newStatus, r.Status(), ctrl.Log)

				if updateErr != nil {
					return ctrl.Result{Requeue: true}, updateErr
				}
			}
		}

		if aquaScannerAccount.Status.CurrentState.PermissionSet != "Created" {
			permissionSetErr := utils.CreateAquaPermissionSet(ctrl.Log, permissionSet)

			if permissionSetErr != nil {
				ctrl.Log.Error(permissionSetErr, "Failed to create role")

				newCurrentState := aquaScannerAccount.Status.CurrentState
				newCurrentState.PermissionSet = "Not Created"
				newStatus := asa.AquaScannerAccountStatus{State: "Failed", Message: "Reconcilliation failed. Was unable to create role. Will re-attempt.", CurrentState: newCurrentState}

				updateErr := utils.UpdateStatus(ctx, aquaScannerAccount, newStatus, r.Status(), ctrl.Log)

				if updateErr != nil {
					return ctrl.Result{Requeue: true}, updateErr
				}

				return ctrl.Result{Requeue: true}, permissionSetErr
			} else {
				newCurrentState := aquaScannerAccount.Status.CurrentState
				newCurrentState.PermissionSet = "Created"
				newStatus := asa.AquaScannerAccountStatus{CurrentState: newCurrentState}

				updateErr := utils.UpdateStatus(ctx, aquaScannerAccount, newStatus, r.Status(), ctrl.Log)

				if updateErr != nil {
					return ctrl.Result{Requeue: true}, updateErr
				}
			}
		}

		if aquaScannerAccount.Status.CurrentState.User != "Created" {

			var pwd *string

			if aquaScannerAccount.Status.AccountSecret != "" {
				pwd = &aquaScannerAccount.Status.AccountSecret
			} else {
				pw := utils.GeneratePassword(16, true, true, true)
				pwd = &pw
			}

			user := utils.User{
				Name:     aquaScannerAccountName,
				Password: *pwd,
				Role:     role,
			}
			// update asa status before creating user just incase user creation fails, the user will be recreated with the
			// same password as before
			updateErr := utils.UpdateStatus(ctx, aquaScannerAccount, asa.AquaScannerAccountStatus{AccountName: user.Name, AccountSecret: user.Password}, r.Status(), ctrl.Log)
			if updateErr != nil {
				return ctrl.Result{Requeue: true}, updateErr
			}

			userErr := utils.CreateAquaAccount(ctrl.Log, user)

			if userErr != nil {
				ctrl.Log.Error(userErr, "Failed to create user")
				newCurrentState := aquaScannerAccount.Status.CurrentState
				newCurrentState.User = "Not Created"
				newStatus := asa.AquaScannerAccountStatus{State: "Failed", Message: "Reconcilliation failed. Was unable to create aqua user. Will re-attempt.", CurrentState: newCurrentState}

				aquaScannerAccount.Status.State = "Failed"

				updateErr := utils.UpdateStatus(ctx, aquaScannerAccount, newStatus, r.Status(), ctrl.Log)

				if updateErr != nil {
					return ctrl.Result{Requeue: true}, updateErr
				}

				return ctrl.Result{Requeue: true}, userErr
			} else {
				newCurrentState := aquaScannerAccount.Status.CurrentState
				newCurrentState.User = "Created"
				newStatus := asa.AquaScannerAccountStatus{CurrentState: newCurrentState}

				updateErr := utils.UpdateStatus(ctx, aquaScannerAccount, newStatus, r.Status(), ctrl.Log)

				if updateErr != nil {
					return ctrl.Result{Requeue: true}, updateErr
				}
			}
		}

		// set status to Complete
		if aquaScannerAccount.Status.CurrentState == aquaScannerAccount.Status.DesiredState {
			newStatus := asa.AquaScannerAccountStatus{State: "Complete", Message: "Reconcilliation Successful!"}

			updateErr := utils.UpdateStatus(ctx, aquaScannerAccount, newStatus, r.Status(), ctrl.Log)
			if updateErr != nil {
				return ctrl.Result{Requeue: true}, updateErr
			}
		}

	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AquaScannerAccountReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&asa.AquaScannerAccount{}).
		Complete(r)
}

func (r *AquaScannerAccountReconciler) finalizeAquaScannerAccount(reqLogger *log.DelegatingLogger, m *asa.AquaScannerAccount, aquaScannerName string) error {

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

	delPermissionSetErr := utils.DeleteAquaPermissionSet(ctrl.Log, aquaScannerName)

	if delPermissionSetErr != nil {
		return delPermissionSetErr
	}

	reqLogger.Info("Successfully finalized AquaScannerAccount")
	return nil
}
