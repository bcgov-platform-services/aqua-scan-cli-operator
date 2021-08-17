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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

type ApplicationScope struct {
	Name               string
	NamespacePrefix    string
	Description        string
	TechnicalLeadEmail string
}

type Role struct {
	Name        string
	Description string
	ApplicationScope
}

type User struct {
	Name string
	Role
	Password string
}

type aquaResponseJson struct {
	Message string `json:"message"`
}

//+kubebuilder:rbac:groups=mamoa.devops.gov.bc.ca.devops.gov.bc.ca,resources=aquascanneraccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=mamoa.devops.gov.bc.ca.devops.gov.bc.ca,resources=aquascanneraccounts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=mamoa.devops.gov.bc.ca.devops.gov.bc.ca,resources=aquascanneraccounts/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// - finish create/delete func for:
//    - applicationscope
//    - role
// - finish finalizer deletion of all aqua objects
// - finish error handling for main reconcilliation
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *AquaScannerAccountReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// Fetch the aqua scanner account instance
	aquaScannerAccount := &mamoadevopsgovbccav1alpha1.AquaScannerAccount{}
	err := r.Get(ctx, req.NamespacedName, aquaScannerAccount)
	aquaScannerAccountName := "ScannerCLI_" + req.Namespace

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
			return ctrl.Result{}, updateErr
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
		accountExists, aquaResErr := doesAquaAccountAlreadyExist(ctrl.Log, aquaScannerAccountName)

		if aquaResErr != nil {
			return ctrl.Result{}, aquaResErr
		}

		if accountExists {
			// only need to delete account becuase we are regenerating a password
			delErr := deleteAquaAccount(ctrl.Log, aquaScannerAccountName)
			if delErr != nil {
				return ctrl.Result{}, delErr
			}
		}
		namespacePrefix := strings.TrimSuffix(req.Namespace, "-test")

		var namespace corev1.Namespace
		namespaceErr := r.Get(ctx, req.NamespacedName, &namespace)

		if namespaceErr != nil {

		}

		contactAnnotation := namespace.Annotations["contacts"]

		technicalLeadEmail := utils.GetTechnicalContactFromAnnotation(contactAnnotation)

		applicationScope := ApplicationScope{
			Name:               aquaScannerAccountName,
			Description:        "Scanner scoped to " + namespacePrefix + "-* and DockerHub only.",
			TechnicalLeadEmail: technicalLeadEmail,
			NamespacePrefix:    namespacePrefix,
		}

		role := Role{
			Name:             aquaScannerAccountName,
			Description:      "AquaScannerAccount created Role to allow Scanning of resources scoped to " + namespacePrefix + "-* and DockerHub only.",
			ApplicationScope: applicationScope,
		}

		applicationScopeErr := createAquaApplicationScope(ctrl.Log, applicationScope)

		if applicationScopeErr != nil {

		}

		roleErr := createAquaRole(ctrl.Log, role)

		if roleErr != nil {

		}

		config := generator.Config{
			Length:                     16,
			IncludeSymbols:             false,
			IncludeNumbers:             true,
			IncludeLowercaseLetters:    true,
			IncludeUppercaseLetters:    true,
			ExcludeSimilarCharacters:   true,
			ExcludeAmbiguousCharacters: true,
		}
		g, _ := generator.New(&config)

		pwd, _ := g.Generate()

		user := User{
			Name:     aquaScannerAccountName,
			Password: *pwd,
			Role:     role,
		}

		userErr := createAquaAccount(ctrl.Log, user)

		if userErr != nil {

		}

		// set status to Complete
		aquaScannerAccount.Status.CurrentState = "Failure"

		aquaScannerAccount.Status.Timestamp = v1.Timestamp{Seconds: time.Now().Unix(), Nanos: int32(time.Now().UnixNano())}
		updateErr = r.Status().Update(ctx, aquaScannerAccount)

		if updateErr != nil {
			ctrl.Log.Error(updateErr, "Failed to update aquaScannerAccount status")
			return ctrl.Result{}, updateErr
		}

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

	delAcctErr := deleteAquaAccount(ctrl.Log, aquaScannerName)

	if delAcctErr != nil {
		return delAcctErr
	}
	// delete application scope
	// delete role

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
	qualifiedResource := schema.GroupResource{Group: mamoadevopsgovbccav1alpha1.GroupVersion.Group, Resource: "AquaScannerAccount"}

	return false, errors.NewGenericServerResponse(res.StatusCode, "GET", qualifiedResource, "Generic Error", errorMsg, 10, true)
}

func deleteAquaAccount(reqLogger *log.DelegatingLogger, accountName string) error {
	reqLogger.Info("Deleting user %v in aqua", accountName)
	reqUrl := os.Getenv("AQUA_URL") + "/users/" + accountName
	client := &http.Client{}
	req, _ := http.NewRequest("DELETE", reqUrl, nil)
	req.Header.Set("Authorization", "Bearer "+os.Getenv("AQUA_SECRET"))
	req.Header.Set("Accept", "application/json")

	res, err := client.Do(req)

	if err != nil {
		reqLogger.Error(err, "Failed request to DELETE %v from aqua", accountName)
		return err
	}

	if res.StatusCode != 204 {
		reqLogger.Error(err, "Failed to DELETE %v from aqua", accountName)
		return errors.NewBadRequest("Failed to DELETE user from aqua")
	}
	reqLogger.Info("User %v deleted", accountName)
	return nil
}

// TODO
// func deleteAquaApplicationScope() error {}
// func deleteAquaRole() error {}

func createAquaRole(reqLogger *log.DelegatingLogger, role Role) error {
	reqLogger.Info("Creating Role %v in aqua", role.Name)

	b, fileErr := ioutil.ReadFile("../templates/Role.json.tmpl")

	if fileErr != nil {
		reqLogger.Error(fileErr, "Failed to read template file Role.json.tmpl")
		return fileErr
	}

	ut, templateErr := template.New("Role").Parse(string(b))

	if templateErr != nil {
		reqLogger.Error(templateErr, "Failed to parse template file Role.json.tmpl")
		return templateErr
	}

	var roleBuffer bytes.Buffer
	ut.Execute(&roleBuffer, role)

	reqUrl := os.Getenv("AQUA_URL") + "/access_management/roles"
	client := &http.Client{}
	req, _ := http.NewRequest("POST", reqUrl, &roleBuffer)
	req.Header.Set("Authorization", "Bearer "+os.Getenv("AQUA_SECRET"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	res, err := client.Do(req)

	if err != nil {
		reqLogger.Error(err, "Failed request to POST to /access_management/roles in aqua")
		return err
	}
	defer res.Body.Close()

	var jsonData aquaResponseJson
	body, _ := ioutil.ReadAll(res.Body)

	json.Unmarshal(body, &jsonData)

	if res.StatusCode == 404 && strings.Contains(jsonData.Message, "role "+role.Name+" already exists") || res.StatusCode == 201 {
		return nil
	} else {
		e := errors.NewBadRequest(fmt.Sprintf("Error: Could not create role, the response status from aqua was %v", res.StatusCode))

		reqLogger.Error(e, "Unable to create Role")
		return e
	}
}

func createAquaApplicationScope(reqLogger *log.DelegatingLogger, appScope ApplicationScope) error {
	reqLogger.Info("Creating applicationScope %v-* in aqua", appScope.NamespacePrefix)

	b, fileErr := ioutil.ReadFile("../templates/ApplicationScope.json.tmpl")

	if fileErr != nil {
		reqLogger.Error(fileErr, "Failed to read template file ApplicationScope.json.tmpl")
		return fileErr
	}

	ut, templateErr := template.New("ApplicationScope").Parse(string(b))

	if templateErr != nil {
		reqLogger.Error(templateErr, "Failed to parse template file ApplicationScope.json.tmpl")
		return templateErr
	}

	var appScopeBuffer bytes.Buffer
	ut.Execute(&appScopeBuffer, appScope)

	reqUrl := os.Getenv("AQUA_URL") + "/access_management/scopes"
	client := &http.Client{}
	req, _ := http.NewRequest("POST", reqUrl, &appScopeBuffer)
	req.Header.Set("Authorization", "Bearer "+os.Getenv("AQUA_SECRET"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	res, err := client.Do(req)

	if err != nil {
		reqLogger.Error(err, "Failed request to POST to /access_management/scopes in aqua")
		return err
	}
	defer res.Body.Close()

	var jsonData aquaResponseJson
	body, _ := ioutil.ReadAll(res.Body)

	json.Unmarshal(body, &jsonData)

	if res.StatusCode == 404 && strings.Contains(jsonData.Message, "application scope "+appScope.Name+" already exists") || res.StatusCode == 201 {
		return nil
	} else {
		e := errors.NewBadRequest(fmt.Sprintf("Error: Could not create ApplicationScope, the response status from aqua was %v", res.StatusCode))

		reqLogger.Error(e, "Unable to create ApplicationScope")
		return e
	}
}

func createAquaAccount(reqLogger *log.DelegatingLogger, user User) error {
	reqLogger.Info("Creating user %v in aqua", user.Name)

	b, fileErr := ioutil.ReadFile("../templates/User.json.tmpl")

	if fileErr != nil {
		reqLogger.Error(fileErr, "Failed to read template file User.json.tmpl")
		return fileErr
	}

	ut, templateErr := template.New("User").Parse(string(b))

	if templateErr != nil {
		reqLogger.Error(templateErr, "Failed to parse template file User.json.tmpl")
		return templateErr
	}

	var userBuffer bytes.Buffer
	ut.Execute(&userBuffer, user)

	reqUrl := os.Getenv("AQUA_URL") + "/users/"
	client := &http.Client{}
	req, _ := http.NewRequest("POST", reqUrl, &userBuffer)
	req.Header.Set("Authorization", "Bearer "+os.Getenv("AQUA_SECRET"))
	req.Header.Set("Content-Type", "application/json")

	res, err := client.Do(req)

	if err != nil {
		reqLogger.Error(err, "Failed request to POST %v from aqua", user.Name)
		return err
	}

	if res.StatusCode != 204 {
		reqLogger.Error(err, "Failed to POST %v from aqua. Status code is %v", user.Name, res.StatusCode)
		return errors.NewBadRequest("Failed to POST user from aqua")
	}
	reqLogger.Info("User %v created in aqua", user.Name)
	return nil
}
