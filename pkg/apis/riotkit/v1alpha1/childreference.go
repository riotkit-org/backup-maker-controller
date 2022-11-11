package v1alpha1

import (
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"strings"
)

const LabelTrackingId = "riotkit.org/job-tracking-id"

type ChildReference struct {
	// API version of the referent.
	APIVersion string `json:"apiVersion" protobuf:"bytes,5,opt,name=apiVersion"`
	// Kind of the referent.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds
	Kind string `json:"kind" protobuf:"bytes,1,opt,name=kind"`
	// We cannot use name and UID, because those are unknown, when a resource is using "generateName"
	// Instead we will be using a label and a unique ID
	// TrackingId is the execution id. Every created object is getting its unique id
	TrackingId string `json:"trackingId" protobuf:"bytes,4,opt,name=trackingId"`
}

func (cr *ChildReference) GetGVK() schema.GroupVersionKind {
	split := strings.Split(cr.APIVersion, "/")
	var group string
	var version string
	if len(split) == 0 {
		group = ""
		version = split[0]
	} else {
		group = split[0]
		version = split[1]
	}
	return schema.GroupVersionKind{
		Group:   group,
		Version: version,
		Kind:    cr.Kind,
	}
}

type ChildrenReferences []ChildReference

// AddOwnedObject is adding object to the list of children references, so the parent object could have a complete list
// of all its children
func AddOwnedObject(ref *ChildrenReferences, doc *unstructured.Unstructured) {
	labels := doc.GetLabels()
	if _, ok := labels[LabelTrackingId]; !ok {
		logrus.Warnf("Cannot find label '%s' for object", LabelTrackingId)
		return
	}
	gvk := doc.GroupVersionKind()
	*ref = append(*ref, ChildReference{
		APIVersion: gvk.Group + "/" + gvk.Version,
		Kind:       gvk.Kind,
		TrackingId: labels[LabelTrackingId],
	})
}

// AppendJobIdTo is setting a label that marks a resource with a unique id
func AppendJobIdTo(doc *unstructured.Unstructured) {
	labels := doc.GetLabels()
	if len(labels) == 0 {
		labels = make(map[string]string)
	}
	id := uuid.New()
	labels[LabelTrackingId] = id.String()
	doc.SetLabels(labels)
}
