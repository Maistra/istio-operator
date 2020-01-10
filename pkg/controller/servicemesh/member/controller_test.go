package member

import (
	"testing"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	clienttesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	"github.com/maistra/istio-operator/pkg/apis"
	maistra "github.com/maistra/istio-operator/pkg/apis/maistra/v1"
	"github.com/maistra/istio-operator/pkg/controller/common"
	"github.com/maistra/istio-operator/pkg/controller/common/test"
	"github.com/maistra/istio-operator/pkg/controller/common/test/assert"
)

const (
	memberName            = "default"
	memberRollName        = "default"
	appNamespace          = "app-namespace"
	controlPlaneName      = "my-mesh"
	controlPlaneNamespace = "cp-namespace"
)

var (
	request = reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      memberName,
			Namespace: appNamespace,
		},
	}

	oneMinuteAgo = meta.NewTime(time.Now().Add(-time.Minute))
)

func init() {
	logf.SetLogger(logf.ZapLogger(true))
}

func TestReconcileAddsFinalizer(t *testing.T) {
	member := newMember()
	member.Finalizers = []string{}

	cl, tracker, r := createClientAndReconciler(t, member)

	assertReconcileSucceeds(r, t)

	updatedMember := test.GetUpdatedObject(cl, member.ObjectMeta, &maistra.ServiceMeshMember{}).(*maistra.ServiceMeshMember)

	assert.DeepEquals(updatedMember.GetFinalizers(), []string{common.FinalizerName}, "Invalid finalizers in SMM", t)
	test.AssertNumberOfWriteActions(t, tracker.Actions(), 1)
}

func TestReconcileAddsNamespaceToMemberRoll(t *testing.T) {
	member := newMember()
	memberRoll := newMemberRoll()

	cl, _, r := createClientAndReconciler(t, member, memberRoll)

	assertReconcileSucceeds(r, t)

	updatedMemberRoll := test.GetUpdatedObject(cl, memberRoll.ObjectMeta, &maistra.ServiceMeshMemberRoll{}).(*maistra.ServiceMeshMemberRoll)
	assert.DeepEquals(updatedMemberRoll.Spec.Members, []string{appNamespace}, "App namespace not found in SMMR members", t)
}

func TestReconcileCreatesMemberRollIfNeeded(t *testing.T) {
	member := newMember()
	cl, _, r := createClientAndReconciler(t, member)

	assertReconcileSucceeds(r, t)

	memberRollKey := types.NamespacedName{Namespace: controlPlaneNamespace, Name: common.MemberRollName}
	test.AssertObjectExists(cl, memberRollKey, &maistra.ServiceMeshMemberRoll{}, "Expected reconcile to create the SMMR, but it didn't", t)

	memberRoll := test.GetObject(cl, memberRollKey, &maistra.ServiceMeshMemberRoll{}).(*maistra.ServiceMeshMemberRoll)

	createdByAnnotation, annotationFound := memberRoll.Annotations[common.CreatedByKey]
	if !annotationFound {
		t.Fatalf("Expected reconcile to create the SMMR with the annotation %s, but the annotation was missing.", common.CreatedByKey)
	}
	assert.DeepEquals(createdByAnnotation, controllerName, "Wrong annotation value", t)
	assert.DeepEquals(memberRoll.Spec.Members, []string{appNamespace}, "App namespace not found in SMMR members", t)
}

func TestReconcileDoesNothingIfReferencedControlPlaneNamespaceDoesNotExist(t *testing.T) {
	member := newMember()
	member.Spec.ControlPlaneRef.Namespace = "nonexistent-ns"

	_, tracker, r := createClientAndReconciler(t, member)

	tracker.AddReactor(func(action clienttesting.Action) (handled bool, err error) {
		if action.GetNamespace() == "nonexistent-ns" {
			return true, apierrors.NewNotFound(schema.GroupResource{
				Group:    "",
				Resource: "Namespace",
			}, action.GetNamespace())
		}
		return false, nil
	})

	assertReconcileSucceeds(r, t)
}

func TestReconcileRemovesNamespaceFromMemberRollAndRemovesFinalizerFromMember(t *testing.T) {
	member := newMember()
	member.DeletionTimestamp = &oneMinuteAgo
	memberRoll := newMemberRoll()
	memberRoll.Spec.Members = []string{appNamespace}

	cl, _, r := createClientAndReconciler(t, member, memberRoll)

	assertReconcileSucceeds(r, t)

	updatedMemberRoll := test.GetUpdatedObject(cl, memberRoll.ObjectMeta, &maistra.ServiceMeshMemberRoll{}).(*maistra.ServiceMeshMemberRoll)
	assert.StringArrayEmpty(updatedMemberRoll.Spec.Members, "Expected members list in SMMR to be empty", t)

	updatedMember := test.GetUpdatedObject(cl, member.ObjectMeta, &maistra.ServiceMeshMember{}).(*maistra.ServiceMeshMember)
	assert.StringArrayEmpty(updatedMember.Finalizers, "Expected finalizers list in SMM to be empty", t)
}

func TestReconcilePreservesOtherNamespacesInMembersRollWhenAddingMember(t *testing.T) {
	member := newMember()
	memberRoll := newMemberRoll()
	memberRoll.Spec.Members = []string{"other-ns-1", "other-ns-2"}

	cl, _, r := createClientAndReconciler(t, member, memberRoll)

	assertReconcileSucceeds(r, t)

	updatedMemberRoll := test.GetUpdatedObject(cl, memberRoll.ObjectMeta, &maistra.ServiceMeshMemberRoll{}).(*maistra.ServiceMeshMemberRoll)
	assert.DeepEquals(updatedMemberRoll.Spec.Members, []string{"other-ns-1", "other-ns-2", appNamespace}, "Unexpected members in SMMR", t)
}

func TestReconcilePreservesOtherNamespacesInMembersRollWhenRemovingMember(t *testing.T) {
	member := newMember()
	member.DeletionTimestamp = &oneMinuteAgo
	memberRoll := newMemberRoll()
	memberRoll.Spec.Members = []string{"other-ns-1", appNamespace, "other-ns-2"}

	cl, _, r := createClientAndReconciler(t, member, memberRoll)

	assertReconcileSucceeds(r, t)

	updatedMemberRoll := test.GetUpdatedObject(cl, memberRoll.ObjectMeta, &maistra.ServiceMeshMemberRoll{}).(*maistra.ServiceMeshMemberRoll)
	assert.DeepEquals(updatedMemberRoll.Spec.Members, []string{"other-ns-1", "other-ns-2"}, "Unexpected members in SMMR", t)
}

func TestReconcileDeletesMemberRollIfItHadCreatedIt(t *testing.T) {
	member := newMember()
	member.DeletionTimestamp = &oneMinuteAgo
	memberRoll := newMemberRoll()
	memberRoll.Annotations = map[string]string{
		common.CreatedByKey: controllerName,
	}
	memberRoll.Spec.Members = []string{appNamespace}

	cl, _, r := createClientAndReconciler(t, member, memberRoll)

	assertReconcileSucceeds(r, t)
	test.AssertNotFound(cl, types.NamespacedName{controlPlaneNamespace, common.MemberRollName}, &maistra.ServiceMeshMemberRoll{}, "Expected reconcile to delete the SMMR, but it didn't", t)
}

func TestReconcilePreservesMemberRollIfItCreatedItButUserManuallyAddedAnotherNamespace(t *testing.T) {
	member := newMember()
	member.DeletionTimestamp = &oneMinuteAgo
	memberRoll := newMemberRoll()
	memberRoll.Annotations = map[string]string{
		common.CreatedByKey: controllerName,
	}
	memberRoll.Spec.Members = []string{appNamespace, "other-ns-1"}

	cl, _, r := createClientAndReconciler(t, member, memberRoll)

	assertReconcileSucceeds(r, t)

	updatedMemberRoll := test.GetUpdatedObject(cl, memberRoll.ObjectMeta, &maistra.ServiceMeshMemberRoll{}).(*maistra.ServiceMeshMemberRoll)
	assert.DeepEquals(updatedMemberRoll.Spec.Members, []string{"other-ns-1"}, "Unexpected members in SMMR", t)
}

func TestReconcileWorksIfMembersRollDoesNotExistWhenRemovingMember(t *testing.T) {
	member := newMember()
	member.DeletionTimestamp = &oneMinuteAgo

	_, _, r := createClientAndReconciler(t, member)

	assertReconcileSucceeds(r, t)
}

func TestReconcileWorksIfMembersRollIsDeletedExternallyWhenRemovingMember(t *testing.T) {
	member := newMember()
	member.DeletionTimestamp = &oneMinuteAgo
	memberRoll := newMemberRoll()
	memberRoll.Annotations = map[string]string{
		common.CreatedByKey: controllerName,
	}
	memberRoll.Spec.Members = []string{appNamespace}

	_, tracker, r := createClientAndReconciler(t, member, memberRoll)
	tracker.AddReactor(func(action clienttesting.Action) (handled bool, err error) {
		if action.Matches("delete", "servicemeshmemberrolls") {
			return true, apierrors.NewNotFound(schema.GroupResource{
				Group:    maistra.APIGroup,
				Resource: "ServiceMeshMemberRoll",
			}, memberRollName)
		}
		return false, nil
	})

	assertReconcileSucceeds(r, t)
}

func TestReconcileDoesNothingIfMemberIsDeletedAndHasNoFinalizers(t *testing.T) {
	member := newMember()
	member.DeletionTimestamp = &oneMinuteAgo
	member.Finalizers = nil

	_, tracker, r := createClientAndReconciler(t, member)

	assertReconcileSucceeds(r, t)
	test.AssertNumberOfWriteActions(t, tracker.Actions(), 0)
}

func TestReconcileDoesNothingWhenMemberIsNotFound(t *testing.T) {
	_, tracker, r := createClientAndReconciler(t)
	assertReconcileSucceeds(r, t)
	test.AssertNumberOfWriteActions(t, tracker.Actions(), 0)
}

func TestReconcileReturnsErrorIfItCanNotReadMember(t *testing.T) {
	_, tracker, r := createClientAndReconciler(t)
	tracker.AddReactor(test.ClientFailsOn("get", "servicemeshmembers"))
	assertReconcileFails(r, t)
}

func TestReconcileReturnsErrorIfClientOperationFails(t *testing.T) {
	cases := []struct {
		name                      string
		memberRollExists          bool
		memberRollCreatedManually bool
		deletion                  bool
		reactor                   test.ReactFunc
	}{
		{
			name:    "get-member-fails",
			reactor: test.ClientFailsOn("get", "servicemeshmembers"),
		},
		{
			name:    "update-member-fails",
			reactor: test.ClientFailsOn("update", "servicemeshmembers"),
		},
		{
			name:    "get-member-roll-fails",
			reactor: test.ClientFailsOn("get", "servicemeshmemberrolls"),
		},
		{
			name:    "create-member-roll-fails",
			reactor: test.ClientFailsOn("create", "servicemeshmemberrolls"),
		},
		{
			name:             "update-member-roll-fails",
			memberRollExists: true,
			reactor:          test.ClientFailsOn("update", "servicemeshmemberrolls"),
		},
		{
			name:     "get-member-roll-fails-during-delete",
			deletion: true,
			reactor:  test.ClientFailsOn("get", "servicemeshmemberrolls"),
		},
		{
			name:                      "update-member-roll-fails-during-delete",
			memberRollExists:          true,
			memberRollCreatedManually: true,
			deletion:                  true,
			reactor:                   test.ClientFailsOn("update", "servicemeshmemberrolls"),
		},
		{
			name:             "delete-member-roll-fails-during-delete",
			memberRollExists: true,
			deletion:         true,
			reactor:          test.ClientFailsOn("delete", "servicemeshmemberrolls"),
		},
		{
			name:             "update-member-fails-during-delete",
			memberRollExists: true,
			deletion:         true,
			reactor:          test.ClientFailsOn("update", "servicemeshmembers"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var objects []runtime.Object

			member := newMember()
			if tc.deletion {
				member.DeletionTimestamp = &oneMinuteAgo
			}
			objects = append(objects, member)

			if tc.memberRollExists {
				memberRoll := newMemberRoll()
				if !tc.memberRollCreatedManually {
					memberRoll.Annotations = map[string]string{
						common.CreatedByKey: controllerName,
					}
				}
				if tc.deletion {
					memberRoll.Spec.Members = []string{appNamespace}
				}
				objects = append(objects, memberRoll)
			}

			_, tracker, r := createClientAndReconciler(t, objects...)
			tracker.AddReactor(tc.reactor)

			assertReconcileFails(r, t)
		})
	}
}

func newMember() *maistra.ServiceMeshMember {
	return &maistra.ServiceMeshMember{
		ObjectMeta: meta.ObjectMeta{
			Name:       memberName,
			Namespace:  appNamespace,
			Finalizers: []string{common.FinalizerName},
		},
		Spec: maistra.ServiceMeshMemberSpec{
			ControlPlaneRef: maistra.ServiceMeshControlPlaneRef{
				Name:      controlPlaneName,
				Namespace: controlPlaneNamespace,
			},
		},
	}
}

func newMemberRoll() *maistra.ServiceMeshMemberRoll {
	return &maistra.ServiceMeshMemberRoll{
		ObjectMeta: meta.ObjectMeta{
			Name:      memberRollName,
			Namespace: controlPlaneNamespace,
		},
		Spec: maistra.ServiceMeshMemberRollSpec{
			Members: []string{},
		},
	}
}

func createClientAndReconciler(t *testing.T, clientObjects ...runtime.Object) (client.Client, *test.EnhancedTracker, *MemberReconciler) {
	s := scheme.Scheme
	if err := apis.AddToScheme(s); err != nil { // scheme must be initialized before creating the client below
		t.Fatalf("Could not add to scheme: %v", err)
	}

	tracker := clienttesting.NewObjectTracker(scheme.Scheme, scheme.Codecs.UniversalDecoder())
	enhancedTracker := test.NewEnhancedTracker(tracker)
	cl := fake.NewFakeClientWithSchemeAndTracker(scheme.Scheme, &enhancedTracker, clientObjects...)
	fakeEventRecorder := &record.FakeRecorder{}

	r := &MemberReconciler{ResourceManager: common.NewResourceManager(cl, s, log, ""), eventRecorder: fakeEventRecorder}
	return cl, &enhancedTracker, r
}

func assertReconcileSucceeds(r *MemberReconciler, t *testing.T) {
	res, err := r.Reconcile(request)
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}
	if res.Requeue {
		t.Error("Reconcile requeued the request, but it shouldn't have")
	}
}

func assertReconcileFails(r *MemberReconciler, t *testing.T) {
	_, err := r.Reconcile(request)
	if err == nil {
		t.Fatal("Expected reconcile to fail, but it didn't")
	}
}
