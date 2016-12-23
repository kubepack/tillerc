package watcher

import (
	acs "github.com/appscode/tillerc/client/clientset"
	kapi "k8s.io/kubernetes/pkg/api"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/watch"
)

func DaemonSetListFunc(c clientset.Interface) func(kapi.ListOptions) (runtime.Object, error) {
	return func(opts kapi.ListOptions) (runtime.Object, error) {
		return c.Extensions().DaemonSets(kapi.NamespaceAll).List(opts)
	}
}

func DaemonSetWatchFunc(c clientset.Interface) func(options kapi.ListOptions) (watch.Interface, error) {
	return func(options kapi.ListOptions) (watch.Interface, error) {
		return c.Extensions().DaemonSets(kapi.NamespaceAll).Watch(options)
	}
}

func ReplicaSetListFunc(c clientset.Interface) func(kapi.ListOptions) (runtime.Object, error) {
	return func(opts kapi.ListOptions) (runtime.Object, error) {
		return c.Extensions().ReplicaSets(kapi.NamespaceAll).List(opts)
	}
}

func ReplicaSetWatchFunc(c clientset.Interface) func(options kapi.ListOptions) (watch.Interface, error) {
	return func(options kapi.ListOptions) (watch.Interface, error) {
		return c.Extensions().ReplicaSets(kapi.NamespaceAll).Watch(options)
	}
}

func StatefulSetListFunc(c clientset.Interface) func(kapi.ListOptions) (runtime.Object, error) {
	return func(opts kapi.ListOptions) (runtime.Object, error) {
		return c.Apps().StatefulSets(kapi.NamespaceAll).List(opts)
	}
}

func StatefulSetWatchFunc(c clientset.Interface) func(options kapi.ListOptions) (watch.Interface, error) {
	return func(options kapi.ListOptions) (watch.Interface, error) {
		return c.Apps().StatefulSets(kapi.NamespaceAll).Watch(options)
	}
}

func CertificateListFunc(c acs.AppsCodeExtensionInterface) func(kapi.ListOptions) (runtime.Object, error) {
	return func(opts kapi.ListOptions) (runtime.Object, error) {
		return c.Certificate(kapi.NamespaceAll).List(opts)
	}
}

func CertificateWatchFunc(c acs.AppsCodeExtensionInterface) func(options kapi.ListOptions) (watch.Interface, error) {
	return func(options kapi.ListOptions) (watch.Interface, error) {
		return c.Certificate(kapi.NamespaceAll).Watch(options)
	}
}
