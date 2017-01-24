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

package driver

import (
	"encoding/base64"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/appscode/tillerc/client/clientset"
	"k8s.io/kubernetes/pkg/api"
	kberrs "k8s.io/kubernetes/pkg/api/errors"

	"strings"

	hapi "github.com/appscode/tillerc/api"
	rspb "k8s.io/helm/pkg/proto/hapi/release"
	"k8s.io/kubernetes/pkg/api/unversioned"
	kblabels "k8s.io/kubernetes/pkg/labels"
)

var _ Driver = (*ReleaseVersions)(nil)

// ReleaseVersionDriverName is the string name of the driver.
const ReleaseVersion = "ReleaseVersion"

var b64 = base64.StdEncoding

var magicGzip = []byte{0x1f, 0x8b, 0x08}

// ReleaseVersion is a wrapper around an implementation of a kubernetes
// ReleaseVersions Interface.
type ReleaseVersions struct {
	impl client.ReleaseVersionInterface
}

// NewRevisionVersion initializes a new ConfigMaps wrapping an implmenetation of
// the kubernetes ThirdPartResource.
func NewReleaseVersion(impl client.ReleaseVersionInterface) *ReleaseVersions {
	return &ReleaseVersions{impl: impl}
}

// Name returns the name of the driver.
func (version *ReleaseVersions) Name() string {
	return ReleaseVersion
}

// Get fetches the release named by key. The corresponding release is returned
// or error if not found.
func (versions *ReleaseVersions) Get(key string) (*hapi.Release, error) {
	// fetch the configmap holding the release named by key
	obj, err := versions.impl.Get(key)
	if err != nil {
		if kberrs.IsNotFound(err) {
			return nil, ErrReleaseNotFound
		}

		logerrf(err, "get: failed to get %q", key)
		return nil, err
	}

	// return the release object
	// TODO sauman add more info
	s := strings.SplitN(obj.Name, "-", 2)
	meta := api.ObjectMeta{
		Namespace: obj.Namespace,
		Name:      s[0],
	}
	type_ := unversioned.TypeMeta{
		Kind:       "Release",
		APIVersion: "helm.sh/v1beta1",
	}
	r := &hapi.Release{
		ObjectMeta: meta,
		TypeMeta:   type_,
		Spec:       obj.Spec.ReleaseSpec,
	}
	return r, nil
}

// List fetches all releases and returns the list releases such
// that filter(release) == true. An error is returned if the
// configmap fails to retrieve the releases.
func (versions *ReleaseVersions) List(filter func(*rspb.Release) bool) ([]*rspb.Release, error) {
	var results []*rspb.Release
	return results, nil
}

// Query fetches all releases that match the provided map of labels.
// An error is returned if the configmap fails to retrieve the releases.
func (versions *ReleaseVersions) Query(labels map[string]string) ([]*hapi.Release, error) {
	ls := kblabels.Set{}
	for k, v := range labels {
		ls[k] = v
	}
	opts := api.ListOptions{LabelSelector: ls.AsSelector()}

	list, err := versions.impl.List(opts)
	if err != nil {
		logerrf(err, "query: failed to query with labels")
		return nil, err
	}
	if len(list.Items) == 0 {
		return nil, ErrReleaseNotFound
	}

	var results []*hapi.Release
	for _, item := range list.Items {
		rls, err := getReleaseFromReleaseVersion(&item) // Make release From release version equivalent to previous decode process
		if err != nil {
			logerrf(err, "query: failed to decode release: %s", err)
			continue
		}
		results = append(results, rls)
	}
	return results, nil
}

// Create creates a new ReleaseVersion holding the release. If the
// ReleaseVersion already exists, ErrReleaseExists is returned.
func (version *ReleaseVersions) Create(key string, rls *hapi.Release) error {
	// set labels for release version object meta data
	var lbs labels

	lbs.init()
	lbs.set("CREATED_AT", strconv.Itoa(int(time.Now().Unix())))

	// create a new releaseversion to hold the release
	obj, err := newReleaseVersionObject(key, rls, lbs)
	if err != nil {
		logerrf(err, "create: failed to encode release %q", rls.Name)
		return err
	}
	// push the release version object out into the kubiverse
	if _, err := version.impl.Create(obj); err != nil {
		if kberrs.IsAlreadyExists(err) {
			return ErrReleaseExists
		}
		logerrf(err, "create: failed to create")
		return err
	}
	return nil
}

// Update updates the release_version holding the release. If not found
// the release_version is created to hold the release.
func (versions *ReleaseVersions) Update(key string, rls *hapi.Release) error {
	// set labels for release_version object meta data
	var lbs labels

	lbs.init()
	lbs.set("MODIFIED_AT", strconv.Itoa(int(time.Now().Unix())))

	// create a new re object to hold the release
	obj, err := newReleaseVersionObject(key, rls, lbs)
	if err != nil {
		logerrf(err, "update: failed to encode release %q", rls.Name)
		return err
	}
	// push the release_version object out into the kubiverse
	_, err = versions.impl.Update(obj)
	if err != nil {
		logerrf(err, "update: failed to update")
		return err
	}
	return nil
}

// Delete deletes the release_version holding the release named by key.
func (versions *ReleaseVersions) Delete(key string) (rls *hapi.Release, err error) {
	// fetch the release to check existence
	if rls, err = versions.Get(key); err != nil {
		if kberrs.IsNotFound(err) {
			return nil, ErrReleaseNotFound
		}

		logerrf(err, "delete: failed to get release %q", key)
		return nil, err
	}
	// delete the release
	if err = versions.impl.Delete(key, &api.DeleteOptions{}); err != nil {
		return rls, err
	}
	return rls, nil
}

// newConfigMapsObject constructs a kubernetes ConfigMap object
// to store a release. Each configmap data entry is the base64
// encoded string of a release's binary protobuf encoding.
//
// The following labels are used within each configmap:
//
//    "MODIFIED_AT"    - timestamp indicating when this configmap was last modified. (set in Update)
//    "CREATED_AT"     - timestamp indicating when this configmap was created. (set in Create)
//    "VERSION"        - version of the release.
//    "STATUS"         - status of the release (see proto/hapi/release.status.pb.go for variants)
//    "OWNER"          - owner of the configmap, currently "TILLER".
//    "NAME"           - name of the release.
//
func newReleaseVersionObject(key string, rls *hapi.Release, lbs labels) (*hapi.ReleaseVersion, error) {
	const owner = "TILLER"
	if lbs == nil {
		lbs.init()
	}

	// apply labels
	lbs.set("NAME", rls.Name)
	lbs.set("OWNER", owner)
	lbs.set("STATUS", rspb.Status_Code_name[int32(rls.Status.Status.Code)])
	lbs.set("VERSION", strconv.Itoa(int(rls.Spec.Version)))
	//create and return release version object
	//  TODO Handle first release and last release

	releaseVersion := &hapi.ReleaseVersion{
		ObjectMeta: api.ObjectMeta{
			Name:   key,
			Labels: lbs.toMap(),
		},
	}
	releaseVersion.Spec.ReleaseSpec = rls.Spec
	releaseVersion.Status.Status = rls.Status.Status // status of release kept in release version
	releaseVersion.Status.Deployed = rls.Status.LastDeployed
	return releaseVersion, nil
}

// logerrf wraps an error with the a formatted string (used for debugging)
func logerrf(err error, format string, args ...interface{}) {
	log.Printf("configmaps: %s: %s\n", fmt.Sprintf(format, args...), err)
}

func getReleaseFromReleaseVersion(rv *hapi.ReleaseVersion) (*hapi.Release, error) {
	rs := &hapi.Release{}
	rs.TypeMeta.Kind = "Release"
	rs.TypeMeta.Kind = "helm.sh/v1beta1"
	rs.Spec = rv.Spec.ReleaseSpec
	rs.ObjectMeta = rv.ObjectMeta
	rs.Status.Status = rv.Status.Status
	rs.Status.LastDeployed = rv.Status.Deployed
	rs.Name = GetReleaseNameFromReleaseVersion(rv.Name)
	return rs, nil
}

func GetReleaseNameFromReleaseVersion(name string) string {
	releaseName := strings.Split(name, "-")
	return releaseName[0]
}
