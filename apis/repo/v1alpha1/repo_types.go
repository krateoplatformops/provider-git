package v1alpha1

import (
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RepoCredentials required to authenticate.
type RepoCredentials struct {
	// Source of the ReST API Token.
	// +kubebuilder:validation:Enum=None;Secret;Environment
	Source xpv1.CredentialsSource `json:"source"`

	xpv1.CommonCredentialSelectors `json:",inline"`
}

type RepoOpts struct {
	// Url: the repository URL.
	// +immutable
	Url string `json:"url"`

	// Path: name of the folder in the git repository
	// to copy from (or to).
	// +optional
	// +immutable
	Path *string `json:"path,omitempty"`
}

type RepoParameters struct {
	// DeploymentId: correlationId with UI
	// +optional
	DeploymentId *string `json:"deploymentId"`

	// FromRepo: .
	// +immutable
	FromRepo RepoOpts `json:"fromRepo"`

	// ToRepo: .
	// +immutable
	ToRepo RepoOpts `json:"toRepo"`
}

type RepoObservation struct {
	// DeploymentId: correlation identifier with UI
	DeploymentId *string `json:"deploymentId,omitempty"`

	// CommitId: commit id of the last copy.
	CommitId *string `json:"commitId,omitempty"`
}

// A RepoSpec defines the desired state of a Repo.
type RepoSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       RepoParameters `json:"forProvider"`
}

// A RepoStatus represents the observed state of a Repo.
type RepoStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          RepoObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespace

// A Repo is a managed resource that represents a Krateo Git Repository
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="DEPLOYMENT_ID",type="string",JSONPath=".status.atProvider.deploymentId"
// +kubebuilder:printcolumn:name="COMMIT_ID",type="string",JSONPath=".status.atProvider.commitId"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,git}
type Repo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RepoSpec   `json:"spec"`
	Status RepoStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespace

// RepoList contains a list of Repo.
type RepoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Repo `json:"items"`
}
