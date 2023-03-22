package operatorutils

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ProcessFinalizer performs the following tasks on an object:
//  1. It checks if the object is about to be deleted, and has the finalizer added. If runs the finalizeFn in this case
//  2. Adds the finalizer to the object, if it is not already contained
//
// if err is returned, the finalizeFn returned an error
// if modified is returned true, you should update the object
func ProcessFinalizer(object client.Object, finalizer string, finalizeFn func() error) (modified bool, err error) {

	isMarkedToBeDeleted := object.GetDeletionTimestamp() != nil
	if isMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(object, finalizer) {
			controllerutil.RemoveFinalizer(object, finalizer)
			err = finalizeFn()
			return true, err
		}
	}

	// Add finalizer for this CR
	if !controllerutil.ContainsFinalizer(object, finalizer) {
		controllerutil.AddFinalizer(object, finalizer)
		return true, nil
	}

	return false, nil
}

// SetStatusCondition sets the corresponding condition in conditions to newCondition.
// conditions must be non-nil.
//  1. if the condition of the specified type already exists (all fields of the existing condition are updated to
//     newCondition, LastTransitionTime is set to now if the new status differs from the old status)
//  2. if a condition of the specified type does not exist (LastTransitionTime is set to now() if unset, and newCondition is appended)
//
// Returns true if anything has changed (https://github.com/kubernetes/apimachinery/issues/148)
func SetStatusCondition(conditions *[]metav1.Condition, newCondition metav1.Condition) bool {
	if !meta.IsStatusConditionPresentAndEqual(*conditions, newCondition.Type, newCondition.Status) {
		meta.SetStatusCondition(conditions, newCondition)
		return true
	}

	return false
}

// GetAnnotation returns the value of the annotation 'name', or an empty string if not found
func GetAnnotation(metadata metav1.ObjectMeta, name string) string {
	for aname, value := range metadata.GetAnnotations() {
		if name == aname {
			return value
		}
	}

	return ""
}

// SetAnnotation adds or updates an annotation
func SetAnnotation(metadata *metav1.ObjectMeta, name, value string) {

	if metadata.Annotations == nil {
		metadata.Annotations = make(map[string]string)
	}

	metadata.Annotations[name] = value
}
