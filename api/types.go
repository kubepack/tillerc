package kube

import (
	hapi_chart "k8s.io/helm/pkg/proto/hapi/chart"
	hapi_release "k8s.io/helm/pkg/proto/hapi/release"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
)

//-------------------------------------------------------------------------------------------
// Chart represents a chart that is installed in a Release.
// The ChartSource represents the location and type of a chart to install.
// This is modelled like Volume in Pods, which allows specifying a chart
// inline (like today) or pulling a chart object from a (potentially private) chart registry similar to pulling a Docker image.
// +optional
type ChartSource struct {
	// Inline charts are what is done today with Helm cli. Release request
	// contains the chart definition in the release spec, sent by Helm cli.
	Inline *hapi_chart.Chart `json:"inline,omitempty"`
}

//------------------------------------------------------------

/*
type ReleaseSpec struct {
    Chart Chart `protobuf:"bytes,1,opt,name=chart" json:"chart,omitempty"`
    // Values is a string containing (unparsed) YAML values.
    Values hapi_chart.Config `protobuf:"bytes,2,opt,name=values" json:"values,omitempty"`
    // DryRun, if true, will run through the release logic, but neither create
    // a release object nor deploy to Kubernetes. The release object returned
    // in the response will be fake.
    DryRun bool `protobuf:"varint,3,opt,name=dry_run,json=dryRun" json:"dry_run,omitempty"`
    // Name is the candidate release name. This must be unique to the
    // namespace, otherwise the server will return an error. If it is not
    // supplied, the server will autogenerate one.
    Name string `protobuf:"bytes,4,opt,name=name" json:"name,omitempty"`
    // DisableHooks causes the server to skip running any hooks for the install.
    DisableHooks bool `protobuf:"varint,5,opt,name=disable_hooks,json=disableHooks" json:"disable_hooks,omitempty"`
}

*/

// Release describes a deployment of a chart, together with the chart
// and the variables used to deploy that chart.
type Release struct {
	unversioned.TypeMeta `json:",inline,omitempty"`
	api.ObjectMeta       `json:"metadata,omitempty"`
	Spec                 ReleaseSpec   `json:"spec,omitempty"`
	Status               ReleaseStatus `json:"status,omitempty"`
}

type ReleaseSpec struct {
	// Chart is the protobuf representation of a chart.
	Chart ChartSource `protobuf:"bytes,1,opt,name=chart" json:"chart,omitempty"`

	//// Values is a string containing (unparsed) YAML values.
	//Values *Config `protobuf:"bytes,2,opt,name=values" json:"values,omitempty"`

	// Config is the set of extra Values added to the chart.
	// These values override the default values inside of the chart.
	Config *hapi_chart.Config `protobuf:"bytes,4,opt,name=config" json:"config,omitempty"`

	// DisableHooks causes the server to skip running any hooks for the install.
	DisableHooks bool `protobuf:"varint,5,opt,name=disable_hooks,json=disableHooks" json:"disable_hooks,omitempty"`

	// Manifest is the string representation of the rendered template.
	Manifest string `protobuf:"bytes,5,opt,name=manifest" json:"manifest,omitempty"`

	// Hooks are all of the hooks declared for this release.
	Hooks []*hapi_release.Hook `protobuf:"bytes,6,rep,name=hooks" json:"hooks,omitempty"`

	// Version is an int32 which represents the version of the release.
	Version int32 `protobuf:"varint,7,opt,name=version" json:"version,omitempty"`
}

type ReleaseStatus struct {
	// Info contains information about the release.
	//Info *Info `protobuf:"bytes,2,opt,name=info" json:"info,omitempty"`

	Status        *hapi_release.Status `protobuf:"bytes,1,opt,name=status" json:"status,omitempty"`
	FirstDeployed unversioned.Time     `protobuf:"bytes,2,opt,name=first_deployed,json=firstDeployed" json:"first_deployed,omitempty"`
	LastDeployed  unversioned.Time     `protobuf:"bytes,3,opt,name=last_deployed,json=lastDeployed" json:"last_deployed,omitempty"`
	// Deleted tracks when this object was deleted.
	Deleted unversioned.Time `protobuf:"bytes,4,opt,name=deleted" json:"deleted,omitempty"`
}

type ReleaseList struct {
	unversioned.TypeMeta `json:",inline"`
	unversioned.ListMeta `json:"metadata,omitempty"`
	Items                []Release `json:"items,omitempty"`
}
