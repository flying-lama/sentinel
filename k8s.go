package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	coordinationv1 "k8s.io/api/coordination/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

// K8sClient handles communication with the Kubernetes API
type K8sClient struct {
	clientset *kubernetes.Clientset
}

// NewK8sClient creates a new Kubernetes client
func NewK8sClient() (*K8sClient, error) {
	kubeconfig := os.Getenv("KUBECONFIG")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &K8sClient{clientset: clientset}, nil
}

// GetNodeName retrieves the current node name from environment variable
func (k *K8sClient) GetNodeName() (string, error) {
	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		return "", fmt.Errorf("NODE_NAME environment variable not set")
	}
	return nodeName, nil
}

// GetNodePublicIP retrieves the public IP address from node
func (k *K8sClient) GetNodePublicIP() (string, error) {
	nodeName, err := k.GetNodeName()
	if err != nil {
		return "", err
	}

	node, err := k.clientset.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("error getting node: %v", err)
	}

	// Try to get from label
	publicIP, exists := node.Labels["public_ip"]
	if exists {
		return publicIP, nil
	}

	// Look for ExternalIP in node addresses
	for _, address := range node.Status.Addresses {
		if address.Type == v1.NodeExternalIP {
			return address.Address, nil
		}
	}

	return "", fmt.Errorf("no external IP found for node %s (neither in addresses nor in public_ip label)", nodeName)
}

// IsLeader checks if the current node is the leader by examining controller manager lease
func (k *K8sClient) IsLeader() bool {
	nodeName, err := k.GetNodeName()
	if err != nil {
		log.Printf("Error getting node name: %v", err)
		return false
	}

	lease, err := k.clientset.CoordinationV1().Leases("kube-system").Get(context.TODO(), "kube-controller-manager", metav1.GetOptions{})
	if err != nil {
		log.Printf("Error getting kube-controller-manager lease: %v", err)
		return false
	}

	if lease.Spec.HolderIdentity == nil {
		log.Println("No holder identity found in lease")
		return false
	}

	holderIdentity := *lease.Spec.HolderIdentity

	// Check if the holder identity starts with our node name followed by underscore
	// Format is typically: nodename_uuid
	expectedPrefix := nodeName + "_"
	return strings.HasPrefix(holderIdentity, expectedPrefix)
}

// WatchEvents watches for changes in leader election leases
func (k *K8sClient) WatchEvents(callback func()) {
	listWatcher := cache.NewListWatchFromClient(
		k.clientset.CoordinationV1().RESTClient(),
		"leases",
		"kube-system",
		fields.Everything(),
	)

	informer := cache.NewSharedInformer(
		listWatcher,
		&coordinationv1.Lease{},
		0,
	)

	_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldLease, ok := oldObj.(*coordinationv1.Lease)
			if !ok {
				log.Printf("Error: oldObj is not a Lease object")
				return
			}

			newLease, ok := newObj.(*coordinationv1.Lease)
			if !ok {
				log.Printf("Error: newObj is not a Lease object")
				return
			}

			// Watch for controller manager lease changes
			if oldLease.Name == "kube-controller-manager" {
				oldHolder := ""
				newHolder := ""

				if oldLease.Spec.HolderIdentity != nil {
					oldHolder = *oldLease.Spec.HolderIdentity
				}
				if newLease.Spec.HolderIdentity != nil {
					newHolder = *newLease.Spec.HolderIdentity
				}

				if oldHolder != newHolder {
					log.Printf("Leader change detected: %s -> %s", oldHolder, newHolder)
					callback()
				}
			}
		},
	})
	if err != nil {
		log.Printf("Error adding event handler: %v", err)
		return
	}

	stopCh := make(chan struct{})
	defer close(stopCh)

	go informer.Run(stopCh)

	// Wait forever
	select {}
}

func (k *K8sClient) GetConfigurationErrors() []string {
	return []string{}
}
