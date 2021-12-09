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

package v1alpha1

import (
	v1 "github.com/bcgov-platform-services/aqua-scan-cli-operator/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// ConvertTo converts this AquaScannerAccount to the Hub version (v1).
// src = v1alpha1
// dst = v1
func (src *AquaScannerAccount) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1.AquaScannerAccount)
	dst.Name = src.Name
	dst.Namespace = src.Namespace
	dst.UID = src.UID
	state := src.Status.CurrentState
	dst.Status.Timestamp = src.Status.Timestamp
	dst.Status.AccountName = src.Status.AccountName
	dst.Status.AccountSecret = src.Status.AccountSecret
	dst.Status.State = state
	// ***not setting DesiredState as controller handles that and updates accordingly

	if state == "Complete" {
		dst.Status.CurrentState = v1.AquaScannerAccountAquaObjectState{ApplicationScope: v1.Created.String(), Role: v1.Created.String(), PermissionSet: v1.Created.String(), User: v1.Created.String()}
		dst.Status.Message = "Reconcilliation Successful!"
	} else if state == "Running" {
		dst.Status.Message = "Beginning reconcilliation"
		dst.Status.CurrentState = v1.AquaScannerAccountAquaObjectState{ApplicationScope: v1.NotCreated.String(), Role: v1.NotCreated.String(), PermissionSet: v1.NotCreated.String(), User: v1.NotCreated.String()}
	} else if state == "Failed" {
		dst.Status.Message = "Reconcilliation Failed"
		dst.Status.CurrentState = v1.AquaScannerAccountAquaObjectState{ApplicationScope: v1.NotCreated.String(), Role: v1.NotCreated.String(), PermissionSet: v1.NotCreated.String(), User: v1.NotCreated.String()}
	} else {
		dst.Status.Message = "Reconcilliation not complete"
		dst.Status.CurrentState = v1.AquaScannerAccountAquaObjectState{ApplicationScope: v1.NotCreated.String(), Role: v1.NotCreated.String(), PermissionSet: v1.NotCreated.String(), User: v1.NotCreated.String()}
	}

	return nil
}

// ConvertFrom converts from the Hub version (v1) to this version v1alpha1.
func (dst *AquaScannerAccount) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1.AquaScannerAccount)
	dst.Name = src.Name
	dst.Namespace = src.Namespace
	dst.UID = src.UID
	dst.Status.Timestamp = src.Status.Timestamp
	dst.Status.AccountName = src.Status.AccountName
	dst.Status.AccountSecret = src.Status.AccountSecret
	dst.Status.CurrentState = src.Status.State
	return nil
}
