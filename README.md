# jobgraphs
Kubernetes controller for running jobs specified in a graph definition

Example:
```yaml
apiVersion: ced.io/v1alpha1
kind: JobGraph
metadata:
  name: test-graph
spec:
  graph:
    edges:
      - source: job1
        target: job2
  jobTemplates:
    - metadata:
        name: job1
      spec:
        template:
          spec:
            containers:
              - image: ubuntu:20.04
                command: ["echo","hello world"]
            restartPolicy: Never
    - metadata:
        name: job2
      spec:
        template:
          spec:
            containers:
              - image: ubuntu:20.04
                command: ["echo","hello world"]
            restartPolicy: Never
```
