/*
Copyright 2016 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tiller

import (
	"reflect"
	"time"

	hapi "github.com/appscode/tillerc/api"
	hcs "github.com/appscode/tillerc/client/clientset"
	"github.com/golang/glog"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/cache"
	rest "k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/util/wait"
	"k8s.io/kubernetes/pkg/watch"

	"bytes"
	"errors"
	"fmt"
	"log"
	"path"
	"regexp"
	"strings"

	tc_api "github.com/appscode/tillerc/api"
	"github.com/appscode/tillerc/client/clientset"
	relutil "github.com/appscode/tillerc/pkg/releaseutil"
	"github.com/appscode/tillerc/pkg/storage"
	"github.com/appscode/tillerc/pkg/storage/driver"
	"github.com/appscode/tillerc/pkg/tiller/environment"
	ctx "golang.org/x/net/context"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/kube"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/proto/hapi/release"
	"k8s.io/helm/pkg/proto/hapi/services"
	"k8s.io/helm/pkg/timeconv"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/client/typed/discovery"
)

// releaseNameMaxLen is the maximum length of a release name.
//
// As of Kubernetes 1.4, the max limit on a name is 63 chars. We reserve 10 for
// charts to add data. Effectively, that gives us 53 chars.
// See https://github.com/kubernetes/helm/issues/1528
const releaseNameMaxLen = 53

// NOTESFILE_SUFFIX that we want to treat special. It goes through the templating engine
// but it's not a yaml file (resource) hence can't have hooks, etc. And the user actually
// wants to see this file after rendering in the status command. However, it must be a suffix
// since there can be filepath in front of it.
const notesFileSuffix = "NOTES.txt"

var (
	// errMissingChart indicates that a chart was not provided.
	errMissingChart = errors.New("no chart provided")
	// errMissingRelease indicates that a release (name) was not provided.
	errMissingRelease = errors.New("no release provided")
	// errInvalidRevision indicates that an invalid release revision number was provided.
	errInvalidRevision = errors.New("invalid release revision")
)

// ListDefaultLimit is the default limit for number of items returned in a list.
var ListDefaultLimit int64 = 512

// ValidName is a regular expression for names.
//
// According to the Kubernetes help text, the regular expression it uses is:
//
//	(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?
//
// We modified that. First, we added start and end delimiters. Second, we changed
// the final ? to + to require that the pattern match at least once. This modification
// prevents an empty string from matching.
var ValidName = regexp.MustCompile("^(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])+$")

// ReleaseServer implements the server-side gRPC endpoint for the HAPI services.
type ReleaseServer struct {
	env       *environment.Environment
	clientset internalclientset.Interface

	Client *hcs.ExtensionsClient
	// sync time to sync the list.
	SyncPeriod time.Duration
	Config     *rest.Config
}

func New(env *environment.Environment, c *rest.Config) *ReleaseServer {
	return &ReleaseServer{
		env:        env,
		clientset:  internalclientset.NewForConfigOrDie(c),
		Client:     hcs.NewExtensionsForConfigOrDie(c),
		SyncPeriod: time.Minute * 2,
		Config:     c,
	}
}

// Blocks caller. Intended to be called as a Go routine.
func (s *ReleaseServer) RunAndHold() {
	lw := &cache.ListWatch{
		ListFunc: func(opts api.ListOptions) (runtime.Object, error) {
			return s.Client.Release(api.NamespaceAll).List(api.ListOptions{})
		},
		WatchFunc: func(options api.ListOptions) (watch.Interface, error) {
			return s.Client.Release(api.NamespaceAll).Watch(api.ListOptions{})
		},
	}
	_, controller := cache.NewInformer(lw,
		&hapi.Release{},
		s.SyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				glog.Infoln("got one added event", obj.(*hapi.Release))
				// Setting Storage
				clientset, err := client.NewExtensionsForConfig(s.Config)
				if err != nil {
					log.Print(err)
				}
				s.env.Releases = storage.Init(driver.NewReleaseVersion(clientset.ReleaseVersion(obj.(*hapi.Release).Namespace)))
				err = s.InstallRelease(obj.(*hapi.Release))
				if err != nil {
					log.Print(err)
				}
			},
			DeleteFunc: func(obj interface{}) {
				glog.Infoln("got one deleted event", obj.(*hapi.Release))
				err := s.UninstallRelease(obj.(*hapi.Release))
				if err != nil {
					log.Print(err)
				}
			},
			UpdateFunc: func(old, new interface{}) {
				if !reflect.DeepEqual(old, new) {
					glog.Infoln("got one updated event", new.(*hapi.Release))
					s.UpdateRelease(new.(*hapi.Release))
				}
			},
		},
	)
	controller.Run(wait.NeverStop)
}

func (s *ReleaseServer) doStuff(release *hapi.Release) {

}

// UpdateRelease takes an existing release and new information, and upgrades the release.
func (s *ReleaseServer) UpdateRelease(rel *hapi.Release) error {
	currentRelease, err := s.prepareUpdate(rel)
	if err != nil {
		return err
	}
	fmt.Println(currentRelease.Status)
	err = s.performUpdate(currentRelease, rel)
	if err != nil {
		return err
	}
	if !rel.Spec.DryRun {
		if err := s.env.Releases.Create(rel); err != nil {
			return err
		}
	}
	return nil
}

func (s *ReleaseServer) performUpdate(originalRelease, updatedRelease *hapi.Release) error {
	updatedRelease.Status = tc_api.ReleaseStatus{}
	updatedRelease.Status.Status = new(release.Status)
	/*	originalRelease.Status = tc_api.ReleaseStatus{}
		originalRelease.Status.Status = new(release.Status)*/

	//TODO handle dry run sauman
	/*	if req.DryRun {
		log.Printf("Dry run for %s", updatedRelease.Name)
		return res, nil
	}*/
	// pre-upgrade hooks
	if !updatedRelease.Spec.DisableHooks {
		if err := s.execHook(updatedRelease.Spec.Hooks, updatedRelease.Name, updatedRelease.Namespace, preUpgrade, updatedRelease.Spec.Timeout); err != nil {
			return err
		}
	}
	if err := s.performKubeUpdate(originalRelease, updatedRelease, updatedRelease.Spec.Recreate); err != nil {
		log.Printf("warning: Release Upgrade %q failed: %s", updatedRelease.Name, err)
		originalRelease.Status.Status.Code = release.Status_SUPERSEDED
		updatedRelease.Status.Status.Code = release.Status_FAILED
		s.recordRelease(originalRelease, true)
		s.recordRelease(updatedRelease, false)
		return err
	}
	// post-upgrade hooks
	if !updatedRelease.Spec.DisableHooks {
		if err := s.execHook(updatedRelease.Spec.Hooks, updatedRelease.Name, updatedRelease.Namespace, postUpgrade, updatedRelease.Spec.Timeout); err != nil {
			return err
		}
	}
	originalRelease.Status.Status.Code = release.Status_SUPERSEDED
	s.recordRelease(originalRelease, true)
	updatedRelease.Status.Status.Code = release.Status_DEPLOYED
	return nil
}

// reuseValues copies values from the current release to a new release if the new release does not have any values.
//
// If the request already has values, or if there are no values in the current release, this does nothing.
func (s *ReleaseServer) reuseValues(req *hapi.Release, current *hapi.Release) {
	if (req.Spec.Config == nil || req.Spec.Config.Raw == "" || req.Spec.Config.Raw == "{}\n") && current.Spec.Config != nil && current.Spec.Config.Raw != "" && current.Spec.Config.Raw != "{}\n" {
		log.Printf("Copying values from %s (v%d) to new release.", current.Name, current.Spec.Version)
		req.Spec.Config = current.Spec.Config
	}
}

// prepareUpdate builds an updated release for an update operation.
func (s *ReleaseServer) prepareUpdate(rel *hapi.Release) (*hapi.Release, error) {
	if !ValidName.MatchString(rel.Name) {
		return nil, errMissingRelease
	}
	if rel.Spec.Chart.Inline == nil {
		return nil, errMissingChart
	}
	// finds the non-deleted release with the given name
	currentRelease, err := s.env.Releases.Last(rel.Name)
	if err != nil {
		return nil, err
	}

	// If new values were not supplied in the upgrade, re-use the existing values.
	s.reuseValues(rel, currentRelease)

	// Increment revision count. This is passed to templates, and also stored on
	// the release object.
	revision := currentRelease.Spec.Version + 1
	ts := timeconv.Now()
	options := chartutil.ReleaseOptions{
		Name:      rel.Name,
		Time:      ts,
		Namespace: currentRelease.Namespace,
		IsUpgrade: true,
		Revision:  int(revision),
	}

	valuesToRender, err := chartutil.ToRenderValues(rel.Spec.Chart.Inline, rel.Spec.Config, options)
	if err != nil {
		return nil, err
	}

	hooks, manifestDoc, notesTxt, err := s.renderResources(rel.Spec.Chart.Inline, valuesToRender)
	if err != nil {
		return nil, err
	}

	// Store an updated release.
	/*	updatedRelease := &release.Release{
		Name:      rel.Name,
		Namespace: currentRelease.Namespace,
		Chart:     rel.Spec.Chart,
		Config:    rel.Spec.Config,
		Info: &release.Info{
			FirstDeployed: currentRelease.Info.FirstDeployed,
			LastDeployed:  ts,
			Status:        &release.Status{Code: release.Status_UNKNOWN},
		},
		Version:  revision,
		Manifest: manifestDoc.String(),
		Hooks:    hooks,
	}*/
	//rel is the updated release
	rel.Spec.Hooks = hooks
	rel.Spec.Version = revision
	rel.Spec.Manifest = manifestDoc.String()

	if len(notesTxt) > 0 {
		rel.Status.Status.Notes = notesTxt
	}
	err = validateManifest(s.env.KubeClient, currentRelease.Namespace, manifestDoc.Bytes())
	return currentRelease, err
}

// RollbackRelease rolls back to a previous version of the given release.
func (s *ReleaseServer) RollbackRelease(c ctx.Context, req *services.RollbackReleaseRequest) (*services.RollbackReleaseResponse, error) {
	/*	currentRelease, targetRelease, err := s.prepareRollback(req)
		if err != nil {
			return nil, err
		}

		res, err := s.performRollback(currentRelease, targetRelease, req)
		if err != nil {
			return res, err
		}

		if !req.DryRun {
			if err := s.env.Releases.Create(targetRelease); err != nil {
				return res, err
			}
		}*/
	res := &services.RollbackReleaseResponse{}

	return res, nil
}

func (s *ReleaseServer) performRollback(currentRelease, targetRelease *release.Release, req *services.RollbackReleaseRequest) (*services.RollbackReleaseResponse, error) {
	res := &services.RollbackReleaseResponse{Release: targetRelease}
	/*
		if req.DryRun {
			log.Printf("Dry run for %s", targetRelease.Name)
			return res, nil
		}

		// pre-rollback hooks
		if !req.DisableHooks {
			if err := s.execHook(targetRelease.Hooks, targetRelease.Name, targetRelease.Namespace, preRollback, req.Timeout); err != nil {
				return res, err
			}
		}

		if err := s.performKubeUpdate(currentRelease, targetRelease, req.Recreate); err != nil {
			log.Printf("warning: Release Rollback %q failed: %s", targetRelease.Name, err)
			currentRelease.Info.Status.Code = release.Status_SUPERSEDED
			targetRelease.Info.Status.Code = release.Status_FAILED
			s.recordRelease(currentRelease, true)
			s.recordRelease(targetRelease, false)
			return res, err
		}

		// post-rollback hooks
		if !req.DisableHooks {
			if err := s.execHook(targetRelease.Hooks, targetRelease.Name, targetRelease.Namespace, postRollback, req.Timeout); err != nil {
				return res, err
			}
		}

		currentRelease.Info.Status.Code = release.Status_SUPERSEDED
		s.recordRelease(currentRelease, true)

		targetRelease.Info.Status.Code = release.Status_DEPLOYED*/

	return res, nil
}

func (s *ReleaseServer) performKubeUpdate(currentRelease, targetRelease *hapi.Release, recreate bool) error {
	kubeCli := s.env.KubeClient
	current := bytes.NewBufferString(currentRelease.Spec.Manifest)
	target := bytes.NewBufferString(targetRelease.Spec.Manifest)
	return kubeCli.Update(targetRelease.Namespace, current, target, recreate)
}

// prepareRollback finds the previous release and prepares a new release object with
//  the previous release's configuration
func (s *ReleaseServer) prepareRollback(req *services.RollbackReleaseRequest) (*release.Release, *release.Release, error) {
	/*switch {
	case !ValidName.MatchString(req.Name):
		return nil, nil, errMissingRelease
	case req.Version < 0:
		return nil, nil, errInvalidRevision
	}

	crls, err := s.env.Releases.Last(req.Name)
	if err != nil {
		return nil, nil, err
	}

	rbv := req.Version
	if req.Version == 0 {
		rbv = crls.Version - 1
	}

	log.Printf("rolling back %s (current: v%d, target: v%d)", req.Name, crls.Version, rbv)

	prls, err := s.env.Releases.Get(req.Name, rbv)
	if err != nil {
		return nil, nil, err
	}

	// Store a new release object with previous release's configuration
	target := &release.Release{
		Name:      req.Name,
		Namespace: crls.Namespace,
		Chart:     prls.Chart,
		Config:    prls.Config,
		Info: &release.Info{
			FirstDeployed: crls.Info.FirstDeployed,
			LastDeployed:  timeconv.Now(),
			Status: &release.Status{
				Code:  release.Status_UNKNOWN,
				Notes: prls.Info.Status.Notes,
			},
		},
		Version:  crls.Version + 1,
		Manifest: prls.Manifest,
		Hooks:    prls.Hooks,
	}
	*/
	crls := &release.Release{}
	target := &release.Release{}
	return crls, target, nil
}

func (s *ReleaseServer) engine(ch *chart.Chart) environment.Engine {
	renderer := s.env.EngineYard.Default()
	if ch.Metadata.Engine != "" {
		if r, ok := s.env.EngineYard.Get(ch.Metadata.Engine); ok {
			renderer = r
		} else {
			log.Printf("warning: %s requested non-existent template engine %s", ch.Metadata.Name, ch.Metadata.Engine)
		}
	}
	return renderer
}

// InstallRelease installs a release and stores the release record.
func (s *ReleaseServer) InstallRelease(rel *hapi.Release) error {
	err := s.prepareRelease(rel)
	if err != nil {
		log.Printf("Failed install prepare step: %s", err)
		// On dry run, append the manifest contents to a failed release. This is
		// a stop-gap until we can revisit an error backchannel post-2.0.
		//TODO check later
		/*		if req.DryRun && strings.HasPrefix(err.Error(), "YAML parse error") {
				err = fmt.Errorf("%s\n%s", err, rel.Manifest)
			}*/
		return err
	}
	err = s.performRelease(rel)
	if err != nil {
		log.Printf("Failed install perform step: %s", err)
	}
	return err
}

// prepareRelease builds a release for an install operation.
func (s *ReleaseServer) prepareRelease(rel *hapi.Release) error {
	if rel.Spec.Chart.Inline == nil || len(rel.Spec.Chart.Inline.Templates) == 0 {
		return errMissingChart
	}

	// Tamal
	//name, err := s.uniqName(req.Name, req.ReuseName)
	//if err != nil {
	//	return nil, err
	//}

	revision := 1
	ts := timeconv.Now()
	options := chartutil.ReleaseOptions{
		Name:      rel.Name,
		Time:      ts,
		Namespace: rel.Namespace,
		Revision:  revision,
		IsInstall: true,
	}
	valuesToRender, err := chartutil.ToRenderValues(rel.Spec.Chart.Inline, rel.Spec.Config, options) //TODO check rel.Spec.Config
	if err != nil {

		return err
	}
	hooks, manifestDoc, _, err := s.renderResources(rel.Spec.Chart.Inline, valuesToRender) // noteTxt
	if err != nil {
		return err
	}

	// Store a release.
	rel.Status.FirstDeployed = unversioned.Now()
	rel.Status.LastDeployed = unversioned.Now()
	rel.Status.Status = new(release.Status)
	rel.Status.Status.Code = release.Status_UNKNOWN
	rel.Spec.Hooks = hooks
	rel.Spec.Manifest = manifestDoc.String()
	rel.Spec.Version = int32(revision)

	/*	if len(notesTxt) > 0 {
		rel.Info.Status.Notes = notesTxt
	}*/

	err = validateManifest(s.env.KubeClient, rel.Namespace, manifestDoc.Bytes())
	return err
}

func getVersionSet(client discovery.ServerGroupsInterface) (versionSet, error) {
	defVersions := newVersionSet("v1")

	groups, err := client.ServerGroups()
	if err != nil {
		return defVersions, err
	}

	// FIXME: The Kubernetes test fixture for cli appears to always return nil
	// for calls to Discovery().ServerGroups(). So in this case, we return
	// the default API list. This is also a safe value to return in any other
	// odd-ball case.
	if groups == nil {
		return defVersions, nil
	}

	versions := unversioned.ExtractGroupVersions(groups)
	return newVersionSet(versions...), nil
}

func (s *ReleaseServer) renderResources(ch *chart.Chart, values chartutil.Values) ([]*release.Hook, *bytes.Buffer, string, error) {

	renderer := s.engine(ch)
	files, err := renderer.Render(ch, values)
	if err != nil {
		return nil, nil, "", err
	}

	// NOTES.txt gets rendered like all the other files, but because it's not a hook nor a resource,
	// pull it out of here into a separate file so that we can actually use the output of the rendered
	// text file. We have to spin through this map because the file contains path information, so we
	// look for terminating NOTES.txt. We also remove it from the files so that we don't have to skip
	// it in the sortHooks.
	notes := ""
	for k, v := range files {
		if strings.HasSuffix(k, notesFileSuffix) {
			// Only apply the notes if it belongs to the parent chart
			// Note: Do not use filePath.Join since it creates a path with \ which is not expected
			if k == path.Join(ch.Metadata.Name, "templates", notesFileSuffix) {
				notes = v
			}
			delete(files, k)
		}
	}

	// Sort hooks, manifests, and partials. Only hooks and manifests are returned,
	// as partials are not used after renderer.Render. Empty manifests are also
	// removed here.
	vs, err := getVersionSet(s.clientset.Discovery())
	if err != nil {
		return nil, nil, "", fmt.Errorf("Could not get apiVersions from Kubernetes: %s", err)
	}
	hooks, manifests, err := sortManifests(files, vs, InstallOrder)
	if err != nil {
		// By catching parse errors here, we can prevent bogus releases from going
		// to Kubernetes.
		//
		// We return the files as a big blob of data to help the user debug parser
		// errors.
		b := bytes.NewBuffer(nil)
		for name, content := range files {
			if len(strings.TrimSpace(content)) == 0 {
				continue
			}
			b.WriteString("\n---\n# Source: " + name + "\n")
			b.WriteString(content)
		}
		return nil, b, "", err
	}

	// Aggregate all valid manifests into one big doc.
	b := bytes.NewBuffer(nil)
	for _, m := range manifests {
		b.WriteString("\n---\n# Source: " + m.name + "\n")
		b.WriteString(m.content)
	}

	return hooks, b, notes, nil
}

func (s *ReleaseServer) recordRelease(r *hapi.Release, reuse bool) {
	if reuse {
		if err := s.env.Releases.Update(r); err != nil {
			log.Printf("warning: Failed to update release %q: %s", r.Name, err)
		}
	} else if err := s.env.Releases.Create(r); err != nil {
		log.Printf("warning: Failed to record release %q: %s", r.Name, err)
	}
}

// performRelease runs a release.
func (s *ReleaseServer) performRelease(rel *hapi.Release) error {
	//res := &services.InstallReleaseResponse{Release: r}
	rel.Status = tc_api.ReleaseStatus{}
	rel.Status.Status = new(release.Status)

	//TODO check DryRun later
	/*	if req.DryRun {
		log.Printf("Dry run for %s", r.Name)
		return res, nil
	}*/

	// pre-install hooks
	if !rel.Spec.DisableHooks {
		if err := s.execHook(rel.Spec.Hooks, rel.Name, rel.ObjectMeta.Name, preInstall, rel.Spec.Timeout); err != nil {
			return err
		}
	}

	switch /* h, err := s.env.Releases.History(rel.Name);*/ {

	// TODO handle replace part
	// if this is a replace operation, append to the release history
	/*case req.ReuseName && err == nil && len(h) >= 1:
	// get latest release revision
	relutil.Reverse(h, relutil.SortByRevision)

	// old release
	old := h[0]

	// update old release status
	old.Info.Status.Code = release.Status_SUPERSEDED
	s.recordRelease(old, true)

	// update new release with next revision number
	// so as to append to the old release's history
	r.Version = old.Version + 1

	if err := s.performKubeUpdate(old, r, false); err != nil {
		log.Printf("warning: Release replace %q failed: %s", r.Name, err)
		old.Info.Status.Code = release.Status_SUPERSEDED
		r.Info.Status.Code = release.Status_FAILED
		s.recordRelease(old, true)
		s.recordRelease(r, false)
		return res, err
	}*/

	default:
		// nothing to replace, create as normal
		// regular manifests
		b := bytes.NewBufferString(rel.Spec.Manifest)
		//os.Exit(1)
		if err := s.env.KubeClient.Create(rel.Namespace, b); err != nil {
			log.Printf("warning: Release %q failed: %s", rel.Name, err)
			rel.Status.Status.Code = release.Status_FAILED
			// record release in a release version ...
			s.recordRelease(rel, false)
			return fmt.Errorf("release %s failed: %s", rel.Name, err)
		}
	}

	// post-install hooks
	if !rel.Spec.DisableHooks {
		if err := s.execHook(rel.Spec.Hooks, rel.Name, rel.ObjectMeta.Namespace, postInstall, rel.Spec.Timeout); err != nil {
			log.Printf("warning: Release %q failed post-install: %s", rel.Name, err)
			rel.Status.Status.Code = release.Status_FAILED
			s.recordRelease(rel, false)
			return err
		}
	}

	// This is a tricky case. The release has been created, but the result
	// cannot be recorded. The truest thing to tell the user is that the
	// release was created. However, the user will not be able to do anything
	// further with this release.
	//
	// One possible strategy would be to do a timed retry to see if we can get
	// this stored in the future.
	rel.Status.Status.Code = release.Status_DEPLOYED
	s.recordRelease(rel, false)

	return nil
}

func (s *ReleaseServer) execHook(hs []*release.Hook, name, namespace, hook string, timeout int64) error {
	kubeCli := s.env.KubeClient
	code, ok := events[hook]
	if !ok {
		return fmt.Errorf("unknown hook %q", hook)
	}

	log.Printf("Executing %s hooks for %s", hook, name)
	for _, h := range hs {
		found := false
		for _, e := range h.Events {
			if e == code {
				found = true
			}
		}
		// If this doesn't implement the hook, skip it.
		if !found {
			continue
		}
		b := bytes.NewBufferString(h.Manifest)
		if err := kubeCli.Create(namespace, b); err != nil {
			log.Printf("warning: Release %q pre-install %s failed: %s", name, h.Path, err)
			return err
		}
		// No way to rewind a bytes.Buffer()?
		b.Reset()
		b.WriteString(h.Manifest)
		if err := kubeCli.WatchUntilReady(namespace, b, timeout); err != nil {
			log.Printf("warning: Release %q pre-install %s could not complete: %s", name, h.Path, err)
			return err
		}
		h.LastRun = timeconv.Now()
	}
	log.Printf("Hooks complete for %s %s", hook, name)
	return nil
}

func (s *ReleaseServer) purgeReleases(rels ...*hapi.Release) error {
	for _, rel := range rels {
		if _, err := s.env.Releases.Delete(rel.Name, rel.Spec.Version); err != nil {
			return err
		}
	}
	return nil
}

// UninstallRelease deletes all of the resources associated with this release, and marks the release DELETED.
func (s *ReleaseServer) UninstallRelease(rel *hapi.Release) error {
	if !ValidName.MatchString(rel.Name) {
		log.Printf("uninstall: Release not found: %s", rel.Name)
		return errMissingRelease
	}

	rels, err := s.env.Releases.History(rel.Name)
	if err != nil {
		log.Printf("uninstall: Release not loaded: %s", rel.Name)
		return err
	}
	if len(rels) < 1 {
		return errMissingRelease
	}

	relutil.SortByRevisionVersion(rels)
	rel = rels[len(rels)-1]
	// TODO: Are there any cases where we want to force a delete even if it's
	// already marked deleted?
	if rel.Spec.Purge {
		if err := s.purgeReleases(rels...); err != nil {
			log.Printf("uninstall: Failed to purge the release: %s", err)
			return err
		}
		return nil
	}
	log.Printf("uninstall: Deleting %s", rel.Name)
	rel.Status.Status = new(release.Status)
	rel.Status.Status.Code = release.Status_DELETING
	rel.Status.Deleted = unversioned.Now()
	if !rel.Spec.DisableHooks {
		if err := s.execHook(rel.Spec.Hooks, rel.Name, rel.Namespace, preDelete, rel.Spec.Timeout); err != nil {
			return err
		}
	}
	vs, err := getVersionSet(s.clientset.Discovery())
	if err != nil {
		return fmt.Errorf("Could not get apiVersions from Kubernetes: %s", err)
	}
	// From here on out, the release is currently considered to be in Status_DELETING
	// state.
	if err := s.env.Releases.Update(rel); err != nil {
		log.Printf("uninstall: Failed to store updated release: %s", err)
	}

	manifests := splitManifests(rel.Spec.Manifest)
	_, files, err := sortManifests(manifests, vs, UninstallOrder)
	if err != nil {
		// We could instead just delete everything in no particular order.
		// FIXME: One way to delete at this point would be to try a label-based
		// deletion. The problem with this is that we could get a false positive
		// and delete something that was not legitimately part of this release.
		return fmt.Errorf("corrupted release record. You must manually delete the resources: %s", err)
	}
	/*filesToKeep*/ _, filesToDelete := filterManifestsToKeep(files)
	/*	if len(filesToKeep) > 0 {
		res.Info = summarizeKeptManifests(filesToKeep)
	}*/
	// Collect the errors, and return them later.
	es := []string{}
	for _, file := range filesToDelete {
		b := bytes.NewBufferString(file.content)
		if err := s.env.KubeClient.Delete(rel.Namespace, b); err != nil {
			log.Printf("uninstall: Failed deletion of %q: %s", rel.Name, err)
			if err == kube.ErrNoObjectsVisited {
				// Rewrite the message from "no objects visited"
				err = errors.New("object not found, skipping delete")
			}
			es = append(es, err.Error())
		}
	}
	if !rel.Spec.DisableHooks {
		if err := s.execHook(rel.Spec.Hooks, rel.Name, rel.Namespace, postDelete, rel.Spec.Timeout); err != nil {
			es = append(es, err.Error())
		}
	}

	if rel.Spec.Purge {
		if err := s.purgeReleases(rels...); err != nil {
			log.Printf("uninstall: Failed to purge the release: %s", err)
		}
	}

	rel.Status.Status.Code = release.Status_DELETED
	if err := s.env.Releases.Update(rel); err != nil {
		log.Printf("uninstall: Failed to store updated release: %s", err)
	}
	var errs error
	if len(es) > 0 {
		errs = fmt.Errorf("deletion completed with %d error(s): %s", len(es), strings.Join(es, "; "))
	}
	return errs
}

func splitManifests(bigfile string) map[string]string {
	// This is not the best way of doing things, but it's how k8s itself does it.
	// Basically, we're quickly splitting a stream of YAML documents into an
	// array of YAML docs. In the current implementation, the file name is just
	// a place holder, and doesn't have any further meaning.
	sep := "\n---\n"
	tpl := "manifest-%d"
	res := map[string]string{}
	tmp := strings.Split(bigfile, sep)
	for i, d := range tmp {
		res[fmt.Sprintf(tpl, i)] = d
	}
	return res
}

func validateManifest(c environment.KubeClient, ns string, manifest []byte) error {
	r := bytes.NewReader(manifest)
	_, err := c.Build(ns, r)
	return err
}
