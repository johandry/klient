package kubectl

import (
	"io/ioutil"
	"os"
	"testing"

	"log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var testData map[string][]byte

func init() {
	testData = make(map[string][]byte)
	for _, f := range []string{"cm.yaml"} {
		content, err := ioutil.ReadFile("./testdata/" + f)
		if err != nil {
			log.Fatalf("cannot load test data from file %q", f)
		}
		// name := strings.Split(f,".")[0]
		testData[f] = content
	}
}

func TestClient_Apply_CreateAndDelete(t *testing.T) {
	envContext := os.Getenv(contextEnvVarName)
	envKubeconfig := os.Getenv(kubeconfigEnvVarName)

	tests := []struct {
		name       string
		content    []byte
		context    string
		kubeconfig string
		wantErr    bool
	}{
		{"apply configMap", testData["cm.yaml"], envContext, envKubeconfig, false},
		{"apply configMapList", testData["cml.yaml"], envContext, envKubeconfig, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewClientE(tt.context, tt.kubeconfig)
			if err != nil {
				t.Fatalf("failed to create the client with context %q and kubeconfig %q", tt.context, tt.kubeconfig)
			}
			if err := c.Apply(tt.content); (err != nil) != tt.wantErr {
				t.Errorf("Client.Apply() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_Apply_PatchAndDelete(t *testing.T) {
	envContext := os.Getenv(contextEnvVarName)
	envKubeconfig := os.Getenv(kubeconfigEnvVarName)

	tests := []struct {
		name           string
		initial        []byte
		modified       []byte
		getChange      func(*Client) (string, error)
		want           string
		context        string
		kubeconfig     string
		wantInitialErr bool
		wantPatchErr   bool
		wantGetErr     bool
	}{
		{
			"apply & patch configMap testApplyPatch001",
			[]byte(`{"apiVersion": "v1", "kind": "ConfigMap", "metadata": { "name": "testapplypatch001" }, "data": {	"key1": "apple" } }`),
			[]byte(`{"apiVersion": "v1", "kind": "ConfigMap", "metadata": { "name": "testapplypatch001" }, "data": {	"key1": "orange" } }`),
			func(c *Client) (string, error) {
				cm, err := c.Clientset.CoreV1().ConfigMaps("default").Get("testapplypatch001", metav1.GetOptions{})
				if err != nil {
					return "", err
				}

				return "key1: " + cm.Data["key1"], nil
			},
			"key1: orange",
			envContext, envKubeconfig,
			false, false, false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewClientE(tt.context, tt.kubeconfig)
			if err != nil {
				t.Fatalf("failed to create the client with context %q and kubeconfig %q", tt.context, tt.kubeconfig)
			}
			if err := c.Apply(tt.initial); (err != nil) != tt.wantInitialErr {
				t.Errorf("Client.Apply() error = %v, wantErr %v", err, tt.wantInitialErr)
				return
			}
			defer func() {
				if err := c.Delete(tt.initial); err != nil {
					t.Errorf("Client.Delete() error = %v", err)
				}
			}()

			if err := c.Apply(tt.modified); (err != nil) != tt.wantPatchErr {
				t.Errorf("Client.Apply() error = %v, wantErr %v", err, tt.wantPatchErr)
				return
			}

			got, err := tt.getChange(c)
			if (err != nil) != tt.wantGetErr {
				t.Errorf("Patch test check failed. error = %v, wantErr %v", err, tt.wantGetErr)
				return
			}
			if got != tt.want {
				t.Errorf("Client.Apply() = %v, want %v ", got, tt.want)
			}
		})
	}
}
