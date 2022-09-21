# showksec - Show Kubernetes secrets

Pipe a Kubernetes secret YAML into this file and it will base64-decode its
secrets.

## Usage

```console
$ cat test-secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: test-secret
data:
  foo: YmFy

$ cat test-secret.yaml | showksec
apiVersion: v1
kind: Secret
metadata:
  name: test-secret
stringData:
  foo: bar
```
