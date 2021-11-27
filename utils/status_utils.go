package utils

import (
	"context"
	"time"

	mamoadevopsgovbccav1alpha1 "github.com/bcgov-platform-services/aqua-scan-cli-operator/api/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

/*
	A struct merge with the caveat desiredStatus is not merged and remains static from the old status since it should never change
*/
func MergeStatus(oldStatus mamoadevopsgovbccav1alpha1.AquaScannerAccountStatus, newStatus mamoadevopsgovbccav1alpha1.AquaScannerAccountStatus) mamoadevopsgovbccav1alpha1.AquaScannerAccountStatus {
	var mergedStatus mamoadevopsgovbccav1alpha1.AquaScannerAccountStatus

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

	if newStatus.CurrentState != (mamoadevopsgovbccav1alpha1.AquaScannerAccountAquaObjectState{}) {
		mergedStatus.CurrentState = newStatus.CurrentState
	} else {
		mergedStatus.CurrentState = oldStatus.CurrentState
	}
	// desired state should remain static
	mergedStatus.DesiredState = oldStatus.DesiredState

	return mergedStatus
}

func UpdateStatus(ctx context.Context, account *mamoadevopsgovbccav1alpha1.AquaScannerAccount, newStatus mamoadevopsgovbccav1alpha1.AquaScannerAccountStatus, clientWriter client.StatusWriter, reqLogger *log.DelegatingLogger) error {

	mergedStatus := MergeStatus(account.Status, newStatus)
	mergedStatus.Timestamp = v1.Timestamp{Seconds: time.Now().Unix(), Nanos: int32(time.Now().UnixNano())}
	account.Status = mergedStatus

	err := clientWriter.Update(ctx, account)

	if err != nil {
		reqLogger.Error(err, "Failed to update aquaScannerAccount status")
	}

	return err
}
