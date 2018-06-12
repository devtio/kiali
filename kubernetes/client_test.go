package kubernetes

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func formatTime(t time.Time) string {
	return t.Format(time.RFC3339)
}

func TestNewClient(t *testing.T) {
	assert := assert.New(t)
	client, err := NewClient()

	virtualServices, err2 := GetVirtualServices(client, "default")
	for _, vs := range virtualServices {
		fmt.Println("Found Virtual Service: ")
		fmt.Println("Name: ", vs.GetObjectMeta().Name)
		fmt.Println("Created: ", formatTime(vs.GetObjectMeta().CreationTimestamp.Time))
		fmt.Println("Resource Version: ", vs.GetObjectMeta().ResourceVersion)
		fmt.Println("Hosts: ", vs.GetSpec()["hosts"])
		fmt.Println("Gateways: ", vs.GetSpec()["gateways"])
		fmt.Println("Http: ", vs.GetSpec()["http"])
		fmt.Println("Tcp: ", vs.GetSpec()["tcp"])
	}

	assert.Equal(err, nil)
	assert.Equal(err2, nil)
}

// test services

// GetVirtualServices return all VirtualServices for a given namespace.
// If serviceName param is provided it will filter all VirtualServices having a host defined on a particular service.
// It returns an error on any problem.
func GetVirtualServices(in *IstioClient, namespace string) ([]IstioObject, error) {
	result, err := in.istioNetworkingApi.Get().Namespace(namespace).Resource(virtualServices).Do().Get()
	if err != nil {
		return nil, err
	}
	virtualServiceList, ok := result.(*VirtualServiceList)
	if !ok {
		return nil, fmt.Errorf("%s doesn't return a VirtualService list", namespace)
	}

	virtualServices := make([]IstioObject, 0)
	for _, virtualService := range virtualServiceList.GetItems() {
		virtualServices = append(virtualServices, virtualService.DeepCopyIstioObject())
	}
	return virtualServices, nil
}

// old

func TestFilterDeploymentsForService(t *testing.T) {
	assert := assert.New(t)

	service := v1.Service{
		Spec: v1.ServiceSpec{
			Selector: map[string]string{"foo": "bar"}}}

	pods := v1.PodList{
		Items: []v1.Pod{
			v1.Pod{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:   "httpbin-v1",
					Labels: map[string]string{"foo": "bazz", "version": "v1"}}},
			v1.Pod{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:   "reviews-v1",
					Labels: map[string]string{"foo": "bar", "version": "v1"}}},
			v1.Pod{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:   "reviews-v2",
					Labels: map[string]string{"foo": "bar", "version": "v2"}}},
		}}

	deployments := v1beta1.DeploymentList{
		Items: []v1beta1.Deployment{
			v1beta1.Deployment{
				ObjectMeta: meta_v1.ObjectMeta{Name: "reviews-v1"},
				Spec: v1beta1.DeploymentSpec{
					Selector: &meta_v1.LabelSelector{
						MatchLabels: map[string]string{"foo": "bar", "version": "v1"}}}},
			v1beta1.Deployment{
				ObjectMeta: meta_v1.ObjectMeta{Name: "reviews-v2"},
				Spec: v1beta1.DeploymentSpec{
					Selector: &meta_v1.LabelSelector{
						MatchLabels: map[string]string{"foo": "bar", "version": "v2"}}}},
			v1beta1.Deployment{
				ObjectMeta: meta_v1.ObjectMeta{Name: "httpbin-v1"},
				Spec: v1beta1.DeploymentSpec{
					Selector: &meta_v1.LabelSelector{
						MatchLabels: map[string]string{"foo": "bazz", "version": "v1"}}}},
		}}

	matches := FilterDeploymentsForService(&service, &pods, &deployments)

	assert.Len(matches, 2)
	assert.Equal("reviews-v1", matches[0].ObjectMeta.Name)
	assert.Equal("reviews-v2", matches[1].ObjectMeta.Name)
}

func TestFilterDeploymentsForServiceWithSpecificLabels(t *testing.T) {
	assert := assert.New(t)

	service := v1.Service{
		Spec: v1.ServiceSpec{
			Selector: map[string]string{"jaeger-infra": "jaeger-pod"}}}

	pods := v1.PodList{
		Items: []v1.Pod{
			v1.Pod{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:   "jaeger-pod",
					Labels: map[string]string{"jaeger-infra": "jaeger-pod", "hash": "123456"}}},
		}}

	deployments := v1beta1.DeploymentList{
		Items: []v1beta1.Deployment{
			v1beta1.Deployment{
				Spec: v1beta1.DeploymentSpec{
					Selector: &meta_v1.LabelSelector{
						MatchLabels: map[string]string{"jaeger-infra": "jaeger-pod"}}}},
		}}

	matches := FilterDeploymentsForService(&service, &pods, &deployments)
	assert.Len(matches, 1)
}

func TestFilterDeploymentsForServiceWithoutPod(t *testing.T) {
	assert := assert.New(t)

	service := v1.Service{
		Spec: v1.ServiceSpec{
			Selector: map[string]string{"app": "foo"}}}

	pods := v1.PodList{}

	deployments := v1beta1.DeploymentList{
		Items: []v1beta1.Deployment{
			v1beta1.Deployment{
				ObjectMeta: meta_v1.ObjectMeta{
					Labels: map[string]string{"app": "foo", "hash": "123456"}}},
		}}

	matches := FilterDeploymentsForService(&service, &pods, &deployments)
	assert.Len(matches, 1)
}

func TestFilterPodsForEndpoints(t *testing.T) {
	assert := assert.New(t)

	endpoints := v1.Endpoints{
		Subsets: []v1.EndpointSubset{
			v1.EndpointSubset{
				Addresses: []v1.EndpointAddress{
					v1.EndpointAddress{
						TargetRef: &v1.ObjectReference{
							Name: "pod-1",
							Kind: "Pod",
						},
					},
					v1.EndpointAddress{
						TargetRef: &v1.ObjectReference{
							Name: "pod-2",
							Kind: "Pod",
						},
					},
					v1.EndpointAddress{
						TargetRef: &v1.ObjectReference{
							Name: "other",
							Kind: "Other",
						},
					},
					v1.EndpointAddress{},
				},
			},
			v1.EndpointSubset{
				Addresses: []v1.EndpointAddress{
					v1.EndpointAddress{
						TargetRef: &v1.ObjectReference{
							Name: "pod-3",
							Kind: "Pod",
						},
					},
				},
			},
		},
	}

	pods := v1.PodList{
		Items: []v1.Pod{
			v1.Pod{ObjectMeta: meta_v1.ObjectMeta{Name: "pod-1"}},
			v1.Pod{ObjectMeta: meta_v1.ObjectMeta{Name: "pod-2"}},
			v1.Pod{ObjectMeta: meta_v1.ObjectMeta{Name: "pod-3"}},
			v1.Pod{ObjectMeta: meta_v1.ObjectMeta{Name: "pod-999"}},
			v1.Pod{ObjectMeta: meta_v1.ObjectMeta{Name: "other"}},
		},
	}

	filtered := filterPodsForEndpoints(&endpoints, &pods)
	assert.Len(filtered, 3)
	assert.Equal("pod-1", filtered[0].Name)
	assert.Equal("pod-2", filtered[1].Name)
	assert.Equal("pod-3", filtered[2].Name)
}
