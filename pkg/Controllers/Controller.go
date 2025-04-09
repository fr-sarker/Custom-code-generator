package Controller

import (
	"fmt"
	v1 "github.com/frsarker/crd/pkg/generated/informers/externalversions/frsarker.dev/v1"
	songlisters "github.com/frsarker/crd/pkg/generated/listers/frsarker.dev/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"time"
)

type Controller struct {
	// To check if the internal cache synced fully or not

	HasSynced cache.InformerSynced

	// Read Only source of truth for cluster state

	Lister songlisters.SongLister

	// WorkQueue to keep track of what’s going in the controller

	Queue workqueue.TypedRateLimitingInterface[string]
}

func NewController(song v1.SongInformer) *Controller {
	c := &Controller{
		HasSynced: song.Informer().HasSynced,
		Lister:    song.Lister(),
		Queue:     workqueue.NewTypedRateLimitingQueue[string](workqueue.DefaultTypedControllerRateLimiter[string]()),
	}
	song.Informer().AddEventHandler(
		&cache.ResourceEventHandlerFuncs{
			AddFunc:    c.HandleAdd,
			UpdateFunc: c.HandleUpdate,
			DeleteFunc: c.HandleDelete,
		},
	)
	return c
}

func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.Queue.ShutDown()
	fmt.Println("Starting Song controller")

	//if !cache.WaitForCacheSync(stopCh, c.HasSynced) {
	//	fmt.Println("Timed out waiting for caches to sync")
	//	return
	//}
	fmt.Println("Caches synced")

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, 1*time.Second, stopCh)
	}
	<-stopCh
	fmt.Println("Stopping Song controller")
}

func (c *Controller) runWorker() {
	fmt.Println("Starting worker thread")
	for c.processNextItem() {

	}
}
func (c *Controller) processNextItem() bool {
	item, quit := c.Queue.Get()
	fmt.Println(item)
	if quit {
		return false
	}

	println(item)

	defer c.Queue.Done(item)
	key, err := cache.MetaNamespaceKeyFunc(item)
	if err == nil {
		fmt.Println("Processing key:", key)
	}
	song, err := c.Lister.Songs("default").Get(key)
	if err == nil {
		fmt.Println("Processing song:", song.Name)
	}
	return true
}

func (c *Controller) HandleAdd(obj interface{}) {
	fmt.Println("Adding song:", obj)
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err == nil {
		c.Queue.Add(key)
	}
}
func (c *Controller) HandleUpdate(old, new interface{}) {
	fmt.Println("Updating song:", new)
	key, err := cache.MetaNamespaceKeyFunc(new)
	if err == nil {
		c.Queue.Add(key)
	}
}
func (c *Controller) HandleDelete(obj interface{}) {
	fmt.Println("Deleting song:", obj)
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err == nil {
		c.Queue.Add(key)
	}
}
