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

// PodBackupReconciler reconciles a PodBackup object
type PodBackupReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	BackupFinalizer = "kop.hanshal.com/podbackup-finalizer"
)

// +kubebuilder:rbac:groups=backup.kop.hanshal.com,resources=podbackups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=backup.kop.hanshal.com,resources=podbackups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=backup.kop.hanshal.com,resources=podbackups/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PodBackup object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *PodBackupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log.Log.Info("Starting reconciliation", "backupName", req.Name)

	var bkpod v1delta1.PodBackup
	if err := r.Get(ctx, req.NamespacedName, &bkpod); err != nil {
		if errors.IsNotFound(err) {
			log.Log.Info("Backup pod deleted", "name", req.Name, "namespace", req.Namespace)
			return ctrl.Result{}, nil
		}
		log.Log.Error(err, "Error fetching backup pod")
		return ctrl.Result{}, err
	}

	if bkpod.DeletionTimestamp != nil {
		if checkFinalizers(bkpod) {
			if err := removeBackup(bkpod); err != nil {
				log.Log.Error(err, "Error in removing backup file")
				return ctrl.Result{}, err
			}

			removeRestores(bkpod)

			removeFinalizers(&bkpod)
			if err := r.Update(ctx, &bkpod); err != nil {
				log.Log.Error(err, "Error updating backup pod after finalizer removal")
				return ctrl.Result{}, err
			}

			log.Log.Info("Finalizers removed and backup pod deletion handled", "name", req.Name, "namespace", req.Namespace)
		}
		return ctrl.Result{}, nil
	}

	if !checkFinalizers(bkpod) {
		addFinalizers(&bkpod)
		if err := r.Update(ctx, &bkpod); err != nil {
			log.Log.Error(err, "Error updating backup pod after finalizer removal")
			return ctrl.Result{}, err
		}
	}

	res, err := r.isBackup(bkpod)
	if res != 0 {
		if err != nil {
			return ctrl.Result{}, err
		} else {
			return ctrl.Result{RequeueAfter: res}, nil
		}
	}

	err = r.backupPod(ctx, &bkpod)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Log.Info("pod not present", "podName", bkpod.Spec.PodName, "podNamespace", bkpod.Spec.Namespace)
			return ctrl.Result{}, nil
		}
		log.Log.Error(err, "Error in pod backup")
		return ctrl.Result{}, err
	}

	bkpod.Status.LastBackup = time.Now().Format(time.RFC3339)
	if err := r.Status().Update(ctx, &bkpod); err != nil {
		log.Log.Error(err, "Error updating backup pod status")
		return ctrl.Result{}, err
	}

	log.Log.Info("Last backup updated for pod", "backupName", bkpod.Name, "lastBackup", bkpod.Status.LastBackup)

	return ctrl.Result{}, nil
}

func (r *PodBackupReconciler) isBackup(bkpod v1delta1.PodBackup) (time.Duration, error) {
	podBackupTime := bkpod.Spec.PodBackupTime
	if podBackupTime == 0 {
		log.Log.Info("PodBackupTime not set, skipping backup")
		return 0, fmt.Errorf("pod backup time not set")
	}

	lastBackupTime := bkpod.Status.LastBackup
	if lastBackupTime == "" {
		lastBackupTime = "1970-01-01T00:00:00Z"
	}

	lastBackup, err := time.Parse(time.RFC3339, lastBackupTime)
	if err != nil {
		log.Log.Error(err, "error parsing last backup time")
		return 0, err
	}

	currentTime := time.Now()
	timeSinceLastBackup := currentTime.Sub(lastBackup)

	remainingTime := time.Duration(podBackupTime)*time.Second - timeSinceLastBackup

	if timeSinceLastBackup < time.Duration(podBackupTime)*time.Second {
		log.Log.Info("Skipping backup, not enough time has passed", "lastBackup", lastBackupTime, "nextBackup", currentTime.Add(remainingTime))

		return remainingTime, nil
	}

	return 0, fmt.Errorf("skiping the backup")
}

func (r *PodBackupReconciler) backupPod(ctx context.Context, bkpod *v1delta1.PodBackup) error {
	var pod corev1.Pod
	if err := r.Get(ctx, client.ObjectKey{
		Name:      bkpod.Spec.PodName,
		Namespace: bkpod.Spec.Namespace,
	}, &pod); err != nil {
		return err
	}

	backupDir := bkpod.Spec.PodBackupLocation
	if err := os.MkdirAll(backupDir, os.ModePerm); err != nil {
		return err
	}

	storePod, err := encodetoBytes(&pod)
	if err != nil {
		return err
	}

	filePath := filepath.Join(backupDir, fmt.Sprintf("%s-%s.yaml", bkpod.Name, bkpod.Namespace))
	if err := os.WriteFile(filePath, storePod, os.ModePerm); err != nil {
		return err
	}

	bkpod.Status.LastBackup = time.Now().UTC().Format(time.RFC3339)

	return nil
}

func removeBackup(bkpod v1delta1.PodBackup) error {
	filePath := filepath.Join(bkpod.Spec.PodBackupLocation, fmt.Sprintf("%s-%s.yaml", bkpod.Name, bkpod.Namespace))
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			log.Log.Info("backup not present", "filepath", filePath)
			return nil
		}
		return err
	}
	if err := os.Remove(filePath); err != nil {
		return err
	}
	return nil
}

func checkFinalizers(bkpod v1delta1.PodBackup) bool {
	for _, f := range bkpod.Finalizers {
		if f == BackupFinalizer {
			return true
		}
	}
	return false
}

func removeFinalizers(bkpod *v1delta1.PodBackup) {
	newFinalizers := []string{}
	for _, f := range bkpod.Finalizers {
		if f != BackupFinalizer {
			newFinalizers = append(newFinalizers, f)
		}
	}
	bkpod.Finalizers = newFinalizers
}

func addFinalizers(bkpod *v1delta1.PodBackup) {
	bkpod.Finalizers = append(bkpod.Finalizers, BackupFinalizer)
}

func encodetoBytes(pod *corev1.Pod) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(&pod); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func removeRestores(bkpod v1delta1.PodBackup) {
	log.Log.Info("this removes all the podRestore resources linked to the current podBackup", "podBackup", bkpod)
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodBackupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1delta1.PodBackup{}).
		Named("podbackup").
		Complete(r)
}
