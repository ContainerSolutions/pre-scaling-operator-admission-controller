package controller

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/golang/glog"
	ocapps "github.com/openshift/api/apps/v1"
	v1 "k8s.io/api/admission/v1"
	apps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//ServerHandler listen to admission requests and serve responses
type ServerHandler struct {
}

type objectInfo struct {
	Namespace         string
	Name              string
	Labels            map[string]string
	SpecReplica       int32
	AvailableReplicas int32
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

	newobject := getObjectData(arRequest.Request.Kind.Kind, raw)

	optinFound, optInValue := getLabelValue(newobject.Labels, "scaler/opt-in")

	if !optinFound {
		return
	}

	if !optInValue {
		glog.Infof("Optin label for deployment %s is false - Ignoring", newobject.Name)
		return
	}

	if newobject.AvailableReplicas == newobject.SpecReplica {
		glog.Infof("No replica update for deployment %s - Ignoring", newobject.Name)
		return
	}

	glog.Infof("Attempt to adjust replicas from %v to %v for deployment %s - Blocking", newobject.AvailableReplicas, newobject.SpecReplica, newobject.Name)

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

func getObjectData(kind string, raw []byte) objectInfo {

	var newobject objectInfo

	if kind == "Deployment" {
		dep := apps.Deployment{}
		if err := json.Unmarshal(raw, &dep); err != nil {
			glog.Error("error deserializing deployment")
			return objectInfo{}
		}

		newobject = objectInfo{
			Namespace:         dep.Namespace,
			Name:              dep.Name,
			Labels:            dep.GetLabels(),
			SpecReplica:       *dep.Spec.Replicas,
			AvailableReplicas: dep.Status.AvailableReplicas,
		}
	} else if kind == "DeploymentConfig" {
		dc := ocapps.DeploymentConfig{}
		if err := json.Unmarshal(raw, &dc); err != nil {
			glog.Error("error deserializing deployment")
			return objectInfo{}
		}

		newobject = objectInfo{
			Namespace:         dc.Namespace,
			Name:              dc.Name,
			Labels:            dc.GetLabels(),
			SpecReplica:       dc.Spec.Replicas,
			AvailableReplicas: dc.Status.AvailableReplicas,
		}
	}

	return newobject
}
