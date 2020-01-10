package controlplane

import (
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"regexp"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ControlPlaneReconciler) patchHtpasswdSecret(secret *corev1.Secret) error {
	var rawPassword, auth string

	htSecret := &corev1.Secret{}
	err := r.Client.Get(context.TODO(), client.ObjectKey{Namespace: secret.GetNamespace(), Name: "htpasswd"}, htSecret)
	if err == nil {
		rawPassword = string(htSecret.Data["rawPassword"])
		auth = string(htSecret.Data["auth"])
	} else {
		r.Log.Info("Creating HTPasswd entry", secret.GetObjectKind(), secret.GetName())

		rawPassword, err = generatePassword(255)
		if err != nil {
			r.Log.Error(err, "failed to generate the HTPasswd password")
			return err
		}
		h := sha1.New()
		h.Write([]byte(rawPassword))
		auth = "internal:{SHA}" + base64.StdEncoding.EncodeToString(h.Sum(nil))
	}

	b64Password := base64.StdEncoding.EncodeToString([]byte(rawPassword))
	b64Auth := base64.StdEncoding.EncodeToString([]byte(auth))

	// We store the raw password in order to be able to retrieve it below, when patching Grafana ConfigMap
	secret.Data["rawPassword"] = []byte(b64Password)
	secret.Data["auth"] = []byte(b64Auth)

	return nil
}

func (r *ControlPlaneReconciler) getRawHtPasswd(namespace string) (string, error) {
	htSecret := &corev1.Secret{}
	err := r.Client.Get(context.TODO(), client.ObjectKey{Namespace: namespace, Name: "htpasswd"}, htSecret)
	if err != nil {
		r.Log.Error(err, "error retrieving htpasswd Secret")
		return "", err
	}

	return string(htSecret.Data["rawPassword"]), nil
}

func (r *ControlPlaneReconciler) patchGrafanaConfig(object *corev1.ConfigMap) error {
	dsYaml, found := object.Data["datasources.yaml"]
	if !found {
		r.Log.Info("skipping configuration of Grafana-Prometheus link: Could not find/retrieve datasources.yaml from Grafana ConfigMap")
		return nil
	}

	r.Log.Info("patching Grafana-Prometheus link", object.GetObjectKind(), object.GetName())

	rawPassword, err := r.getRawHtPasswd(object.GetNamespace())
	if err != nil {
		return err
	}

	var re = regexp.MustCompile("(?s)(basicAuthPassword:).*?\n")
	dsYaml = re.ReplaceAllString(dsYaml, fmt.Sprintf("${1} %s\n", rawPassword))
	object.Data["datasources.yaml"] = dsYaml

	return nil
}

func generatePassword(n int) (string, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(b), nil
}
