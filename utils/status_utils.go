package utils

import (
	"context"
	"time"

	asa "github.com/bcgov-platform-services/aqua-scan-cli-operator/api/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

/*
	Returns the desired state if needed, the boolean return allows calling function to decide if it should update status or now
*/
func SetDesiredStateIfNeeded(state asa.AquaScannerAccountAquaObjectState) (asa.AquaScannerAccountAquaObjectState, bool) {
	if state == (asa.AquaScannerAccountAquaObjectState{}) {
		return asa.AquaScannerAccountAquaObjectState{ApplicationScope: asa.Created.String(), PermissionSet: asa.Created.String(), Role: asa.Created.String(), User: asa.Created.String()}, true
	}
	return asa.AquaScannerAccountAquaObjectState{}, false
}

/*
	A struct merge with the caveat desiredStatus is not merged and remains static from the old status since it should never change
*/
func MergeStatus(oldStatus asa.AquaScannerAccountStatus, newStatus asa.AquaScannerAccountStatus) asa.AquaScannerAccountStatus {
	var mergedStatus asa.AquaScannerAccountStatus

	if newStatus.State != "" {
		mergedStatus.State = newStatus.State
	} else {
		mergedStatus.State = oldStatus.State
	}

	if newStatus.AccountName != "" {
		mergedStatus.AccountName = newStatus.AccountName
	} else {
		mergedStatus.AccountName = oldStatus.AccountName
	}

	if newStatus.AccountSecret != "" {
		mergedStatus.AccountSecret = newStatus.AccountSecret
	} else {
		mergedStatus.AccountSecret = oldStatus.AccountSecret
	}

	if newStatus.Message != "" {
		mergedStatus.Message = newStatus.Message
	} else {
		mergedStatus.Message = oldStatus.Message
	}

	// because updateStatus doesn't necessarily need to provide a new status with the currentState or desiredState
	// check if the newStatus is updating currentState, if so use it, otherwise, use the old status
	if newStatus.CurrentState != (asa.AquaScannerAccountAquaObjectState{}) {
		mergedStatus.CurrentState = newStatus.CurrentState
	} else {
		mergedStatus.CurrentState = oldStatus.CurrentState
	}

	if newStatus.DesiredState != (asa.AquaScannerAccountAquaObjectState{}) {
		mergedStatus.DesiredState = newStatus.DesiredState
	} else {
		mergedStatus.DesiredState = oldStatus.DesiredState
	}

	return mergedStatus
}

func UpdateStatus(ctx context.Context, account *asa.AquaScannerAccount, newStatus asa.AquaScannerAccountStatus, clientWriter client.StatusWriter, reqLogger *log.DelegatingLogger) error {

	mergedStatus := MergeStatus(account.Status, newStatus)
	mergedStatus.Timestamp = v1.Timestamp{Seconds: time.Now().Unix(), Nanos: int32(time.Now().UnixNano())}
	account.Status = mergedStatus

	// to prevent race condition between rapid updates
	time.Sleep(200 * time.Millisecond)
	err := clientWriter.Update(ctx, account)

	if err != nil {
		reqLogger.Error(err, "Failed to update aquaScannerAccount status")
	}

	return err
}
