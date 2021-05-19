package controller

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/golang/glog"
	v1 "k8s.io/api/admission/v1"
	apps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//ServerHandler listen to admission requests and serve responses
type ServerHandler struct {
}

//Serve receives requests and responds accordingly
func (gs *ServerHandler) Serve(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}
	if len(body) == 0 {
		glog.Error("empty body")
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}

	if r.URL.Path != "/validate" {
		glog.Error("no validate")
		http.Error(w, "no validate", http.StatusBadRequest)
		return
	}

	arRequest := v1.AdmissionReview{}
	if err := json.Unmarshal(body, &arRequest); err != nil {
		glog.Error("incorrect body")
		http.Error(w, "incorrect body", http.StatusBadRequest)
	}

	raw := arRequest.Request.Object.Raw

	dep := apps.Deployment{}
	if err := json.Unmarshal(raw, &dep); err != nil {
		glog.Error("error deserializing deployment")
		return
	}

	optinFound, optInValue := getLabelValue(dep.GetLabels(), "scaler/opt-in")

	if !optinFound {
		return
	}

	if !optInValue {
		glog.Infof("Optin label for deployment %s is false - Ignoring", dep.Name)
		return
	}

	if dep.Status.AvailableReplicas == *dep.Spec.Replicas {
		glog.Infof("No replica update for deployment %s - Ignoring", dep.Name)
		return
	}

	glog.Infof("Attempt to adjust replicas from %v to %v for deployment %s - Blocking", dep.Status.AvailableReplicas, *dep.Spec.Replicas, dep.Name)

	arResponse := v1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AdmissionReview",
			APIVersion: "admission.k8s.io/v1",
		},
		Response: &v1.AdmissionResponse{
			UID:     arRequest.Request.UID,
			Allowed: false,
			Result:  &metav1.Status{Message: "The application is currently being managed by the operator - To take back control change the optIn label"},
		},
	}

	resp, err := json.Marshal(arResponse)
	if err != nil {
		glog.Errorf("Can't encode response: %v", err)
		http.Error(w, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
	}

	if _, err := w.Write(resp); err != nil {
		glog.Errorf("Can't write response: %v", err)
		http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
	}
}

func getLabelValue(labels map[string]string, optin string) (bool, bool) {
	var detect bool
	if v, found := labels[optin]; found {
		detect, _ = strconv.ParseBool(v)
		if detect {
			return true, true
		}
		return true, false
	}
	return false, false
}
