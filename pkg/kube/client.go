package kube

import (
	"github.com/operator-framework/operator-sdk/pkg/sdk"
)

// New returns a new instance of Client.
func New() Client {
	return &client{}
}

// Client interface wraps the operator sdk package level functions like
// sdk.Create, sdk.Get and such.
//
// Why is this wrapper interface necessary?
// Because these package level functions don't provide a way to inject a fake
// client, it's hard to unit test reconciliation logic.
// As a workaround, the reconciliation logic, instead of using the package level
// function directly, it will use this interface. We can mock this interface
// and thus write proper unit test(s) for reconciliation logic.
//
// operator-sdk team is working on integrating controller-runtime
// which will make it possible to inject fake clients in the future. See
// https://github.com/operator-framework/operator-sdk/issues/382 for more.
type Client interface {
	// Create creates the provided object on the server and updates the arg
	// "object" with the result from the server(UID, resourceVersion, etc).
	// Returns an error if the object’s TypeMeta(Kind, APIVersion) or ObjectMeta(Name/GenerateName, Namespace) is missing or incorrect.
	// Can also return an api error from the server
	// e.g AlreadyExists https://github.com/kubernetes/apimachinery/blob/master/pkg/api/errors/errors.go#L423
	Create(object sdk.Object) error

	// Get gets the specified object and unmarshals the retrieved data into the "into" object.
	// "into" is a Object that must have
	// "Kind" and "APIVersion" specified in its "TypeMeta" field
	// and "Name" and "Namespace" specified in its "ObjectMeta" field.
	// "opts" configures the Get operation.
	//  When passed With WithGetOptions(o), the specified metav1.GetOptions is set.
	Get(into sdk.Object, opts ...sdk.GetOption) error

	// Update updates the provided object on the server and updates the arg
	// "object" with the result from the server(UID, resourceVersion, etc).
	// Returns an error if the object’s TypeMeta(Kind, APIVersion) or ObjectMeta(Name, Namespace) is missing or incorrect.
	// Can also return an api error from the server
	// e.g Conflict https://github.com/kubernetes/apimachinery/blob/master/pkg/api/errors/errors.go#L428
	Update(object sdk.Object) error
}

// client implements Client interface.
type client struct{}

func (*client) Create(object sdk.Object) error {
	return sdk.Create(object)
}

func (*client) Get(into sdk.Object, opts ...sdk.GetOption) error {
	return sdk.Get(into, opts...)
}

func (*client) Update(object sdk.Object) error {
	return sdk.Update(object)
}
