/*


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
	"fmt"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/danieldorado/redis-operator/api/v1alpha1"
	redisv1alpha1 "github.com/danieldorado/redis-operator/api/v1alpha1"
)

const (
	// TODO Change this
	imageRedisOperator = "danieldorado/redisop"
	imageRedis         = "danieldorado/redis"
)

// RedisReconciler reconciles a Redis object
type RedisReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=redis.danieldorado.github.io,resources=redis,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=redis.danieldorado.github.io,resources=redis/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=redis.danieldorado.github.io,resources=redis/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// Creates or reconciles a StatefulSet (the redis cluster) and a Deployment (the redis operator).
func (r *RedisReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("redis", req.NamespacedName)

	// Fetch the Redis instance
	redis := &redisv1alpha1.Redis{}
	err := r.Get(ctx, req.NamespacedName, redis)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use
			// final

			// Return and don't requeue
			log.Info("Redis resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		//  Error reading the object - requeue the request.
		log.Error(err, "Failed to get Redis - reque the request.")
		return ctrl.Result{}, err
	}

	// Does the StatefulSet exist?
	ctxtError := []interface{}{"Redis.Name", redis.Name, "Redis.Namespace", redis.Namespace, "Redis.Size",
		redis.Spec.Size}
	found := &appsv1.StatefulSet{}
	err = r.Get(ctx, types.NamespacedName{Name: redis.Name, Namespace: redis.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		ss := r.statefulSetForRedis(redis)
		log.Info("Creating a new StatefulSet", ctxtError...)
		// We can change create for path. Is it better?
		err = r.Create(ctx, ss)
		if err != nil {
			errorSS := []interface{}{"StatefulSet.Name", ss.Name, "StatefulSet.Namespace", ss.Namespace}
			errorSS = append(errorSS, ctxtError...)
			log.Error(err, "Failed to create new StatefulSet", errorSS)
			return ctrl.Result{}, err
		}
		// StatefulSet created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get StatefulSet")
		return ctrl.Result{}, err
	}

	// Redis Size == StatefulSet Replicas?
	ctxtError = append(ctxtError, []interface{}{"StatefulSet.Name", found.Name, "StatefulSet.Replicas", *found.Spec.Replicas}...)
	if *found.Spec.Replicas != redis.Spec.Size {
		// applyOpts := []client.PatchOption{client.ForceOwnership, client.FieldOwner("guestbook")}
		log.Info("Fixing replicas mismatch:", ctxtError...)
		*found.Spec.Replicas = redis.Spec.Size
		// *&found.ManagedFields = nil
		// if err := r.Patch(ctx, found, client.Apply, []client.PatchOption{}...); err != nil {
		// options := client.PatchOption{}
		// "PatchOptions.meta.k8s.io \"\" is invalid: fieldManager: Required value:
		//   is required for apply patch"
		// "error": "metadata.managedFields must be nil"}
		// option := &client.PatchOptions{FieldManager: "main"}
		// if err := r.Patch(ctx, found, client.Apply, option); err != nil {
		if err := r.Update(ctx, found); err != nil {
			log.Error(err, "Failed fixing replicas mismatch", ctxtError...)
			return ctrl.Result{}, err
		}

		err := fmt.Errorf("Mismatch StatefulSet replicas and Redis size: %+v", ctxtError...)
		return ctrl.Result{}, err
	}

	// Are StatefulSet Pods OK?
	if found.Status.ReadyReplicas != redis.Spec.Size {
		// log.Info("Waiting for pods to be ready...", ctxtError...)
		err := fmt.Errorf("Waiting for Redis pods: %v", ctxtError...)
		return ctrl.Result{}, err
	}

	// The Redis Pod are in cluster?
	log.Info("Begining the Redis operations...", ctxtError...)

	// The cluster is Ok?
	// Check cluster. Configure cluster.
	// is configured??? Ask to -0... $ cluster info serviceName -0

	// The Slots are assigned?

	/*
		if *found.Spec.Replicas != size {
			found.Spec.Replicas = &size
			err = r.Update(ctx, found)
			if err != nil {
				log.Error(err, "Failed to update StatefulSet", "StatefulSet.Namespace", found.Namespace, "StatefulSet.Name", found.Name)
				return ctrl.Result{}, err
			}
			// Spec updated - return and requeue
			return ctrl.Result{Requeue: true}, nil
		}
	*/
	// Update the Redis status with the pod names
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(redis.Namespace),
		client.MatchingLabels(labelsForRedis(redis.Name)),
	}
	if err = r.List(ctx, podList, listOpts...); err != nil {
		log.Error(err, "Failed to list pods", "Redis.Namespace", redis.Namespace, "Redis.Name", redis.Name)
		return ctrl.Result{}, err
	}
	podNames := getPodNames(podList.Items)

	// Update status.Nodes if needed
	if !reflect.DeepEqual(podNames, redis.Status.Nodes) {
		redis.Status.Nodes = podNames
		err := r.Status().Update(ctx, redis)
		if err != nil {
			log.Error(err, "Failed to update Memcached status")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager wath to watch
func (r *RedisReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// owns is like Watches(&source.Kind{Type: <ForType-forInput>}, &handler.EnqueueRequestForOwner{OwnerType: apiType, IsController: true})
	// But the StatefulSet Pods??? We don't care about Pods, StatefulSet only *Redis* objects.
	return ctrl.NewControllerManagedBy(mgr).
		For(&redisv1alpha1.Redis{}).
		Owns(&appsv1.StatefulSet{}).
		Complete(r)
	/* TODO: Watch other objects that does not owns: https://pres.metamagical.dev/kubecon-us-2019/#slide-33
	return ctrl.NewControllerManagedBy(mgr).
		For(&redisv1alpha1.Redis{}).
		Owns(&appsv1.StatefulSet{}).
		// Watches.... (example booksUsingRedis...)
		Complete(r)
	*/
}

// TODO fill this
// statefulSetForRedis returns the redis operator statefulSet which does the cluster operations.
func (r *RedisReconciler) statefulSetForRedis(redis *v1alpha1.Redis) *appsv1.StatefulSet {
	ls := labelsForRedis(redis.Name)
	replicas := int32(redis.Spec.Size)

	ss := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      redis.Name,
			Namespace: redis.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image:   "redis:5.0.10",
						Name:    "redis",
						Command: []string{},
						Ports:   []corev1.ContainerPort{},
					}},
				}},
		},
	}
	ctrl.SetControllerReference(redis, ss, r.Scheme)
	return ss
}

// TODO fill this
func labelsForRedis(name string) map[string]string {
	return map[string]string{"app": "redis", "redis_cr": name}
}

// TODO fill this
func getPodNames(pods []corev1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}
