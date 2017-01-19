package test

import (
	"fmt"
	_env "github.com/appscode/go/env"
	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	rest "k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
	"log"
	"os"
	"testing"
)

func TestRollback(t *testing.T) {
	kubeClient := getKubernetesClient()
	event := &kapi.Event{
		ObjectMeta: kapi.ObjectMeta{
			Name:      "releaseRollback1",
			Namespace: "default",
		},
		InvolvedObject: kapi.ObjectReference{
			Kind:      "release",
			Namespace: "default",
			Name:      "testbusybox1",
		},
		Source: kapi.EventSource{
			Component: "release",
			Host:      "node_count@" + "test",
		},
		Reason:  "releaseRollback",
		Message: "test",
		Type:    kapi.EventTypeNormal,
		Count:   1,
	}
	_, err := kubeClient.Core().Events("default").Create(event)
	if err != nil {
		log.Fatalln(err)
	}

}

func getKubernetesClient() *internalclientset.Clientset {
	config, err := GetKubeConfig()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	client := internalclientset.NewForConfigOrDie(config)
	return client
}
func GetKubeConfig() (config *rest.Config, err error) {
	debugEnabled := _env.FromHost().DebugEnabled()
	if !debugEnabled {
		config, err = rest.InClusterConfig()
	} else {
		rules := clientcmd.NewDefaultClientConfigLoadingRules()
		rules.DefaultClientConfig = &clientcmd.DefaultClientConfig
		overrides := &clientcmd.ConfigOverrides{ClusterDefaults: clientcmd.ClusterDefaults}
		config, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig()
	}
	return
}
