package app

import (
	"github.com/appscode/log"
	"github.com/appscode/tillerc/pkg/events"
	"github.com/appscode/tillerc/pkg/stash"
	acw "github.com/appscode/tillerc/pkg/watcher"
)

type Watcher struct {
	acw.Watcher

	// name of the cloud provider
	ProviderName string

	// name of the cluster the daemon running.
	ClusterName string

	// Loadbalancer image name that will be used to create the LoadBalancer.
	LoadbalancerImage string
}

func (watch *Watcher) Run() {
	watch.Storage = &stash.Storage{}
	watch.Pod()
	watch.StatefulSet()
	watch.DaemonSet()
	watch.ReplicaSet()
	watch.Namespace()
	watch.Node()
	watch.Service()
	watch.RC()
	watch.Endpoint()
}

func (w *Watcher) Dispatch(e *events.Event) error {
	if e.Ignorable() {
		return nil
	}
	log.Debugln("Dispatching event with resource", e.ResourceType, "event", e.EventType)

	log.Infoln("DO SOMETHING REAL")
	return nil
}
