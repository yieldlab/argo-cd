package e2e

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-cd/common"
	"github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
)

func TestKustomize2AppSource(t *testing.T) {
	fixture.EnsureCleanState()

	appName := "app-" + strconv.FormatInt(time.Now().Unix(), 10)
	_, err := fixture.RunCli("app", "create",
		"--name", appName,
		"--repo", fixture.RepoURL(),
		"--path", guestbookPath,
		"--dest-server", common.KubernetesInternalAPIServerAddr,
		"--dest-namespace", fixture.DeploymentNamespace,
		"--nameprefix", "k2-")
	assert.NoError(t, err)

	_, err = fixture.RunCli("app", "get", appName, "--hard-refresh")
	assert.NoError(t, err)

	_, err = fixture.RunCli("app", "patch", appName, "--patch",
		`[
			{
				"op": "replace",
				"path": "/spec/source/kustomize/namePrefix",
				"value": "k2-patched-"
			},
			{
				"op": "add",
				"path": "/spec/source/kustomize/commonLabels",
				"value": {
					"patched-by": "argo-cd"
				}
			}
		]`,
	)
	assert.NoError(t, err)

	_, err = fixture.RunCli("app", "sync", appName)
	assert.NoError(t, err)

	WaitUntil(t, func() (done bool, err error) {
		app, err := fixture.AppClientset.ArgoprojV1alpha1().Applications(fixture.ArgoCDNamespace).Get(appName, metav1.GetOptions{})
		return err == nil && app.Status.Sync.Status == v1alpha1.SyncStatusCodeSynced, err
	})

	labelValue, err := Run(
		"", "kubectl", "-n="+fixture.DeploymentNamespace,
		"get", "svc,deploy", "k2-patched-guestbook-ui",
		"-ojsonpath={range .items[*]}{.metadata.labels.patched-by}{\" \"}{end}",
	)
	assert.NoError(t, err)
	assert.Equal(t, "argo-cd argo-cd", labelValue)
}
