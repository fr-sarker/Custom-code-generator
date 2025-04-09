package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"time"

	"github.com/frsarker/crd/pkg/Controllers"                             // Adjust the import path based on your project structure
	clientset "github.com/frsarker/crd/pkg/generated/clientset/versioned" // Your generated clientset
	v1 "github.com/frsarker/crd/pkg/generated/informers/externalversions"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "absolute path to kubeconfig")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to kubeconfig")
	}
	flag.Parse()

	// Load kubeconfig and initialize the clientset
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}

	client, err := clientset.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	// Create the shared informer factory
	informerFactory := v1.NewSharedInformerFactory(client, time.Second*30)

	// Create the custom controller
	songInformer := informerFactory.Music().V1().Songs()
	controller := Controller.NewController(songInformer)

	// Create a channel to signal the stop of the controller
	stopCh := make(chan struct{})
	defer close(stopCh)

	// Start the controller in a separate goroutine
	go controller.Run(2, stopCh) // 2 is the number of worker threads (you can adjust as needed)

	// Start the informer factory
	informerFactory.Start(stopCh)

	// Wait for the shutdown signal
	<-stopCh
	fmt.Println("Shutting down the controller")
}
