package v1alpha1

type JobHealthStatus struct {
	ChildReference `json:",inline"`
	Message        string `json:"message"`
	Succeeded      bool   `json:"succeeded"`
	Running        bool   `json:"running"`
	Failed         bool   `json:"failed"`
}
