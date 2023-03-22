package operatorutils

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"github.com/gprossliner/xhdl"
	"github.com/world-direct/operatorutils/apicall"
	ctrl "sigs.k8s.io/controller-runtime"
	cclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type controllerBuilder[TObj cclient.Object] struct {
	cclient.Client
	Finalizers []finalizerRegistration[TObj]
	Steps      []StepWithResultFn[TObj]
	log        logr.Logger
}

type finalizerRegistration[TObj cclient.Object] struct {
	finalizer string
	fn        StepFn[TObj]
}

type ControllerBuilder[TObj cclient.Object] interface {
	WithLog(log logr.Logger) ControllerBuilder[TObj]
	Finalizer(finalizer string, finalizeFn StepFn[TObj]) ControllerBuilder[TObj]
	Step(stepFn StepFn[TObj]) ControllerBuilder[TObj]
	StepWithResult(stepFn StepWithResultFn[TObj]) ControllerBuilder[TObj]

	Build() func(ctx context.Context, req ctrl.Request) (ctrl.Result, error)
}

type ResultAction string

const (

	// ResultContinue directly executes the next step
	ResultContinue ResultAction = "Continue"

	// ResultRequeue stops processing steps, and requeues the workflow after RequeueAfter
	ResultRequeue ResultAction = "Requeue"

	// ResultExit stops all workflow processing. It will only be triggered by another reconcile
	ResultExit ResultAction = "Exit"
)

type StepResult struct {
	Action       ResultAction
	RequeueAfter time.Duration
}

type StepFn[TObj cclient.Object] func(ctx xhdl.Context, object TObj, client cclient.Client)
type StepWithResultFn[TObj cclient.Object] func(ctx xhdl.Context, object TObj, client cclient.Client) StepResult

func (cb *controllerBuilder[TObj]) WithLog(log logr.Logger) ControllerBuilder[TObj] {
	cb.log = log
	return cb
}

func (cb *controllerBuilder[TObj]) Finalizer(finalizer string, finalizeFn StepFn[TObj]) ControllerBuilder[TObj] {
	cb.Finalizers = append(cb.Finalizers, struct {
		finalizer string
		fn        StepFn[TObj]
	}{finalizer, finalizeFn})

	return cb
}

func (cb *controllerBuilder[TObj]) Step(stepFn StepFn[TObj]) ControllerBuilder[TObj] {
	return cb.StepWithResult(func(ctx xhdl.Context, object TObj, client cclient.Client) StepResult {
		stepFn(ctx, object, client)
		return StepResult{Action: ResultContinue}
	})
}

func (cb *controllerBuilder[TObj]) StepWithResult(stepFn StepWithResultFn[TObj]) ControllerBuilder[TObj] {
	cb.Steps = append(cb.Steps, stepFn)
	return cb
}

func (cb *controllerBuilder[TObj]) Build() func(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	return func(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

		res := ctrl.Result{}

		err := xhdl.RunContext(ctx, func(ctx xhdl.Context) {

			log := cb.log

			// get the object
			obj := apicall.ApiTryGetG[TObj](ctx, cb.Client, req.NamespacedName)
			if obj == nil {
				// object must have been deleted
				return
			}

			// process finalizers
			for _, fnreg := range cb.Finalizers {
				if fnreg.Run(ctx, obj, cb) {
					// need update, save object

					log.V(2).Info("Update object for finalizer and requeue")
					apicall.ApiUpdate(ctx, cb.Client, obj)

					// and requeue...
					res = ctrl.Result{Requeue: true}
					return
				}
			}

			// process steps
			for _, step := range cb.Steps {
				sres := step(ctx, obj.(TObj), cb.Client)
				if sres.RequeueAfter != time.Duration(0) {
					sres.Action = ResultRequeue
				}

				switch sres.Action {
				case ResultContinue:
					continue
				case ResultExit:
					res = ctrl.Result{}
					return
				case ResultRequeue:
					res = ctrl.Result{Requeue: true, RequeueAfter: sres.RequeueAfter}
					return
				}
			}

		})

		return res, err
	}

}

func New[TObj cclient.Object](client cclient.Client) ControllerBuilder[TObj] {
	return &controllerBuilder[TObj]{
		Client: client,
		log:    logr.Discard(),
	}
}

func (fnreg *finalizerRegistration[TObj]) Run(ctx xhdl.Context, obj cclient.Object, cb *controllerBuilder[TObj]) (needsupdate bool) {
	log := cb.log

	isMarkedToBeDeleted := obj.GetDeletionTimestamp() != nil
	if isMarkedToBeDeleted {

		if controllerutil.ContainsFinalizer(obj, fnreg.finalizer) {
			log.V(2).Info("Object marked for deletion, remove finalizer and execute finalize logic")
			controllerutil.RemoveFinalizer(obj, fnreg.finalizer)
			fnreg.fn(ctx, obj.(TObj), cb.Client)
			return true
		}
	}

	// Add finalizer for this CR
	if !controllerutil.ContainsFinalizer(obj, fnreg.finalizer) {

		log.V(2).Info("added finalizer to object")
		controllerutil.AddFinalizer(obj, fnreg.finalizer)
		return true
	}

	return false
}
