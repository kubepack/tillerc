package test

import (
	"fmt"
	"log"
	"os"
	"testing"

	_env "github.com/appscode/go/env"
	"github.com/appscode/tillerc/pkg/tiller"
	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	rest "k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"

	"encoding/json"
)

func TestRollback(t *testing.T) {
	kubeClient := getKubernetesClient()

	req := tiller.RollbackReq{
		DryRun:       false,
		Recreate:     true,
		DisableHooks: false,
		Timeout:      1,
	}
	reqByte, err := json.Marshal(req)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	event := &kapi.Event{
		ObjectMeta: kapi.ObjectMeta{
			Name:      "releaseRollback",
			Namespace: "default",
		},
		InvolvedObject: kapi.ObjectReference{
			Kind:      "release",
			Namespace: "default",
			Name:      "testbusybox1-v1",
		},
		Reason:  "releaseRollback",
		Message: string(reqByte),
		Type:    kapi.EventTypeNormal,
		Count:   1,
	}
	_, err = kubeClient.Core().Events("default").Create(event)
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
