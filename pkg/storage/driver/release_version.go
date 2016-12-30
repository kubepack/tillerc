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
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"time"

	"github.com/appscode/tillerc/client/clientset"
	"github.com/golang/protobuf/proto"
	"k8s.io/kubernetes/pkg/api"
	kberrs "k8s.io/kubernetes/pkg/api/errors"

	hapi "github.com/appscode/tillerc/api"
	rspb "k8s.io/helm/pkg/proto/hapi/release"
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
	/*	// fetch the configmap holding the release named by key
		obj, err := versions.impl.Get(key)
		if err != nil {
			if kberrs.IsNotFound(err) {
				return nil, ErrReleaseNotFound
			}

			logerrf(err, "get: failed to get %q", key)
			return nil, err
		}
		// found the configmap, decode the base64 data string
		// TODO r, err := decodeRelease(obj.Data["release"])
		if err != nil {
			logerrf(err, "get: failed to decode data %q", key)
			return nil, err
		}
		// return the release object*/
	r := &hapi.Release{}
	return r, nil
}

// List fetches all releases and returns the list releases such
// that filter(release) == true. An error is returned if the
// configmap fails to retrieve the releases.
func (versions *ReleaseVersions) List(filter func(*rspb.Release) bool) ([]*rspb.Release, error) {
	lsel := kblabels.Set{"OWNER": "TILLER"}.AsSelector()
	opts := api.ListOptions{LabelSelector: lsel}

	list, err := versions.impl.List(opts)
	if err != nil {
		logerrf(err, "list: failed to list")
		return nil, err
	}

	var results []*rspb.Release

	// iterate over the configmaps object list
	// and decode each release
	for _, item := range list.Items {
		fmt.Println(item)
		rls, err := decodeRelease("") //(item.Data["release"])
		if err != nil {
			logerrf(err, "list: failed to decode release: %v", item)
			continue
		}
		if filter(rls) {
			results = append(results, rls)
		}
	}

	return results, nil
}

// Query fetches all releases that match the provided map of labels.
// An error is returned if the configmap fails to retrieve the releases.
func (versions *ReleaseVersions) Query(labels map[string]string) ([]*rspb.Release, error) {
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

	var results []*rspb.Release
	for _, item := range list.Items {
		fmt.Println(item)
		rls, err := decodeRelease("") //(item.Data["release"])
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

// Update updates the ConfigMap holding the release. If not found
// the ConfigMap is created to hold the release.
func (versions *ReleaseVersions) Update(key string, rls *hapi.Release) error {
	/*	// set labels for configmaps object meta data
		var lbs labels

		lbs.init()
		lbs.set("MODIFIED_AT", strconv.Itoa(int(time.Now().Unix())))

		// create a new configmap object to hold the release
		obj, err := newReleaseVersionObject(key, rls, lbs)
		if err != nil {
			logerrf(err, "update: failed to encode release %q", rls.Name)
			return err
		}
		// push the configmap object out into the kubiverse
		// TODO Check _, err = versions.impl.Update(obj)
		if err != nil {
			logerrf(err, "update: failed to update")
			return err
		}*/
	return nil
}

// Delete deletes the ConfigMap holding the release named by key.
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

	// encode the release
	/*	s, err := encodeRelease(rls)
		if err != nil {
			return nil, err
		}*/

	if lbs == nil {
		lbs.init()
	}

	// apply labels
	lbs.set("NAME", rls.Name)
	lbs.set("OWNER", owner)
	lbs.set("STATUS", rspb.Status_Code_name[int32(rls.Status.Status.Code)])
	lbs.set("VERSION", strconv.Itoa(int(rls.Spec.Version)))
	//create and return release version object

	releaseVersion := &hapi.ReleaseVersion{
		ObjectMeta: api.ObjectMeta{
			Name:   key,
			Labels: lbs.toMap(),
		},
	}
	releaseVersion.Spec.ReleaseSpec = rls.Spec
	return releaseVersion, nil

	// create and return configmap object
	/*	return &api.ConfigMap{
		ObjectMeta: api.ObjectMeta{
			Name:   key,
			Labels: lbs.toMap(),
		},
		Data: map[string]string{"release": s},
	}, nil*/
}

func encodeRelease(rls *rspb.Release) (string, error) {
	b, err := proto.Marshal(rls)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	w, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return "", err
	}
	if _, err = w.Write(b); err != nil {
		return "", err
	}
	w.Close()

	return b64.EncodeToString(buf.Bytes()), nil
}

// decodeRelease decodes the bytes in data into a release
// type. Data must contain a base64 encoded string of a
// valid protobuf encoding of a release, otherwise
// an error is returned.
func decodeRelease(data string) (*rspb.Release, error) {
	// base64 decode string
	b, err := b64.DecodeString(data)
	if err != nil {
		return nil, err
	}

	// For backwards compatibility with releases that were stored before
	// compression was introduced we skip decompression if the
	// gzip magic header is not found
	if bytes.Equal(b[0:3], magicGzip) {
		r, err := gzip.NewReader(bytes.NewReader(b))
		if err != nil {
			return nil, err
		}
		b2, err := ioutil.ReadAll(r)
		if err != nil {
			return nil, err
		}
		b = b2
	}

	var rls rspb.Release
	// unmarshal protobuf bytes
	if err := proto.Unmarshal(b, &rls); err != nil {
		return nil, err
	}
	return &rls, nil
}

// logerrf wraps an error with the a formatted string (used for debugging)
func logerrf(err error, format string, args ...interface{}) {
	log.Printf("configmaps: %s: %s\n", fmt.Sprintf(format, args...), err)
}
