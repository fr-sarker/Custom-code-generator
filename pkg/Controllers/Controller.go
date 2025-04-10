package Controller

import (
	"context"
	"fmt"
	v1alpha1 "github.com/frsarker/crd/pkg/apis/frsarker.dev/v1"
	//v1 "github.com/frsarker/crd/pkg/generated/applyconfiguration/frsarker.dev/v1"
	clientset "github.com/frsarker/crd/pkg/generated/clientset/versioned"
	informers "github.com/frsarker/crd/pkg/generated/informers/externalversions/frsarker.dev/v1"
	songlisters "github.com/frsarker/crd/pkg/generated/listers/frsarker.dev/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	appsinformer "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	appslisters "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"log"
	"time"
)

type Controller struct {
	kubeclientset     kubernetes.Interface
	myclientset       clientset.Interface
	deploymentsLister appslisters.DeploymentLister
	deploymentsSynced cache.InformerSynced
	songLister        songlisters.SongLister
	songSynced        cache.InformerSynced
	workqueue         workqueue.TypedRateLimitingInterface[interface{}]
	recorder          record.EventRecorder
}

func NewController(
	kubeclientset kubernetes.Interface,
	myclientset clientset.Interface,
	deploymentInformer appsinformer.DeploymentInformer,
	songInformer informers.SongInformer) *Controller {

	c := &Controller{
		kubeclientset:     kubeclientset,
		myclientset:       myclientset,
		deploymentsLister: deploymentInformer.Lister(),
		deploymentsSynced: deploymentInformer.Informer().HasSynced,
		songLister:        songInformer.Lister(),
		songSynced:        songInformer.Informer().HasSynced,
		workqueue:         workqueue.NewTypedRateLimitingQueue[interface{}](workqueue.DefaultTypedControllerRateLimiter[interface{}]()),
	}
	log.Println("setting up event handlers")
	songInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueSong,
		UpdateFunc: func(old, new interface{}) {
			c.enqueSong(new)
		},
		DeleteFunc: func(obj interface{}) {
			c.enqueSong(obj)
		},
	})
	return c
}

func (c *Controller) enqueSong(obj interface{}) {
	log.Println("enqueuing song")
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Println("error getting key: ", err)
		return
	}
	c.workqueue.AddRateLimited(key)
}

func (c *Controller) Run(stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	log.Println("Starting Song controller")

	log.Println("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.deploymentsSynced, c.songSynced); !ok {
		return fmt.Errorf("failed to wait for cache to sync")
	}
	log.Println("Starting workers")

	log.Println("Starting workers")
	go wait.Until(c.runWorker, time.Second, stopCh)
	<-stopCh
	log.Println("Shutting down workers")
	return nil
}

func (c *Controller) runWorker() {
	for c.ProcessNextItem() {

	}
}

func (c *Controller) ProcessNextItem() bool {
	obj, shutdown := c.workqueue.Get()
	if shutdown {
		return false
	}
	fmt.Println("work queue:", obj)
	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		if key, ok = obj.(string); !ok {
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}

		if err := c.syncHandler(key); err != nil {
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		c.workqueue.Forget(obj)
		log.Printf("Successfully synced", "resourceName", key)
		return nil
	}(obj)
	if err != nil {
		utilruntime.HandleError(err)
		return true
	}
	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Song resource
// with the current status of the resource.
// implement the business logic here.

func (c *Controller) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}
	song, err := c.songLister.Songs(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("song '%s' in work queue no longer exists", key))
			return nil
		}
		return err
	}

	desireDeployment := NewDeployment(song)
	log.Printf("Deployments Created '%s'", desireDeployment.Name)
	// Get the deployment with the name specified in song.spec
	deployment, err := c.kubeclientset.AppsV1().Deployments(song.Namespace).Get(context.TODO(), desireDeployment.Name, metav1.GetOptions{})
	// If the resource doesn't exist, we'll create it
	if errors.IsNotFound(err) {
		_, err = c.kubeclientset.AppsV1().Deployments(song.Namespace).Create(context.TODO(), NewDeployment(song), metav1.CreateOptions{})
	}
	// If an error occurs during Get/Create, we'll requeue the item, so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		fmt.Errorf("error syncing song: %s", err.Error())
		return err
	} else {
		// Exists: update if needed (optional: deep equality check)
		desireDeployment.ResourceVersion = deployment.ResourceVersion
		_, err = c.kubeclientset.AppsV1().Deployments(song.Namespace).Update(context.TODO(), desireDeployment, metav1.UpdateOptions{})
		log.Printf("Deployment '%s' updated", desireDeployment.Name)
	}
	// If this number of the replicas on the song resource is specified, and the
	// number does not equal the current desired replicas on the Deployment, we
	// should update the Deployment resource.

	// Update Song status
	deployment, _ = c.kubeclientset.AppsV1().Deployments(song.Namespace).Get(context.TODO(), desireDeployment.Name, metav1.GetOptions{})
	err = c.updateSongCodeStatus(song, deployment)
	if err != nil {
		return err
	}
	// Finally, we update the status block of the song resource to reflect the
	// current state of the world

	serviceName := song.Name + "-service"
	desiredService := NewService(song)

	service, err := c.kubeclientset.CoreV1().Services(song.Namespace).Get(context.TODO(), serviceName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		_, err = c.kubeclientset.CoreV1().Services(song.Namespace).Create(context.TODO(), desiredService, metav1.CreateOptions{})
		if err != nil {
			return err
		}
		log.Printf("Service '%s' created", serviceName)
	} else if err != nil {
		return err
	} else {
		// Update (Note: ClusterIP is immutable, so reuse it)
		desiredService.ResourceVersion = service.ResourceVersion
		desiredService.Spec.ClusterIP = service.Spec.ClusterIP
		_, err = c.kubeclientset.CoreV1().Services(song.Namespace).Update(context.TODO(), desiredService, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
		log.Printf("Service '%s' updated", serviceName)
	}

	return nil
}

func (c *Controller) updateSongCodeStatus(song *v1alpha1.Song, deployment *appsv1.Deployment) error {
	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	songCopy := song.DeepCopy()
	songCopy.Status.AvailableReplicas = deployment.Status.AvailableReplicas

	// If the CustomResourceSubresources feature gate is not enabled,
	// we must use Update instead of UpdateStatus to update the Status block of the Song resource.
	// UpdateStatus will not allow changes to the Spec of the resource,
	// which is ideal for ensuring nothing other than resource status has been updated.

	_, err := c.myclientset.MusicV1().Songs(song.Namespace).Update(context.TODO(), songCopy, metav1.UpdateOptions{})

	return err
}

func NewDeployment(song *v1alpha1.Song) *appsv1.Deployment {
	labels := map[string]string{
		"app":    song.Name,
		"artist": song.Spec.Artist,
		"title":  song.Spec.Title,
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      song.Name,
			Namespace: song.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(song, v1alpha1.SchemeGroupVersion.WithKind("Song")),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &song.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "song-container",
							Image: song.Spec.Title, // assuming title is the image name
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8080,
									Protocol:      corev1.ProtocolTCP,
								},
							},
						},
					},
				},
			},
		},
	}
}
func NewService(song *v1alpha1.Song) *corev1.Service {
	labels := map[string]string{
		"app":   song.Name,
		"title": song.Spec.Title,
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      song.Name + "-service",
			Namespace: song.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(song, v1alpha1.SchemeGroupVersion.WithKind("Song")),
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Type:     corev1.ServiceTypeNodePort,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       80,
					TargetPort: intstr.FromInt32(8080),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
}
