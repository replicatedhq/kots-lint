## Linting the example locally

First, make sure skaffold is running by running this in the root directory
```shell
$ skaffold dev
```
Then, also from the root directory, lint the example with
```shell
$ tar cvf - example/files-to-lint | curl -XPOST --data-binary @- http://localhost:30082/v1/lint
```
