package main

import (
	"fmt"
	"time"

	logs "github.com/appscode/log/golog"
	_ "github.com/appscode/tillerc/api/install"
	"github.com/appscode/tillerc/pkg/watcher"
	"github.com/spf13/pflag"
	"k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
	"k8s.io/kubernetes/pkg/util/flag"
	"k8s.io/kubernetes/pkg/util/runtime"
	"k8s.io/kubernetes/pkg/version/verflag"
)

func main() {
	pflag.StringVar(&Master, "master", "", "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	pflag.StringVar(&KubeConfig, "kubeconfig", "", "Path to kubeconfig file with authorization information (the master location is set by the master flag).")

	flag.InitFlags()
	logs.InitLogs()
	defer logs.FlushLogs()

	verflag.PrintAndExitIfRequested()

	/*	// ref; https://github.com/kubernetes/kubernetes/blob/ba1666fb7b946febecfc836465d22903b687118e/cmd/kube-proxy/app/server.go#L168
		// Create a Kube Client
		// define api config source
		if KubeConfig == "" && Master == "" {
			log.Println("Neither --kubeconfig nor --master was specified.  Using default API client.  This might not work.")
		}
		// This creates a client, first loading any specified kubeconfig
		// file, and then overriding the Master flag, if non-empty.
		c, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: KubeConfig},
			&clientcmd.ConfigOverrides{ClusterInfo: clientcmdapi.Cluster{Server: Master}}).ClientConfig()
		if err != nil {
			panic(err)
		}*/

	config, err := restclient.InClusterConfig()
	if err != nil {
		rules := clientcmd.NewDefaultClientConfigLoadingRules()
		rules.DefaultClientConfig = &clientcmd.DefaultClientConfig
		overrides := &clientcmd.ConfigOverrides{ClusterDefaults: clientcmd.ClusterDefaults}
		config, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig()
	}
	if err != nil {
		fmt.Println("Could not get kubernetes config: %s", err)
		time.Sleep(30 * time.Minute)
		panic(err)
	}
	defer runtime.HandleCrash()
	w := watcher.New(config)
	fmt.Println("Starting tillerc...")
	w.RunAndHold()
}

var (
	Master     string
	KubeConfig string
)
