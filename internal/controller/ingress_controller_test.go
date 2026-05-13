package controller

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/controller/config"
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	"github.com/apache/apisix-ingress-controller/internal/utils"
)

type noopUpdater struct{}

func (noopUpdater) Update(status.Update) {}

func TestIngressStatusFallsBackToReadyPodNodeAddresses(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("add core scheme: %v", err)
	}
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add gatewayproxy scheme: %v", err)
	}

	selector := "app=apisix-ingress-controller"
	oldConfig := *config.ControllerConfig
	t.Cleanup(func() {
		config.SetControllerConfig(&oldConfig)
	})
	config.SetControllerConfig(&config.Config{
		ControllerName:                oldConfig.ControllerName,
		IngressStatusPodLabelSelector: selector,
		ProviderConfig:                oldConfig.ProviderConfig,
		Webhook:                       oldConfig.Webhook,
	})

	ingressClass := &networkingv1.IngressClass{
		ObjectMeta: metav1.ObjectMeta{Name: "apisix"},
		Spec: networkingv1.IngressClassSpec{
			Controller: config.GetControllerName(),
		},
	}
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "demo",
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: ptrTo("apisix"),
		},
	}
	node1 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "node-1"},
		Status: corev1.NodeStatus{
			Addresses: []corev1.NodeAddress{
				{Type: corev1.NodeInternalIP, Address: "10.0.0.1"},
				{Type: corev1.NodeExternalIP, Address: "1.1.1.1"},
			},
		},
	}
	node2 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "node-2"},
		Status: corev1.NodeStatus{
			Addresses: []corev1.NodeAddress{
				{Type: corev1.NodeExternalIP, Address: "1.1.1.1"},
				{Type: corev1.NodeInternalIP, Address: "10.0.0.2"},
			},
		},
	}
	readyPod := func(name, nodeName string) *corev1.Pod {
		return &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ingress-system",
				Name:      name,
				Labels: map[string]string{
					"app": "apisix-ingress-controller",
				},
			},
			Spec: corev1.PodSpec{
				NodeName: nodeName,
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
				Conditions: []corev1.PodCondition{
					{Type: corev1.PodReady, Status: corev1.ConditionTrue},
				},
			},
		}
	}
	notReadyPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ingress-system",
			Name:      "not-ready",
			Labels: map[string]string{
				"app": "apisix-ingress-controller",
			},
		},
		Spec: corev1.PodSpec{
			NodeName: "node-3",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{Type: corev1.PodReady, Status: corev1.ConditionFalse},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ingressClass, ingress, node1, node2, readyPod("ready-1", "node-1"), readyPod("ready-2", "node-2"), notReadyPod).
		Build()

	parsedSelector, err := labels.Parse(selector)
	if err != nil {
		t.Fatalf("parse selector: %v", err)
	}

	reconciler := &IngressReconciler{
		Client:            fakeClient,
		Scheme:            scheme,
		Log:               logr.Discard(),
		Updater:           noopUpdater{},
		statusPodSelector: parsedSelector,
	}

	tctx := provider.NewDefaultTranslateContext(context.Background())
	gatewayProxy := &v1alpha1.GatewayProxy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "proxy",
		},
	}
	tctx.GatewayProxies[utils.NamespacedNameKind(ingressClass)] = *gatewayProxy

	if err := reconciler.updateStatus(context.Background(), tctx, ingress, ingressClass); err != nil {
		t.Fatalf("update status: %v", err)
	}

	got := ingress.Status.LoadBalancer.Ingress
	want := []networkingv1.IngressLoadBalancerIngress{
		{IP: "1.1.1.1"},
	}
	if len(got) != len(want) {
		t.Fatalf("unexpected address count: got %d want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].IP != want[i].IP || got[i].Hostname != want[i].Hostname {
			t.Fatalf("unexpected address at %d: got %+v want %+v", i, got[i], want[i])
		}
	}
}

func ptrTo[T any](v T) *T {
	return &v
}
