package main

import (
	"flag"
	Controller "github.com/frsarker/crd/pkg/Controllers"
	clientset "github.com/frsarker/crd/pkg/generated/clientset/versioned"
	informers "github.com/frsarker/crd/pkg/generated/informers/externalversions"
	kubeinformers "k8s.io/client-go/informers"
	//appsinformer "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
	"time"
)

func main() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	myClient, err := clientset.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	kubeInformationFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	myInformationFactory := informers.NewSharedInformerFactory(myClient, time.Second*30)

	controller := Controller.NewController(
		kubeClient,
		myClient,
		kubeInformationFactory.Apps().V1().Deployments(),
		myInformationFactory.Music().V1().Songs())

	stopCh := make(chan struct{})
	defer close(stopCh)
	kubeInformationFactory.Start(stopCh)
	myInformationFactory.Start(stopCh)
	controller.Run(stopCh)
}
