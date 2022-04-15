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

	// ApiUrl: the baseUrl for the REST API provider.
	// +optional
	// +immutable
	ApiUrl *string `json:"apiUrl,omitempty"`

	// ApiCredentials required to authenticate ReST API git server.
	ApiCredentials RepoCredentials `json:"apiCredentials"`

	// Provider: the REST API provider.
	// Actually only 'github' is supported.
	// +optional
	// +immutable
	Provider *string `json:"provider,omitempty"`

	// Path: name of the folder in the git repository
	// to copy from (or to).
	// +optional
	// +immutable
	Path *string `json:"path,omitempty"`

	// Private: used only for target repository.
	// +optional
	// +immutable
	Private *bool `json:"private,omitempty"`
}

type RepoParameters struct {
	// FromRepo: .
	// +immutable
	FromRepo RepoOpts `json:"fromRepo"`

	// ToRepo: .
	// +immutable
	ToRepo RepoOpts `json:"toRepo"`
}

type RepoObservation struct {
	// CreationTimestamp: Creation timestamp in RFC3339 text
	// format.
	//CreationTimestamp string `json:"creationTimestamp,omitempty"`

	// CommitId: commit id of the last copy.
	CommitId string `json:"commit,omitempty"`
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

// A Repo is a managed resource that represents a Krateo Git Repository
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,git}
type Repo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RepoSpec   `json:"spec"`
	Status RepoStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RepoList contains a list of Repo.
type RepoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Repo `json:"items"`
}
