package controller

import (
	"errors"
	"reflect"

	acrt "github.com/appscode/go/runtime"
	"github.com/appscode/log"
	tapi "github.com/appscode/searchlight/api"
	"github.com/appscode/searchlight/pkg/eventer"
	"github.com/appscode/searchlight/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
)

// Blocks caller. Intended to be called as a Go routine.
func (c *Controller) WatchPods() {
	defer acrt.HandleCrash()

	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return c.KubeClient.CoreV1().Pods(apiv1.NamespaceAll).List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return c.KubeClient.CoreV1().Pods(apiv1.NamespaceAll).Watch(metav1.ListOptions{})
		},
	}
	_, ctrl := cache.NewInformer(lw,
		&apiv1.Pod{},
		c.SyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if pod, ok := obj.(*apiv1.Pod); ok {
					log.Infof("Pod %s@%s added", pod.Name, pod.Namespace)
					if pod.Status.PodIP == "" {
						log.Warningf("Skipping pod %s@%s, since it has no IP", pod.Name, pod.Namespace)
						return
					}

					alerts, err := util.FindPodAlert(c.ExtClient, pod.ObjectMeta)
					if err != nil {
						log.Errorf("Error while searching PodAlert for Pod %s@%s.", pod.Name, pod.Namespace)
						return
					}
					if len(alerts) == 0 {
						log.Errorf("No PodAlert found for Pod %s@%s.", pod.Name, pod.Namespace)
						return
					}
					for i := range alerts {
						err = c.EnsurePod(pod, nil, alerts[i])
						if err != nil {
							log.Errorf("Failed to add icinga2 alert for Pod %s@%s.", pod.Name, pod.Namespace)
							// return
						}
					}
				}
			},
			UpdateFunc: func(old, new interface{}) {
				oldPod, ok := old.(*apiv1.Pod)
				if !ok {
					log.Errorln(errors.New("Invalid Pod object"))
					return
				}
				newPod, ok := new.(*apiv1.Pod)
				if !ok {
					log.Errorln(errors.New("Invalid Pod object"))
					return
				}

				log.Infof("Pod %s@%s updated", newPod.Name, newPod.Namespace)

				if !reflect.DeepEqual(oldPod.Labels, newPod.Labels) || oldPod.Status.PodIP != newPod.Status.PodIP {
					oldAlerts, err := util.FindPodAlert(c.ExtClient, oldPod.ObjectMeta)
					if err != nil {
						log.Errorf("Error while searching PodAlert for Pod %s@%s.", oldPod.Name, oldPod.Namespace)
						return
					}
					newAlerts, err := util.FindPodAlert(c.ExtClient, newPod.ObjectMeta)
					if err != nil {
						log.Errorf("Error while searching PodAlert for Pod %s@%s.", newPod.Name, newPod.Namespace)
						return
					}

					type change struct {
						old *tapi.PodAlert
						new *tapi.PodAlert
					}
					diff := make(map[string]*change)
					for i := range oldAlerts {
						diff[oldAlerts[i].Name] = &change{old: oldAlerts[i]}
					}
					for i := range newAlerts {
						if ch, ok := diff[newAlerts[i].Name]; ok {
							ch.new = newAlerts[i]
						} else {
							diff[newAlerts[i].Name] = &change{new: newAlerts[i]}
						}
					}

					for alert := range diff {
						ch := diff[alert]
						if oldPod.Status.PodIP == "" && newPod.Status.PodIP != "" {
							go c.EnsurePod(newPod, nil, ch.new)
						} else if ch.old == nil && ch.new != nil {
							go c.EnsurePod(newPod, nil, ch.new)
						} else if ch.old != nil && ch.new == nil {
							go c.EnsurePodDeleted(newPod, ch.old)
						} else if ch.old != nil && ch.new != nil && !reflect.DeepEqual(ch.old.Spec, ch.new.Spec) {
							go c.EnsurePod(newPod, ch.old, ch.new)
						}
					}
				}
			},
			DeleteFunc: func(obj interface{}) {
				if pod, ok := obj.(*apiv1.Pod); ok {
					log.Infof("Pod %s@%s deleted", pod.Name, pod.Namespace)

					alerts, err := util.FindPodAlert(c.ExtClient, pod.ObjectMeta)
					if err != nil {
						log.Errorf("Error while searching PodAlert for Pod %s@%s.", pod.Name, pod.Namespace)
						return
					}
					if len(alerts) == 0 {
						log.Errorf("No PodAlert found for Pod %s@%s.", pod.Name, pod.Namespace)
						return
					}
					for i := range alerts {
						err = c.EnsurePodDeleted(pod, alerts[i])
						if err != nil {
							log.Errorf("Failed to delete icinga2 alert for Pod %s@%s.", pod.Name, pod.Namespace)
							// return
						}
					}
				}
			},
		},
	)
	ctrl.Run(wait.NeverStop)
}

func (c *Controller) EnsurePod(pod *apiv1.Pod, old, new *tapi.PodAlert) (err error) {
	defer func() {
		if err == nil {
			c.recorder.Eventf(
				new,
				apiv1.EventTypeWarning,
				eventer.EventReasonSuccessfulSync,
				`Applied PodAlert: "%v"`,
				new.Name,
			)
			return
		} else {
			c.recorder.Eventf(
				new,
				apiv1.EventTypeWarning,
				eventer.EventReasonFailedToSync,
				`Fail to be apply PodAlert: "%v". Reason: %v`,
				new.Name,
				err,
			)
			return
		}
	}()

	if old == nil {
		err = c.podHost.Create(*new, *pod)
	} else {
		err = c.podHost.Update(*new, *pod)
	}
	return
}

func (c *Controller) EnsurePodDeleted(pod *apiv1.Pod, alert *tapi.PodAlert) (err error) {
	defer func() {
		if err == nil {
			c.recorder.Eventf(
				alert,
				apiv1.EventTypeWarning,
				eventer.EventReasonSuccessfulDelete,
				`Deleted PodAlert: "%v"`,
				alert.Name,
			)
			return
		} else {
			c.recorder.Eventf(
				alert,
				apiv1.EventTypeWarning,
				eventer.EventReasonFailedToDelete,
				`Fail to be delete PodAlert: "%v". Reason: %v`,
				alert.Name,
				err,
			)
			return
		}
	}()
	err = c.podHost.Delete(*alert, *pod)
	return
}