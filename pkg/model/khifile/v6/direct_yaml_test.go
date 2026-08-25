// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package khifilev6

import (
	"bytes"
	"testing"
	"unicode/utf8"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
)

func TestDirectYAMLSerializer_SerializeStruct(t *testing.T) {
	testCases := []struct {
		name string
		yaml string
	}{
		{
			name: "simple key value map",
			yaml: `
foo: value1
bar: 123
`,
		},
		{
			name: "nested map and list of scalars",
			yaml: `
map:
  key: true
list:
  - null
  - hello
  - 1234
`,
		},
		{
			name: "map inside list simple",
			yaml: `
list:
  - a: 1
    b: 2
`,
		},
		{
			name: "string quoting edge cases",
			yaml: `
str_bool_true: "true"
str_bool_false: "false"
str_bool_yes: "yes"
str_bool_no: "no"
str_number_int: "12345"
str_number_hex: "0x10"
str_number_oct: "0777"
str_number_float: "1.23e+10"
str_colon: "key: value"
str_null: "null"
str_whitespace: " spaced "
str_empty: ""
str_brackets: "[item1, item2]"
str_braces: "{a: 1}"
str_special_chars: "foo#bar=baz&qux%quux"
str_japanese: "設定マップの更新"
str_unicode_symbols: "🚀 container-started (node-1) ⚡"
`,
		},
		{
			name: "float and timestamp",
			yaml: `
float_val: 1.23
time_val: 2026-04-20T03:00:00Z
`,
		},
		{
			name: "empty maps and lists",
			yaml: `
empty_map: {}
empty_list: []
nested:
  empty_sub_map: {}
  empty_sub_list: []
`,
		},
		{
			name: "projected volume sources with nested maps",
			yaml: `
defaultMode: 420
sources:
  - serviceAccountToken:
      expirationSeconds: 3607
      path: token
  - configMap:
      items:
        - key: ca.crt
          path: ca.crt
      name: kube-root-ca.crt
  - downwardAPI:
      items:
        - fieldRef:
            apiVersion: v1
            fieldPath: metadata.namespace
          path: namespace
`,
		},
		{
			name: "container env with valueFrom and secretKeyRef",
			yaml: `
containers:
  - name: app
    image: "gcr.io/my-project/app:v1.0.0"
    command:
      - /bin/sh
      - -c
      - "echo starting..."
    env:
      - name: POD_NAME
        valueFrom:
          fieldRef:
            fieldPath: metadata.name
      - name: DB_PASSWORD
        valueFrom:
          secretKeyRef:
            name: db-secret
            key: password
      - name: SIMPLE_VAR
        value: plain-text
    ports:
      - containerPort: 8080
        name: http
        protocol: TCP
      - containerPort: 9090
        name: metrics
`,
		},
		{
			name: "affinity, tolerations, and securityContext",
			yaml: `
affinity:
  nodeAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      nodeSelectorTerms:
        - matchExpressions:
            - key: kubernetes.io/os
              operator: In
              values:
                - linux
tolerations:
  - effect: NoSchedule
    key: node-role.kubernetes.io/master
    operator: Exists
securityContext:
  runAsUser: 1000
  runAsNonRoot: true
  fsGroup: 2000
`,
		},
		{
			name: "deployment spec with strategy and selector",
			yaml: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  labels:
    app: nginx
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
        - name: nginx
          image: "nginx:1.14.2"
          ports:
            - containerPort: 80
`,
		},
		{
			name: "deeply nested hierarchy (5 levels)",
			yaml: `
level1:
  - level2:
      - level3:
          - level4:
              - level5:
                  key: leaf_value
`,
		},
		{
			name: "protoPayload with @type and special key characters",
			yaml: `
protoPayload:
  "@type": type.googleapis.com/google.cloud.audit.AuditLog
  serviceName: k8s.io
  methodName: io.k8s.core.v1.pods.create
  resourceName: namespaces/default/pods/nginx
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			node, err := structured.FromYAML(tc.yaml)
			if err != nil {
				t.Fatalf("FromYAML() error = %v", err)
			}

			idGen := NewIDGenerator()
			pool := NewInternPool(idGen)

			internedRef, err := ToInternedStruct(node, pool)
			if err != nil {
				t.Fatalf("ToInternedStruct() error = %v", err)
			}

			// 1. Reference YAML using YAMLNodeSerializer
			yamlSerializer := &structured.YAMLNodeSerializer{}
			origNode, err := FromInternedStruct(internedRef.ToProto(), pool)
			if err != nil {
				t.Fatalf("FromInternedStruct() error = %v", err)
			}
			refBytes, err := yamlSerializer.Serialize(origNode)
			if err != nil {
				t.Fatalf("yamlSerializer.Serialize() error = %v", err)
			}

			// 2. Direct YAML serializer
			directSerializer := NewDirectYAMLSerializer()
			directYAML, err := directSerializer.SerializeStruct(internedRef.ToProto(), pool)
			if err != nil {
				t.Fatalf("directSerializer.SerializeStruct() error = %v", err)
			}

			// 3. Verify semantic equivalence by unmarshaling both to map/any
			var refParsed, directParsed any
			if err := yaml.Unmarshal(refBytes, &refParsed); err != nil {
				t.Fatalf("failed to unmarshal reference YAML:\n%s\nerror: %v", string(refBytes), err)
			}
			if err := yaml.Unmarshal([]byte(directYAML), &directParsed); err != nil {
				t.Fatalf("failed to unmarshal direct YAML:\n%s\nerror: %v", directYAML, err)
			}

			if diff := cmp.Diff(refParsed, directParsed); diff != "" {
				t.Errorf("DirectYAMLSerializer mismatch with reference YAML (-ref +direct):\n%s\nReference YAML:\n%s\nDirect YAML:\n%s", diff, string(refBytes), directYAML)
			}
		})
	}
}

func FuzzDirectYAMLSerializer_StringRoundtrip(f *testing.F) {
	seeds := []string{
		"", "true", "false", "yes", "no", "on", "off", "null", "~", "y", "n",
		"123", "0", "-1", "+1", "0x10", "0o777", "0b1010", "1.23e+10", ".nan", ".inf", "-.inf", "+.inf",
		"key: value", "[1, 2]", "{a: 1}", "# comment", "foo#bar", " spaced ", "a\nb",
		"nginx", "pod-sample", "default", "kubernetes.io/os", "app.kubernetes.io/name",
		"123.456", "0.0", "1e-5", "-0", "+0", "\x01", "\t", "\r\n", "0_", "1_000",
		"Running", "Pending", "Failed", "ClusterIP", "NodePort", "LoadBalancer",
		"v1", "apps/v1", "batch/v1", "gcr.io/google-containers/pause:3.2",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, origStr string) {
		if !utf8.ValidString(origStr) {
			t.Skip()
		}

		var buf bytes.Buffer
		serializer := NewDirectYAMLSerializer()
		buf.WriteString("key: ")
		serializer.emitString(&buf, origStr)
		buf.WriteString("\n")

		yamlBytes := buf.Bytes()

		var parsed map[string]any
		if err := yaml.Unmarshal(yamlBytes, &parsed); err != nil {
			t.Fatalf("failed to unmarshal emitted YAML for input %q:\nYAML:\n%s\nerror: %v", origStr, string(yamlBytes), err)
		}

		val, ok := parsed["key"]
		if !ok {
			t.Fatalf("missing 'key' in parsed YAML for input %q:\nYAML:\n%s", origStr, string(yamlBytes))
		}

		strVal, ok := val.(string)
		if !ok {
			t.Fatalf("value type converted from string to %T (%v) for input %q:\nYAML:\n%s", val, val, origStr, string(yamlBytes))
		}

		if strVal != origStr {
			t.Fatalf("value mismatch for input %q: got %q\nYAML:\n%s", origStr, strVal, string(yamlBytes))
		}
	})
}

func BenchmarkDirectYAMLSerializer_SerializeStruct(b *testing.B) {
	sampleYAML := `metadata:
  name: pod-sample
  namespace: default
  labels:
    app: nginx
    env: prod
spec:
  nodeName: node-1
  restartPolicy: Always
  replicas: 3
  containers:
    - name: nginx
      image: "nginx:latest"
      port: 80
`
	node, err := structured.FromYAML(sampleYAML)
	if err != nil {
		b.Fatalf("FromYAML() error = %v", err)
	}

	idGen := NewIDGenerator()
	pool := NewInternPool(idGen)
	internedRef, err := ToInternedStruct(node, pool)
	if err != nil {
		b.Fatalf("ToInternedStruct() error = %v", err)
	}

	directSerializer := NewDirectYAMLSerializer()

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = directSerializer.SerializeStruct(internedRef.ToProto(), pool)
	}
}
