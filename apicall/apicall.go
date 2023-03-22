package apicall

import (
	"fmt"
	"reflect"

	"github.com/gprossliner/xhdl"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

func createClientObject[T client.Object]() T {
	var tp T
	ty_ptr := reflect.TypeOf(tp)
	ty_stu := ty_ptr.Elem()
	stu_inst := reflect.New(ty_stu).Interface()

	obj := (stu_inst).(client.Object)
	return obj.(T)
}

func ApiGetG[T client.Object](ctx xhdl.Context, cl client.Client, key client.ObjectKey) T {
	log := ctrllog.FromContext(ctx)
	obj := createClientObject[T]()

	ctx.Throw(cl.Get(ctx, key, obj))
	log.V(3).Info(fmt.Sprintf("API: Get kind:%s namespace:%s name:%s OK, ResourceVersion=%s", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetNamespace(), obj.GetName(), obj.GetResourceVersion()))

	return obj
}

// ApiTryGetG tried to get an object. It returns nil if the object has not been found (404)
func ApiTryGetG[T client.Object](ctx xhdl.Context, client client.Client, key client.ObjectKey) client.Object {
	log := ctrllog.FromContext(ctx)
	var obj T

	err := ctx.RunNested(func(xctx xhdl.Context) { obj = ApiGetG[T](xctx, client, key) })
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		} else {
			ctx.Throw(err)
		}
	}

	log.V(3).Info(fmt.Sprintf("API: Get kind:%s namespace:%s name:%s OK, ResourceVersion=%s", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetNamespace(), obj.GetName(), obj.GetResourceVersion()))
	return obj
}

func ApiGet(ctx xhdl.Context, client client.Client, key client.ObjectKey, obj client.Object) {
	log := ctrllog.FromContext(ctx)
	ctx.Throw(client.Get(ctx, key, obj))
	log.V(3).Info(fmt.Sprintf("API: Get kind:%s namespace:%s name:%s OK, ResourceVersion=%s", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetNamespace(), obj.GetName(), obj.GetResourceVersion()))
}

func ApiRefresh(ctx xhdl.Context, client client.Client, obj client.Object) {
	log := ctrllog.FromContext(ctx)
	ctx.Throw(client.Get(ctx, types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, obj))
	log.V(3).Info(fmt.Sprintf("API: Get kind:%s namespace:%s name:%s OK, ResourceVersion=%s", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetNamespace(), obj.GetName(), obj.GetResourceVersion()))
}

func ApiTryGet(ctx xhdl.Context, client client.Client, key client.ObjectKey, obj client.Object) (found bool) {
	log := ctrllog.FromContext(ctx)

	err := client.Get(ctx, key, obj)
	if err != nil {
		if errors.IsNotFound(err) {
			found = false
		} else {
			ctx.Throw(err)
		}
	} else {
		found = true
	}

	log.V(3).Info(fmt.Sprintf("API: Get kind:%s namespace:%s name:%s OK, ResourceVersion=%s", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetNamespace(), obj.GetName(), obj.GetResourceVersion()))
	return
}

func ApiUpdate(ctx xhdl.Context, client client.Client, obj client.Object) {
	log := ctrllog.FromContext(ctx)

	log.V(3).Info(fmt.Sprintf("API: Updating Object kind:%s namespace:%s name:%s OK, ResourceVersion=%s", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetNamespace(), obj.GetName(), obj.GetResourceVersion()))
	ctx.Throw(client.Update(ctx, obj))
	log.V(3).Info(fmt.Sprintf("API: Object updated kind:%s namespace:%s name:%s OK, ResourceVersion=%s", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetNamespace(), obj.GetName(), obj.GetResourceVersion()))
}

func ApiUpdateStatus(ctx xhdl.Context, client client.Client, obj client.Object) {
	log := ctrllog.FromContext(ctx)

	log.V(3).Info(fmt.Sprintf("API: Updating Status kind:%s namespace:%s name:%s OK, ResourceVersion=%s", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetNamespace(), obj.GetName(), obj.GetResourceVersion()))
	ctx.Throw(client.Status().Update(ctx, obj))
	log.V(3).Info(fmt.Sprintf("API: Status updated kind:%s namespace:%s name:%s OK, ResourceVersion=%s", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetNamespace(), obj.GetName(), obj.GetResourceVersion()))
}

func ApiList(ctx xhdl.Context, client client.Client, list client.ObjectList, opts ...client.ListOption) {
	log := ctrllog.FromContext(ctx)

	log.V(3).Info(fmt.Sprintf("API: List kind:%s", list.GetObjectKind().GroupVersionKind().Kind))
	ctx.Throw(client.List(ctx, list, opts...))

	if log.V(3).Enabled() {
		v := reflect.ValueOf(list).Elem()
		vitems := v.FieldByName("Items")

		if vitems.Kind() != reflect.Slice {
			panic("expected slice type, found " + vitems.Kind().String())
		}

		for i := 0; i < vitems.Len(); i++ {
			item := vitems.Index(i).FieldByName("ObjectMeta").Interface()
			obj := item.(metav1.ObjectMeta)
			log.V(3).Info(fmt.Sprintf("API: List Element namespace:%s name:%s ResourceVersion=%s", obj.GetNamespace(), obj.GetName(), obj.GetResourceVersion()))
		}

	}

}

func ApiCreate(ctx xhdl.Context, client client.Client, obj client.Object) {
	log := ctrllog.FromContext(ctx)
	log.V(3).Info(fmt.Sprintf("API: Creating object namespace:%s name:%s", obj.GetNamespace(), obj.GetName()))
	ctx.Throw(client.Create(ctx, obj))
	log.V(3).Info(fmt.Sprintf("API: Created object kind:%s namespace:%s name:%s OK, ResourceVersion=%s", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetNamespace(), obj.GetName(), obj.GetResourceVersion()))
}

func ApiDelete(ctx xhdl.Context, client client.Client, obj client.Object) {
	log := ctrllog.FromContext(ctx)
	log.V(3).Info(fmt.Sprintf("API: Deleting object namespace:%s name:%s", obj.GetNamespace(), obj.GetName()))
	ctx.Throw(client.Delete(ctx, obj))
	log.V(3).Info(fmt.Sprintf("API: Deleted object kind:%s namespace:%s name:%s OK, ResourceVersion=%s", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetNamespace(), obj.GetName(), obj.GetResourceVersion()))
}
