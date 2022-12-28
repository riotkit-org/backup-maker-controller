// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	"time"

	v1alpha1 "github.com/riotkit-org/backup-maker-controller/pkg/apis/riotkit/v1alpha1"
	scheme "github.com/riotkit-org/backup-maker-controller/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// ScheduledBackupsGetter has a method to return a ScheduledBackupInterface.
// A group's client should implement this interface.
type ScheduledBackupsGetter interface {
	ScheduledBackups(namespace string) ScheduledBackupInterface
}

// ScheduledBackupInterface has methods to work with ScheduledBackup resources.
type ScheduledBackupInterface interface {
	Create(ctx context.Context, scheduledBackup *v1alpha1.ScheduledBackup, opts v1.CreateOptions) (*v1alpha1.ScheduledBackup, error)
	Update(ctx context.Context, scheduledBackup *v1alpha1.ScheduledBackup, opts v1.UpdateOptions) (*v1alpha1.ScheduledBackup, error)
	UpdateStatus(ctx context.Context, scheduledBackup *v1alpha1.ScheduledBackup, opts v1.UpdateOptions) (*v1alpha1.ScheduledBackup, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.ScheduledBackup, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.ScheduledBackupList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.ScheduledBackup, err error)
	ScheduledBackupExpansion
}

// scheduledBackups implements ScheduledBackupInterface
type scheduledBackups struct {
	client rest.Interface
	ns     string
}

// newScheduledBackups returns a ScheduledBackups
func newScheduledBackups(c *RiotkitV1alpha1Client, namespace string) *scheduledBackups {
	return &scheduledBackups{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the scheduledBackup, and returns the corresponding scheduledBackup object, and an error if there is any.
func (c *scheduledBackups) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.ScheduledBackup, err error) {
	result = &v1alpha1.ScheduledBackup{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("scheduledbackups").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of ScheduledBackups that match those selectors.
func (c *scheduledBackups) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.ScheduledBackupList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.ScheduledBackupList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("scheduledbackups").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested scheduledBackups.
func (c *scheduledBackups) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("scheduledbackups").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a scheduledBackup and creates it.  Returns the server's representation of the scheduledBackup, and an error, if there is any.
func (c *scheduledBackups) Create(ctx context.Context, scheduledBackup *v1alpha1.ScheduledBackup, opts v1.CreateOptions) (result *v1alpha1.ScheduledBackup, err error) {
	result = &v1alpha1.ScheduledBackup{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("scheduledbackups").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(scheduledBackup).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a scheduledBackup and updates it. Returns the server's representation of the scheduledBackup, and an error, if there is any.
func (c *scheduledBackups) Update(ctx context.Context, scheduledBackup *v1alpha1.ScheduledBackup, opts v1.UpdateOptions) (result *v1alpha1.ScheduledBackup, err error) {
	result = &v1alpha1.ScheduledBackup{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("scheduledbackups").
		Name(scheduledBackup.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(scheduledBackup).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *scheduledBackups) UpdateStatus(ctx context.Context, scheduledBackup *v1alpha1.ScheduledBackup, opts v1.UpdateOptions) (result *v1alpha1.ScheduledBackup, err error) {
	result = &v1alpha1.ScheduledBackup{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("scheduledbackups").
		Name(scheduledBackup.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(scheduledBackup).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the scheduledBackup and deletes it. Returns an error if one occurs.
func (c *scheduledBackups) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("scheduledbackups").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *scheduledBackups) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("scheduledbackups").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched scheduledBackup.
func (c *scheduledBackups) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.ScheduledBackup, err error) {
	result = &v1alpha1.ScheduledBackup{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("scheduledbackups").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
