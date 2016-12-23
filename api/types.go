package kube

import (
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
)

// Maintainer describes a Chart maintainer.
type Maintainer struct {
	// Name is a user name or organization name
	Name string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	// Email is an optional email address to contact the named maintainer
	Email string `protobuf:"bytes,2,opt,name=email" json:"email,omitempty"`
}

// 	Metadata for a Chart file. This models the structure of a Chart.yaml file.
//
// 	Spec: https://k8s.io/helm/blob/master/docs/design/chart_format.md#the-chart-file
type Metadata struct {
	// The name of the chart
	Name string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	// The URL to a relevant project page, git repo, or contact person
	Home string `protobuf:"bytes,2,opt,name=home" json:"home,omitempty"`
	// Source is the URL to the source code of this chart
	Sources []string `protobuf:"bytes,3,rep,name=sources" json:"sources,omitempty"`
	// A SemVer 2 conformant version string of the chart
	Version string `protobuf:"bytes,4,opt,name=version" json:"version,omitempty"`
	// A one-sentence description of the chart
	Description string `protobuf:"bytes,5,opt,name=description" json:"description,omitempty"`
	// A list of string keywords
	Keywords []string `protobuf:"bytes,6,rep,name=keywords" json:"keywords,omitempty"`
	// A list of name and URL/email address combinations for the maintainer(s)
	Maintainers []*Maintainer `protobuf:"bytes,7,rep,name=maintainers" json:"maintainers,omitempty"`
	// The name of the template engine to use. Defaults to 'gotpl'.
	Engine string `protobuf:"bytes,8,opt,name=engine" json:"engine,omitempty"`
	// The URL to an icon file.
	Icon string `protobuf:"bytes,9,opt,name=icon" json:"icon,omitempty"`
	// The API Version of this chart.
	ApiVersion string `protobuf:"bytes,10,opt,name=apiVersion" json:"apiVersion,omitempty"`
}

// Template represents a template as a name/value pair.
//
// By convention, name is a relative path within the scope of the chart's
// base directory.
type Template struct {
	// Name is the path-like name of the template.
	Name string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	// Data is the template as byte data.
	Data []byte `protobuf:"bytes,2,opt,name=data,proto3" json:"data,omitempty"`
}

// Value describes a configuration value as a string.
type Value struct {
	Value string `protobuf:"bytes,1,opt,name=value" json:"value,omitempty"`
}

// Config supplies values to the parametrizable templates of a chart.
type Config struct {
	Raw    string            `protobuf:"bytes,1,opt,name=raw" json:"raw,omitempty"`
	Values map[string]*Value `protobuf:"bytes,2,rep,name=values" json:"values,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
}

// 	Chart is a helm package that contains metadata, a default config, zero or more
// 	optionally parameterizable templates, and zero or more charts (dependencies).
type Chart struct {
	// Contents of the Chartfile.
	Metadata *Metadata `protobuf:"bytes,1,opt,name=metadata" json:"metadata,omitempty"`
	// Templates for this chart.
	Templates []*Template `protobuf:"bytes,2,rep,name=templates" json:"templates,omitempty"`
	// Charts that this chart depends on.
	Dependencies []*Chart `protobuf:"bytes,3,rep,name=dependencies" json:"dependencies,omitempty"`
	// Default config for this template.
	Values *Config `protobuf:"bytes,4,opt,name=values" json:"values,omitempty"`
	// Miscellaneous files in a chart archive,
	// e.g. README, LICENSE, etc.
	Files []byte `protobuf:"bytes,5,rep,name=files" json:"files,omitempty"`
}

//-------------------------------------------------------------------------------------------
// Chart represents a named chart that is installed in a Release.
type ChartVolume struct {
	Name string `json:"name"`
	// The ChartSource represents the location and type of a chart to install.
	// This is modelled like Volume in Pods, which allows specifying a chart
	// inline (like today) or pulling a chart object from a (potentially private) chart registry similar to pulling a Docker image.
	// +optional
	ChartSource `json:",inline,omitempty"`
}

type ChartSource struct {
	// Inline charts are what is done today with Helm cli. Release request
	// contains the chart definition in the release spec, sent by Helm cli.
	Inline *Chart `json:"inline,omitempty"`
}

//--------------------------------------------------------------------------------------------

type ReleaseStatusCode string

const (
	// Status_UNKNOWN indicates that a release is in an uncertain state.
	Status_UNKNOWN ReleaseStatusCode = "UNKNOWN"
	// Status_DEPLOYED indicates that the release has been pushed to Kubernetes.
	Status_DEPLOYED ReleaseStatusCode = "DEPLOYED"
	// Status_DELETED indicates that a release has been deleted from Kubermetes.
	Status_DELETED ReleaseStatusCode = "DELETED"
	// Status_SUPERSEDED indicates that this release object is outdated and a newer one exists.
	Status_SUPERSEDED ReleaseStatusCode = "SUPERSEDED"
	// Status_FAILED indicates that the release was not successfully deployed.
	Status_FAILED ReleaseStatusCode = "FAILED"
	// Status_DELETING indicates that a delete operation is underway.
	Status_DELETING ReleaseStatusCode = "DELETING"
)

// Status defines the status of a release.
type Status struct {
	Code    ReleaseStatusCode `protobuf:"varint,1,opt,name=code,enum=hapi.release.Status_Code" json:"code,omitempty"`
	Details []byte            `protobuf:"bytes,2,opt,name=details" json:"details,omitempty"`
	// Cluster resources as kubectl would print them.
	Resources string `protobuf:"bytes,3,opt,name=resources" json:"resources,omitempty"`
	// Contains the rendered templates/NOTES.txt if available
	Notes string `protobuf:"bytes,4,opt,name=notes" json:"notes,omitempty"`
}

// Info describes release information.
type Info struct {
	Status        *Status          `protobuf:"bytes,1,opt,name=status" json:"status,omitempty"`
	FirstDeployed unversioned.Time `protobuf:"bytes,2,opt,name=first_deployed,json=firstDeployed" json:"first_deployed,omitempty"`
	LastDeployed  unversioned.Time `protobuf:"bytes,3,opt,name=last_deployed,json=lastDeployed" json:"last_deployed,omitempty"`
	// Deleted tracks when this object was deleted.
	Deleted unversioned.Time `protobuf:"bytes,4,opt,name=deleted" json:"deleted,omitempty"`
}

type Hook_Event string

const (
	Hook_UNKNOWN       Hook_Event = "UNKNOWN"
	Hook_PRE_INSTALL   Hook_Event = "PRE_INSTALL"
	Hook_POST_INSTALL  Hook_Event = "POST_INSTALL"
	Hook_PRE_DELETE    Hook_Event = "PRE_DELETE"
	Hook_POST_DELETE   Hook_Event = "POST_DELETE"
	Hook_PRE_UPGRADE   Hook_Event = "PRE_UPGRADE"
	Hook_POST_UPGRADE  Hook_Event = "POST_UPGRADE"
	Hook_PRE_ROLLBACK  Hook_Event = "PRE_ROLLBACK"
	Hook_POST_ROLLBACK Hook_Event = "POST_ROLLBACK"
)

// Hook defines a hook object.
type Hook struct {
	Name string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	// Kind is the Kubernetes kind.
	Kind string `protobuf:"bytes,2,opt,name=kind" json:"kind,omitempty"`
	// Path is the chart-relative path to the template.
	Path string `protobuf:"bytes,3,opt,name=path" json:"path,omitempty"`
	// Manifest is the manifest contents.
	Manifest string `protobuf:"bytes,4,opt,name=manifest" json:"manifest,omitempty"`
	// Events are the events that this hook fires on.
	Events []Hook_Event `protobuf:"varint,5,rep,packed,name=events,enum=hapi.release.Hook_Event" json:"events,omitempty"`
	// LastRun indicates the date/time this was last run.
	LastRun unversioned.Time `protobuf:"bytes,6,opt,name=last_run,json=lastRun" json:"last_run,omitempty"`
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
	Chart ChartVolume `protobuf:"bytes,1,opt,name=chart" json:"chart,omitempty"`

	//// Values is a string containing (unparsed) YAML values.
	//Values *Config `protobuf:"bytes,2,opt,name=values" json:"values,omitempty"`

	// Config is the set of extra Values added to the chart.
	// These values override the default values inside of the chart.
	Config Config `protobuf:"bytes,4,opt,name=config" json:"config,omitempty"`

	// DisableHooks causes the server to skip running any hooks for the install.
	DisableHooks bool `protobuf:"varint,5,opt,name=disable_hooks,json=disableHooks" json:"disable_hooks,omitempty"`

	// Manifest is the string representation of the rendered template.
	Manifest string `protobuf:"bytes,5,opt,name=manifest" json:"manifest,omitempty"`

	// Hooks are all of the hooks declared for this release.
	Hooks []*Hook `protobuf:"bytes,6,rep,name=hooks" json:"hooks,omitempty"`

	// Version is an int32 which represents the version of the release.
	Version int32 `protobuf:"varint,7,opt,name=version" json:"version,omitempty"`
}

type ReleaseStatus struct {
	// Info contains information about the release.
	//Info *Info `protobuf:"bytes,2,opt,name=info" json:"info,omitempty"`

	Status        *Status          `protobuf:"bytes,1,opt,name=status" json:"status,omitempty"`
	FirstDeployed unversioned.Time `protobuf:"bytes,2,opt,name=first_deployed,json=firstDeployed" json:"first_deployed,omitempty"`
	LastDeployed  unversioned.Time `protobuf:"bytes,3,opt,name=last_deployed,json=lastDeployed" json:"last_deployed,omitempty"`
	// Deleted tracks when this object was deleted.
	Deleted unversioned.Time `protobuf:"bytes,4,opt,name=deleted" json:"deleted,omitempty"`
}

type ReleaseList struct {
	unversioned.TypeMeta `json:",inline"`
	unversioned.ListMeta `json:"metadata,omitempty"`
	Items                []Release `json:"items,omitempty"`
}
