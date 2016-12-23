package watcher

import (
	"reflect"
	"time"

	"github.com/appscode/log"
	hapi "github.com/appscode/tillerc/api"
	acs "github.com/appscode/tillerc/client/clientset"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/cache"
	rest "k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/util/wait"
	"k8s.io/kubernetes/pkg/watch"
)

type Watcher struct {
	Client *acs.ExtensionsClient
	// sync time to sync the list.
	SyncPeriod time.Duration
}

func New(c *rest.Config) *Watcher {
	return &Watcher{
		Client:     acs.NewExtensionsForConfigOrDie(c),
		SyncPeriod: time.Minute * 2,
	}
}

// Blocks caller. Intended to be called as a Go routine.
func (w *Watcher) RunAndHold() {
	lw := &cache.ListWatch{
		ListFunc: func(opts api.ListOptions) (runtime.Object, error) {
			return w.Client.Release(api.NamespaceAll).List(api.ListOptions{})
		},
		WatchFunc: func(options api.ListOptions) (watch.Interface, error) {
			return w.Client.Release(api.NamespaceAll).Watch(api.ListOptions{})
		},
	}
	_, controller := cache.NewInformer(lw,
		&hapi.Release{},
		w.SyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				log.Infoln("got one added event", obj.(*hapi.Release))
				w.doStuff(obj.(*hapi.Release))
			},
			DeleteFunc: func(obj interface{}) {
				log.Infoln("got one deleted event", obj.(*hapi.Release))
				w.doStuff(obj.(*hapi.Release))
			},
			UpdateFunc: func(old, new interface{}) {
				if !reflect.DeepEqual(old, new) {
					log.Infoln("got one updated event", new.(*hapi.Release))
					w.doStuff(new.(*hapi.Release))
				}
			},
		},
	)
	controller.Run(wait.NeverStop)
}

func (pl *Watcher) doStuff(release *hapi.Release) {

}
