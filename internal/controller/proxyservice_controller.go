package controller

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	networkingv1alpha1 "github.com/krishjj8/go-proxy-operator/api/v1alpha1"
)

type ProxyServiceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=networking.krish.platform,resources=proxyservices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.krish.platform,resources=proxyservices/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=networking.krish.platform,resources=proxyservices/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps;services,verbs=get;list;watch;create;update;patch;delete

func (r *ProxyServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	proxyService := &networkingv1alpha1.ProxyService{}
	err := r.Get(ctx, req.NamespacedName, proxyService)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("ProxyService resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get ProxyService")
		return ctrl.Result{}, err
	}

	logger.Info("Reconciling ProxyService", "Name", proxyService.Name, "Namespace", proxyService.Namespace)

	existingConfigMap := &corev1.ConfigMap{}
	configMapName := proxyService.Name + "-config"
	err = r.Get(ctx, client.ObjectKey{Namespace: proxyService.Namespace, Name: configMapName}, existingConfigMap)

	if err != nil && errors.IsNotFound(err) {
		cm, err := r.configMapForProxy(proxyService)
		if err != nil {
			logger.Error(err, "Failed to generate desired ConfigMap definition")
			return ctrl.Result{}, err
		}

		logger.Info("Creating a new ConfigMap", "ConfigMap.Namespace", cm.Namespace, "ConfigMap.Name", cm.Name)
		err = r.Create(ctx, cm)
		if err != nil {
			logger.Error(err, "Failed to create ConfigMap inside cluster")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		logger.Error(err, "Failed to inspect ConfigMap status")
		return ctrl.Result{}, err
	}

	existingDeployment := &appsv1.Deployment{}
	deploymentName := proxyService.Name + "-deployment"
	err = r.Get(ctx, client.ObjectKey{Namespace: proxyService.Namespace, Name: deploymentName}, existingDeployment)

	if err != nil && errors.IsNotFound(err) {
		dep, err := r.deploymentForProxy(proxyService)
		if err != nil {
			logger.Error(err, "Failed to generate desired Deployment definition")
			return ctrl.Result{}, err
		}

		logger.Info("Creating a new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		err = r.Create(ctx, dep)
		if err != nil {
			logger.Error(err, "Failed to create Deployment inside cluster")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		logger.Error(err, "Failed to inspect Deployment status")
		return ctrl.Result{}, err
	}

	desiredReplicas := proxyService.Spec.Replicas
	if *existingDeployment.Spec.Replicas != desiredReplicas {
		logger.Info("Scale mismatch detected. Syncing deployment topology",
			"CurrentReplicas", *existingDeployment.Spec.Replicas, "DesiredReplicas", desiredReplicas)

		existingDeployment.Spec.Replicas = &desiredReplicas
		err = r.Update(ctx, existingDeployment)
		if err != nil {
			logger.Error(err, "Failed to execute runtime scale mutation")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	existingService := &corev1.Service{}
	serviceName := proxyService.Name + "-service"
	err = r.Get(ctx, client.ObjectKey{Namespace: proxyService.Namespace, Name: serviceName}, existingService)

	if err != nil && errors.IsNotFound(err) {
		svc, err := r.serviceForProxy(proxyService)
		if err != nil {
			logger.Error(err, "Failed to generate desired Service definition")
			return ctrl.Result{}, err
		}

		logger.Info("Creating a new Cluster Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
		err = r.Create(ctx, svc)
		if err != nil {
			logger.Error(err, "Failed to create Service inside cluster")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		logger.Error(err, "Failed to inspect Service status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ProxyServiceReconciler) configMapForProxy(proxy *networkingv1alpha1.ProxyService) (*corev1.ConfigMap, error) {
	formattedUpstreams := fmt.Sprintf("[\"%s\"]", strings.Join(proxy.Spec.Upstreams, "\",\""))
	proxyConfigData := fmt.Sprintf(`listen_port: %d
rate_limit: %d
upstreams: %s
`, proxy.Spec.ListenPort, proxy.Spec.RateLimit, formattedUpstreams)

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      proxy.Name + "-config",
			Namespace: proxy.Namespace,
		},
		Data: map[string]string{
			"config.yaml": proxyConfigData,
		},
	}

	if err := ctrl.SetControllerReference(proxy, cm, r.Scheme); err != nil {
		return nil, err
	}
	return cm, nil
}

func (r *ProxyServiceReconciler) deploymentForProxy(proxy *networkingv1alpha1.ProxyService) (*appsv1.Deployment, error) {
	replicas := proxy.Spec.Replicas

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      proxy.Name + "-deployment",
			Namespace: proxy.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": proxy.Name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": proxy.Name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:            "reverse-proxy",
						Image:           "go-reverse-proxy:latest",
						ImagePullPolicy: corev1.PullNever,
						Ports: []corev1.ContainerPort{
							{ContainerPort: proxy.Spec.ListenPort, Name: "data-plane"},
							{ContainerPort: 9090, Name: "control-plane"},
						},
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "config-volume",
							MountPath: "/app/config",
						}},
					}},
					Volumes: []corev1.Volume{{
						Name: "config-volume",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: proxy.Name + "-config",
								},
							},
						},
					}},
				},
			},
		},
	}

	if err := ctrl.SetControllerReference(proxy, dep, r.Scheme); err != nil {
		return nil, err
	}
	return dep, nil
}

func (r *ProxyServiceReconciler) serviceForProxy(proxy *networkingv1alpha1.ProxyService) (*corev1.Service, error) {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      proxy.Name + "-service",
			Namespace: proxy.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"app": proxy.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "traffic-ingress",
					Port:       proxy.Spec.ListenPort,
					TargetPort: intstr.FromString("data-plane"),
				},
				{
					Name:       "admin-ingress",
					Port:       9090,
					TargetPort: intstr.FromString("control-plane"),
				},
			},
		},
	}

	if err := ctrl.SetControllerReference(proxy, svc, r.Scheme); err != nil {
		return nil, err
	}

	return svc, nil
}

func (r *ProxyServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1alpha1.ProxyService{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Named("proxyservice").
		Complete(r)
}
