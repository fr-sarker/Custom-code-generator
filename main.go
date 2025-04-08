package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	// Importing the generated clientset and API group
	musicv1 "github.com/frsarker/crd/pkg/apis/frsarker.dev/v1"            // Your generated API group (adjust if needed)
	clientset "github.com/frsarker/crd/pkg/generated/clientset/versioned" // Your generated clientset

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	// Use the correct client for your custom Song CRD
	songClient := client.MusicV1().Songs("default") // Replace with the desired namespace

	// Create a new Song resource
	song := &musicv1.Song{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-custom-song", // Name of the Song resource
		},
		Spec: musicv1.SongSpec{
			Title:    "Shape of You",  // Example title
			Artist:   "Ed Sheeran",    // Example artist
			Rating:   5,               // Example rating
			Genres:   []string{"Pop"}, // Example genres
			Replicas: 4,
		},
	}

	// Create the Song resource
	fmt.Println("Creating Song resource...")
	created, err := songClient.Create(context.TODO(), song, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created Song resource %q.\n", created.GetName())

	// Update Song resource (for demonstration)
	prompt()
	fmt.Println("Updating Song resource...")

	// Modify some spec fields (for example)
	created.Spec.Rating = 4 // Change rating to 4

	updated, err := songClient.Update(context.TODO(), created, metav1.UpdateOptions{})
	if err != nil {
		panic(err)
	}
	fmt.Printf("Updated Song resource %q with rating %d.\n", updated.GetName(), updated.Spec.Rating)

	// List Songs in the namespace
	prompt()
	fmt.Println("Listing Song resources...")
	songList, err := songClient.List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err)
	}
	for _, s := range songList.Items {
		fmt.Printf(" - %s (Artist: %s, Rating: %d)\n", s.Name, s.Spec.Artist, s.Spec.Rating)
	}

	// Delete the Song resource
	prompt()
	fmt.Println("Deleting Song resource...")
	err = songClient.Delete(context.TODO(), "my-custom-song", metav1.DeleteOptions{})
	if err != nil {
		panic(err)
	}
	fmt.Println("Deleted Song resource.")
}

func prompt() {
	fmt.Print("Please type Enter to proceed: ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	fmt.Println()
}
