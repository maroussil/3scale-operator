package reconcilers

import (
	"context"
	"fmt"
	"testing"

	"github.com/3scale/3scale-operator/pkg/common"
	appsv1 "github.com/openshift/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCreateOnlyMutator(t *testing.T) {
	desired := &v1.ConfigMap{}
	existing := &v1.ConfigMap{}
	if changed, err := CreateOnlyMutator(desired, existing); changed || err != nil {
		t.Fatal("Create only mutator returned error or changed")
	}
}

func TestBaseReconcilerCreate(t *testing.T) {
	var (
		namespace = "operator-unittest"
	)

	s := scheme.Scheme
	err := appsv1.AddToScheme(s)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.TODO()

	// Objects to track in the fake client.
	objs := []runtime.Object{}

	// Create a fake client to mock API calls.
	cl := fake.NewFakeClient(objs...)
	clientAPIReader := fake.NewFakeClient(objs...)

	baseReconciler := NewBaseReconciler(cl, s, clientAPIReader, ctx, log)

	desiredConfigmap := &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "myConfigmap",
			Namespace: namespace,
		},
		Data: map[string]string{
			"somekey": "somevalue",
		},
	}

	err = baseReconciler.ReconcileResource(&v1.ConfigMap{}, desiredConfigmap, CreateOnlyMutator)
	if err != nil {
		t.Fatal(err)
	}

	reconciledConfigmap := &v1.ConfigMap{}
	objectKey, err := client.ObjectKeyFromObject(desiredConfigmap)
	if err != nil {
		t.Fatal(err)
	}
	err = cl.Get(context.TODO(), objectKey, reconciledConfigmap)
	// object must exist, that is all required to be tested
	if err != nil {
		t.Errorf("error fetching existing: %v", err)
	}
}

func TestBaseReconcilerUpdateNeeded(t *testing.T) {
	// Test that update is done when mutator tells
	var (
		name      = "myConfigmap"
		namespace = "operator-unittest"
	)

	s := runtime.NewScheme()
	err := appsv1.AddToScheme(s)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.TODO()

	existingConfigmap := &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string]string{
			"somekey": "somevalue",
		},
	}

	// Objects to track in the fake client.
	objs := []runtime.Object{existingConfigmap}

	// Create a fake client to mock API calls.
	cl := fake.NewFakeClient(objs...)
	clientAPIReader := fake.NewFakeClient(objs...)

	baseReconciler := NewBaseReconciler(cl, s, clientAPIReader, ctx, log)

	desiredConfigmap := &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string]string{
			"somekey": "somevalue",
		},
	}

	customMutator := func(existingObj, desiredObj common.KubernetesObject) (bool, error) {
		existing, ok := existingObj.(*v1.ConfigMap)
		if !ok {
			return false, fmt.Errorf("%T is not a *v1.ConfigMap", existingObj)
		}
		if existing.Data == nil {
			existing.Data = map[string]string{}
		}
		existing.Data["customKey"] = "customValue"
		return true, nil
	}

	err = baseReconciler.ReconcileResource(&v1.ConfigMap{}, desiredConfigmap, customMutator)
	if err != nil {
		t.Fatal(err)
	}

	reconciled := &v1.ConfigMap{}
	objectKey, err := client.ObjectKeyFromObject(desiredConfigmap)
	if err != nil {
		t.Fatal(err)
	}
	err = cl.Get(context.TODO(), objectKey, reconciled)
	if err != nil {
		t.Fatalf("error fetching reconciled: %v", err)
	}

	customValue, ok := reconciled.Data["customKey"]
	if !ok {
		t.Fatal("reconciled does not have reconciled data")
	}

	if customValue != "customValue" {
		t.Fatalf("reconciled have reconciled data. Expected: 'customValue', got: %s", customValue)
	}
}

func TestBaseReconcilerDelete(t *testing.T) {
	var (
		resourceName = "example-resource"
		namespace    = "operator-unittest"
	)

	s := runtime.NewScheme()
	err := appsv1.AddToScheme(s)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.TODO()

	existing := &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      resourceName,
			Namespace: namespace,
		},
		Data: map[string]string{
			"somekey": "somevalue",
		},
	}

	// Objects to track in the fake client.
	objs := []runtime.Object{existing}

	// Create a fake client to mock API calls.
	cl := fake.NewFakeClient(objs...)
	clientAPIReader := fake.NewFakeClient(objs...)

	desired := &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      resourceName,
			Namespace: namespace,
		},
		Data: map[string]string{
			"somekey": "somevalue",
		},
	}
	common.TagObjectToDelete(desired)

	baseReconciler := NewBaseReconciler(cl, s, clientAPIReader, ctx, log)
	err = baseReconciler.ReconcileResource(&v1.ConfigMap{}, desired, CreateOnlyMutator)
	if err != nil {
		t.Fatal(err)
	}

	objectKey, err := client.ObjectKeyFromObject(desired)
	if err != nil {
		t.Fatal(err)
	}
	reconciled := &v1.ConfigMap{}
	err = cl.Get(context.TODO(), objectKey, reconciled)
	// object should not exist, that is all required to be tested
	if !errors.IsNotFound(err) {
		t.Fatal(err)
	}
}