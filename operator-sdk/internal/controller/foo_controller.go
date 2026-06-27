/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"k8s.io/client-go/tools/record"

	"github.com/go-logr/logr"

	ctrl "sigs.k8s.io/controller-runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	samplecontrollerv1alpha1 "github.com/takuteh/Foo_Operator/operator-sdk/api/v1alpha1"
)

// FooReconciler reconciles a Foo object
type FooReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=samplecontroller.samplecontroller.example.com,resources=foos,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=samplecontroller.samplecontroller.example.com,resources=foos/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=samplecontroller.samplecontroller.example.com,resources=foos/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Foo object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *FooReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {

	logger := logf.FromContext(ctx)

	//CRのyamlに書かれた情報を格納するための変数fooを定義する
	var foo samplecontrollerv1alpha1.Foo

	//foo(CR)の情報をAPIサーバーから取得してfooに格納
	if err := r.Get(ctx, req.NamespacedName, &foo); err != nil {
		//リソースが存在しない(NotFound)のエラーなら無視する
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	//古いdeployment、ゴミを削除する
	if err := r.cleanupOwnedResources(
		ctx,
		logger,
		&foo,
	); err != nil {
		return ctrl.Result{}, err
	}

	//CRのyamlに書かれた内容をdeploymentのyaml形式に整形する
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      foo.Spec.DeploymentName,
			Namespace: req.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &foo.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": foo.Spec.DeploymentName},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": foo.Spec.DeploymentName},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx",
						},
					},
				},
			},
		},
	}

	// Deploymentの親をFooに設定する
	if err := controllerutil.SetControllerReference(
		&foo,
		dep,
		r.Scheme,
	); err != nil {
		return ctrl.Result{}, err
	}
	// 上記で作ったdeploymentをクラスタに適用（作成or更新）
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, dep, func() error {
		dep.Spec.Replicas = &foo.Spec.Replicas //fooのspec.replicasの値をdeploymentに代入する
		return nil
	})

	if err != nil {
		return ctrl.Result{}, err
	}

	//コントローラーが使うための、fooの中身の情報を取得する
	logger.Info(
		"Foo found",
		"name", foo.Name,
		"deploymentName", foo.Spec.DeploymentName,
		"replicas", foo.Spec.Replicas,
	)
	//deploymentの情報を格納するための変数を定義する
	var deployment appsv1.Deployment

	//Deploymentを検索するためのキーを作成する
	var deploymentNamespacedName = client.ObjectKey{
		Namespace: req.Namespace,
		Name:      foo.Spec.DeploymentName,
	}

	// 上で作ったキーをもとに欲しいdeploymentを抽出してdeployment変数に格納する
	if err := r.Get(ctx, deploymentNamespacedName, &deployment); err != nil {

		logger.Error(err, "unable to fetch Deployment")

		// Deploymentが存在しない場合は無視する
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Deploymentで実際に稼働しているPod数を取得する
	availableReplicas := deployment.Status.AvailableReplicas

	// Foo.statusと同じ値なら更新不要なので終了
	if availableReplicas == foo.Status.AvailableReplicas {
		return ctrl.Result{}, nil
	}

	// Deploymentの状態をFoo.statusへ反映する
	foo.Status.AvailableReplicas = availableReplicas
	foo.Status.LastUpdateTime = metav1.Now()

	// Foo.statusをAPIサーバーへ保存する
	if err := r.Status().Update(ctx, &foo); err != nil {

		logger.Error(err, "unable to update Foo status")

		return ctrl.Result{}, err
	}

	// Foo.statusを更新したことをEventとして記録する
	r.Recorder.Eventf(
		&foo,
		corev1.EventTypeNormal,
		"Updated",
		"Update foo.status.AvailableReplicas: %d",
		foo.Status.AvailableReplicas,
	)
	return ctrl.Result{}, nil
}

var (
	deploymentOwnerKey = ".metadata.controller"                         //下で作るfooが所有するdeploymentを検索するためのインデックスの名前
	apiGVStr           = samplecontrollerv1alpha1.GroupVersion.String() //yamlに書かれたapiVersionの文字列を取得する
)

// cleanupOwnedResources
// Fooが所有しているDeploymentのうち、
// foo.spec.deploymentNameと一致しないものを削除する
func (r *FooReconciler) cleanupOwnedResources(
	ctx context.Context,
	log logr.Logger,
	foo *samplecontrollerv1alpha1.Foo,
) error {

	log.Info("finding existing Deployments for Foo resource")

	// Fooが所有するDeployment一覧を取得
	var deployments appsv1.DeploymentList

	if err := r.List(
		ctx,
		&deployments,
		client.InNamespace(foo.Namespace), //namespace内のdeploymentのうち
		client.MatchingFields(
			map[string]string{
				deploymentOwnerKey: foo.Name, //.metadata.controllerがfooの名前と一致するものを検索する
			},
		),
	); err != nil {
		return err
	}

	// 検索に引っかかったDeploymentを1つずつ確認
	for _, deployment := range deployments.Items {

		// Fooで指定されているDeploymentなら残す
		if deployment.Name == foo.Spec.DeploymentName {
			continue
		}

		// 名前が違うなら古いDeploymentなので削除
		if err := r.Delete(ctx, &deployment); err != nil {
			log.Error(err, "failed to delete Deployment resource")
			return err
		}

		log.Info("delete deployment resource: " + deployment.Name)

		// Kubernetes Eventを記録
		r.Recorder.Eventf(
			foo,
			corev1.EventTypeNormal,
			"Deleted",
			"Deleted deployment %q",
			deployment.Name,
		)
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
// controller起動時に一度だけ呼び出される関数
func (r *FooReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// Fooが所有するDeploymentを後で一発検索するための索引作成するための関数
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&appsv1.Deployment{},
		deploymentOwnerKey,
		func(rawObj client.Object) []string {

			//渡されたobjectをDeployment型に変換する
			deployment := rawObj.(*appsv1.Deployment)

			//渡されたdeploymentのownerReferenceを取得する
			owner := metav1.GetControllerOf(deployment)

			//親がいない(ownerReferenceに何も書いていない)野良のdeploymentは無視する
			if owner == nil {
				return nil
			}
			//親がFooリソースでないものも無視する
			if owner.APIVersion != apiGVStr ||
				owner.Kind != "Foo" {
				return nil
			}

			//この選別を抜けたDeploymentの親はFooリソースなので、親の名前を返す
			//Fooの管理するdeploymetというインデックスに登録する
			return []string{owner.Name}
		},
	); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&samplecontrollerv1alpha1.Foo{}). //このコントローラーはFooリソースの(作成・更新・削除)イベントを監視する
		Owns(&appsv1.Deployment{}).           //Fooリソースが所有するDeploymentリソースの(作成・更新・削除)イベントも監視する//ownsはFooがownerのdeploymentだけという意味
		Named("foo").                         //コントローラーの名前をfooとする
		Complete(r)                           //ここまでの設定を確定して、コントローラーを動作させる
}
