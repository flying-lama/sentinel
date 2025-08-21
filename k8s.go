package main

import (
	"context"
	"fmt"
	"log"
	"os"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// getNodeName retrieves the current node name from environment variable
func (k *K8sClient) getNodeName() (string, error) {
	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		return "", fmt.Errorf("NODE_NAME environment variable not set")
	}
	return nodeName, nil
}

// GetNodePublicIP retrieves the public IP address from node labels
func (k *K8sClient) GetNodePublicIP() (string, error) {
	nodeName, err := k.getNodeName()
	if err != nil {
		return "", err
	}

	node, err := k.clientset.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("error getting node: %v", err)
	}

	publicIP, exists := node.Labels["public_ip"]
	if !exists {
		return "", fmt.Errorf("public_ip label not found on node %s", nodeName)
	}

	return publicIP, nil
}

// IsLeader checks if the current host is the leader based on a ConfigMap
func (k *K8sClient) IsLeader() bool {
	nodeName, err := k.getNodeName()
	if err != nil {
		log.Printf("Error getting node name: %v", err)
		return false
	}

	configMap, err := k.clientset.CoreV1().ConfigMaps("default").Get(context.TODO(), "leader-election", metav1.GetOptions{})
	if err != nil {
		log.Printf("Error getting ConfigMap: %v", err)
		return false
	}

	leader, exists := configMap.Data["leader"]
	if !exists {
		log.Println("Leader key not found in ConfigMap")
		return false
	}

	return leader == nodeName
}

// WatchEvents watches for changes in the leader election ConfigMap
func (k *K8sClient) WatchEvents(callback func()) {
	listWatcher := cache.NewListWatchFromClient(
		k.clientset.CoreV1().RESTClient(),
		"configmaps",
		"default",
		nil,
	)

	informer := cache.NewSharedInformer(
		listWatcher,
		&v1.ConfigMap{},
		0,
	)

	_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldConfigMap := oldObj.(*v1.ConfigMap)
			newConfigMap := newObj.(*v1.ConfigMap)
			if oldConfigMap.Name == "leader-election" && oldConfigMap.Data["leader"] != newConfigMap.Data["leader"] {
				callback()
			}
		},
	})
	if err != nil {
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
