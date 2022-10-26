// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/riotkit-org/backup-maker-operator/pkg/apis/riotkit/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// RequestedBackupActionLister helps list RequestedBackupActions.
// All objects returned here must be treated as read-only.
type RequestedBackupActionLister interface {
	// List lists all RequestedBackupActions in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.RequestedBackupAction, err error)
	// RequestedBackupActions returns an object that can list and get RequestedBackupActions.
	RequestedBackupActions(namespace string) RequestedBackupActionNamespaceLister
	RequestedBackupActionListerExpansion
}

// requestedBackupActionLister implements the RequestedBackupActionLister interface.
type requestedBackupActionLister struct {
	indexer cache.Indexer
}

// NewRequestedBackupActionLister returns a new RequestedBackupActionLister.
func NewRequestedBackupActionLister(indexer cache.Indexer) RequestedBackupActionLister {
	return &requestedBackupActionLister{indexer: indexer}
}

// List lists all RequestedBackupActions in the indexer.
func (s *requestedBackupActionLister) List(selector labels.Selector) (ret []*v1alpha1.RequestedBackupAction, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.RequestedBackupAction))
	})
	return ret, err
}

// RequestedBackupActions returns an object that can list and get RequestedBackupActions.
func (s *requestedBackupActionLister) RequestedBackupActions(namespace string) RequestedBackupActionNamespaceLister {
	return requestedBackupActionNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// RequestedBackupActionNamespaceLister helps list and get RequestedBackupActions.
// All objects returned here must be treated as read-only.
type RequestedBackupActionNamespaceLister interface {
	// List lists all RequestedBackupActions in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.RequestedBackupAction, err error)
	// Get retrieves the RequestedBackupAction from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.RequestedBackupAction, error)
	RequestedBackupActionNamespaceListerExpansion
}

// requestedBackupActionNamespaceLister implements the RequestedBackupActionNamespaceLister
// interface.
type requestedBackupActionNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all RequestedBackupActions in the indexer for a given namespace.
func (s requestedBackupActionNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.RequestedBackupAction, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.RequestedBackupAction))
	})
	return ret, err
}

// Get retrieves the RequestedBackupAction from the indexer for a given namespace and name.
func (s requestedBackupActionNamespaceLister) Get(name string) (*v1alpha1.RequestedBackupAction, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("requestedbackupaction"), name)
	}
	return obj.(*v1alpha1.RequestedBackupAction), nil
}
