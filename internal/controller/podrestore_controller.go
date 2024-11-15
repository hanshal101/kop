/*
Copyright 2024 hanshal101.

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

package controller

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/hanshal101/kOp/api/v1delta1"
)

// PodRestoreReconciler reconciles a PodRestore object
type PodRestoreReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=backup.kop.hanshal.com,resources=podrestores,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=backup.kop.hanshal.com,resources=podrestores/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=backup.kop.hanshal.com,resources=podrestores/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PodRestore object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *PodRestoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log.Log.Info("Starting reconciliation", "restoreName", req.Name)

	var rtPod v1delta1.PodRestore
	if err := r.Get(ctx, req.NamespacedName, &rtPod); err != nil {
		if errors.IsNotFound(err) {
			log.Log.Info("podRestore was deleted", "name", req.Name, "namespace", req.Namespace)
			return ctrl.Result{}, nil
		}
		log.Log.Error(err, "error in fetching podRestore")
		return ctrl.Result{}, err
	}

	var bkpod v1delta1.PodBackup
	bkpodKey := client.ObjectKey{
		Namespace: rtPod.Spec.BackupNamespace,
		Name:      rtPod.Spec.BackupName,
	}
	if err := r.Client.Get(ctx, bkpodKey, &bkpod); err != nil {
		if errors.IsNotFound(err) {
			log.Log.Info("the PodBackup resource was not found", "name", bkpod.Name, "namespace", bkpod.Namespace)
			return ctrl.Result{}, nil
		}
		log.Log.Error(err, "error in fetching the Podbackup")
		return ctrl.Result{}, err
	}

	if err := r.podrestore(ctx, bkpod, rtPod.Spec.AutoRestore); err != nil {
		log.Log.Error(err, "error in restoring pod")
		return ctrl.Result{}, err
	}

	if rtPod.Status.LastRestore != time.Now().UTC().Format(time.RFC3339) {
		rtPod.Status.LastRestore = time.Now().UTC().String()
		rtPod.Status.LastRestoreStatus = "SUCCESSFULL"
		patch := client.MergeFrom(rtPod.DeepCopy())
		if err := r.Status().Patch(ctx, &rtPod, patch); err != nil {
			log.Log.Error(err, "Error updating restore pod status")
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{RequeueAfter: 5 * time.Second}, nil

}

func (r *PodRestoreReconciler) podrestore(ctx context.Context, bkpod v1delta1.PodBackup, autoRestore bool) error {
	pod, err := getPod(bkpod.Spec.PodBackupLocation, bkpod.Name, bkpod.Namespace)
	if err != nil {
		return err
	}
	podKey := client.ObjectKey{
		Namespace: pod.Namespace,
		Name:      pod.Name,
	}

	if err := r.Client.Get(ctx, podKey, &pod); err != nil {
		if errors.IsNotFound(err) {
			log.Log.Info("pod not found; starting the restore if autoRestore=true")
			if autoRestore {
				if err := r.Client.Create(ctx, &pod, &client.CreateOptions{}); err != nil {
					return err
				}
				log.Log.Info("pod scheduled successfully", "podName", bkpod.Spec.PodName, "podNamespace", bkpod.Spec.Namespace)
				return nil
			} else {
				log.Log.Info("autoResore is false")
			}
		}
	}
	return nil
}

func getPod(path, name, namespace string) (corev1.Pod, error) {
	filePath := filepath.Join(path, fmt.Sprintf("%s-%s.yaml", name, namespace))
	bcode, err := os.ReadFile(filePath)
	if err != nil {
		return corev1.Pod{}, err
	}
	pod := decodeByte(bcode)
	if pod.Name == "" || pod.Namespace == "" {
		return corev1.Pod{}, fmt.Errorf("pod not present")
	}
	return pod, nil
}

func decodeByte(bcode []byte) corev1.Pod {
	var pod corev1.Pod
	var buf = bytes.NewBuffer(bcode)
	enc := gob.NewDecoder(buf)
	if err := enc.Decode(&pod); err != nil {
		return corev1.Pod{}
	}
	pod.ResourceVersion = ""
	return pod
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodRestoreReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1delta1.PodRestore{}).
		Named("podrestore").
		Complete(r)
}
