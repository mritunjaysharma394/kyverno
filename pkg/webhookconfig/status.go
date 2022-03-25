package webhookconfig

import (
	"fmt"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	"github.com/kyverno/kyverno/pkg/config"
	"github.com/kyverno/kyverno/pkg/event"
	"github.com/pkg/errors"
<<<<<<< HEAD
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
=======
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coordinationv1 "k8s.io/client-go/kubernetes/typed/coordination/v1"
>>>>>>> e303dddf8... adds lease objects for storing last-request-time and set-status annotations in deployment (#3447)
)

var leaseName string = "kyverno"
var leaseNamespace string = config.KyvernoNamespace

const (
	annCounter         string = "kyverno.io/generationCounter"
	annWebhookStatus   string = "kyverno.io/webhookActive"
	annLastRequestTime string = "kyverno.io/last-request-time"
)

//statusControl controls the webhook status
type statusControl struct {
<<<<<<< HEAD
	register *Register
	eventGen event.Interface
	log      logr.Logger
=======
	eventGen    event.Interface
	log         logr.Logger
	leaseClient coordinationv1.LeaseInterface
>>>>>>> e303dddf8... adds lease objects for storing last-request-time and set-status annotations in deployment (#3447)
}

//success ...
func (vc statusControl) success() error {
	return vc.setStatus("true")
}

//failure ...
func (vc statusControl) failure() error {
	return vc.setStatus("false")
}

// NewStatusControl creates a new webhook status control
<<<<<<< HEAD
func newStatusControl(register *Register, eventGen event.Interface, log logr.Logger) *statusControl {
	return &statusControl{
		register: register,
		eventGen: eventGen,
		log:      log,
=======
func newStatusControl(leaseClient coordinationv1.LeaseInterface, eventGen event.Interface, log logr.Logger) *statusControl {
	return &statusControl{
		eventGen:    eventGen,
		log:         log,
		leaseClient: leaseClient,
>>>>>>> e303dddf8... adds lease objects for storing last-request-time and set-status annotations in deployment (#3447)
	}
}

func (vc statusControl) setStatus(status string) error {
	logger := vc.log.WithValues("name", leaseName, "namespace", leaseNamespace)
	var ann map[string]string
	var err error
<<<<<<< HEAD
	deploy, err := vc.register.client.GetResource("", "Deployment", deployNamespace, deployName)
=======

	lease, err := vc.leaseClient.Get(context.TODO(), "kyverno", metav1.GetOptions{})
>>>>>>> e303dddf8... adds lease objects for storing last-request-time and set-status annotations in deployment (#3447)
	if err != nil {
		vc.log.WithName("UpdateLastRequestTimestmap").Error(err, "Lease 'kyverno' not found. Starting clean-up...")
		return err
	}

	ann = lease.GetAnnotations()
	if ann == nil {
		ann = map[string]string{}
		ann[annWebhookStatus] = status
	}

	leaseStatus, ok := ann[annWebhookStatus]
	if ok {
		if leaseStatus == status {
			logger.V(4).Info(fmt.Sprintf("annotation %s already set to '%s'", annWebhookStatus, status))
			return nil
		}
	}

	// set the status
	logger.Info("updating deployment annotation", "key", annWebhookStatus, "val", status)
	ann[annWebhookStatus] = status
	lease.SetAnnotations(ann)

<<<<<<< HEAD
	// update counter
	_, err = vc.register.client.UpdateResource("", "Deployment", deployNamespace, deploy, false)
=======
	_, err = vc.leaseClient.Update(context.TODO(), lease, metav1.UpdateOptions{})
>>>>>>> e303dddf8... adds lease objects for storing last-request-time and set-status annotations in deployment (#3447)
	if err != nil {
		return errors.Wrapf(err, "key %s, val %s", annWebhookStatus, status)
	}

<<<<<<< HEAD
=======
	logger.Info("updated lease annotation", "key", annWebhookStatus, "val", status)

>>>>>>> e303dddf8... adds lease objects for storing last-request-time and set-status annotations in deployment (#3447)
	// create event on kyverno deployment
	createStatusUpdateEvent(status, vc.eventGen)
	return nil
}

func createStatusUpdateEvent(status string, eventGen event.Interface) {
	e := event.Info{}
	e.Kind = "Lease"
	e.Namespace = leaseNamespace
	e.Name = leaseName
	e.Reason = "Update"
	e.Message = fmt.Sprintf("admission control webhook active status changed to %s", status)
	eventGen.Add(e)
}

<<<<<<< HEAD
//IncrementAnnotation ...
func (vc statusControl) IncrementAnnotation() error {
	logger := vc.log
	var ann map[string]string
	var err error
	deploy, err := vc.register.client.GetResource("", "Deployment", deployNamespace, deployName)
	if err != nil {
		logger.Error(err, "failed to find Kyverno", "deployment", deployName, "namespace", deployNamespace)
		return err
	}

	ann = deploy.GetAnnotations()
	if ann == nil {
		ann = map[string]string{}
	}

	if ann[annCounter] == "" {
		ann[annCounter] = "0"
	}

	counter, err := strconv.Atoi(ann[annCounter])
	if err != nil {
		logger.Error(err, "Failed to parse string", "name", annCounter, "value", ann[annCounter])
		return err
	}

	// increment counter
	counter++
	ann[annCounter] = strconv.Itoa(counter)

	logger.V(3).Info("updating webhook test annotation", "key", annCounter, "value", counter, "deployment", deployName, "namespace", deployNamespace)
	deploy.SetAnnotations(ann)

	// update counter
	_, err = vc.register.client.UpdateResource("", "Deployment", deployNamespace, deploy, false)
	if err != nil {
		logger.Error(err, fmt.Sprintf("failed to update annotation %s for deployment %s in namespace %s", annCounter, deployName, deployNamespace))
		return err
	}

	return nil
}

func (vc statusControl) UpdateLastRequestTimestmap(new time.Time) error {
	_, deploy, err := vc.register.GetKubePolicyDeployment()
	if err != nil {
		return errors.Wrap(err, "unable to get Kyverno deployment")
	}

	annotation, ok, err := unstructured.NestedStringMap(deploy.UnstructuredContent(), "metadata", "annotations")
	if err != nil {
		return errors.Wrap(err, "unable to get annotation")
	}

	if !ok {
=======
func (vc statusControl) UpdateLastRequestTimestmap(new time.Time) error {

	lease, err := vc.leaseClient.Get(context.TODO(), leaseName, metav1.GetOptions{})
	if err != nil {
		vc.log.WithName("UpdateLastRequestTimestmap").Error(err, "Lease 'kyverno' not found. Starting clean-up...")
		return err
	}

	//add label to lease
	label := lease.GetLabels()
	if len(label) == 0 {
		label = make(map[string]string)
		label["app.kubernetes.io/name"] = "kyverno"
	}
	lease.SetLabels(label)

	annotation := lease.GetAnnotations()
	if annotation == nil {
>>>>>>> e303dddf8... adds lease objects for storing last-request-time and set-status annotations in deployment (#3447)
		annotation = make(map[string]string)
	}

	t, err := new.MarshalText()
	if err != nil {
		return errors.Wrap(err, "failed to marshal timestamp")
	}

	annotation[annLastRequestTime] = string(t)
<<<<<<< HEAD
	deploy.SetAnnotations(annotation)
	_, err = vc.register.client.UpdateResource("", "Deployment", deploy.GetNamespace(), deploy, false)
=======
	lease.SetAnnotations(annotation)

	//update annotations in lease
	_, err = vc.leaseClient.Update(context.TODO(), lease, metav1.UpdateOptions{})
>>>>>>> e303dddf8... adds lease objects for storing last-request-time and set-status annotations in deployment (#3447)
	if err != nil {
		return errors.Wrapf(err, "failed to update annotation %s for deployment %s in namespace %s", annLastRequestTime, lease.GetName(), lease.GetNamespace())
	}

	return nil
}
